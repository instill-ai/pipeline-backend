package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/oklog/ulid/v2"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// Service interface
type Service interface {
	GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient
	GetRedisClient() *redis.Client

	CreatePipeline(owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	ListPipelines(owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByID(id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUID(uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, owner *mgmtPB.User, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(id string, owner *mgmtPB.User) error
	UpdatePipelineState(id string, owner *mgmtPB.User, state datamodel.PipelineState) (*datamodel.Pipeline, error)
	UpdatePipelineID(id string, owner *mgmtPB.User, newID string) (*datamodel.Pipeline, error)
	TriggerSyncPipeline(ctx context.Context, req *pipelinePB.TriggerSyncPipelineRequest, owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerSyncPipelineResponse, error)
	TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest, pipelineTriggerID string, owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerAsyncPipelineResponse, error)

	ListPipelinesAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)

	IncludeConnectorTypeInRecipeByPermalink(recipe *datamodel.Recipe) error
	IncludeConnectorTypeInRecipeByName(recipe *datamodel.Recipe, owner *mgmtPB.User) error

	// Controller APIs
	GetResourceState(uid uuid.UUID) (*pipelinePB.Pipeline_State, error)
	UpdateResourceState(uid uuid.UUID, state pipelinePB.Pipeline_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error
	// Influx API
	WriteNewDataPoint(p *write.Point)
}

type service struct {
	repository                    repository.Repository
	mgmtPrivateServiceClient      mgmtPB.MgmtPrivateServiceClient
	connectorPublicServiceClient  connectorPB.ConnectorPublicServiceClient
	connectorPrivateServiceClient connectorPB.ConnectorPrivateServiceClient
	controllerClient              controllerPB.ControllerPrivateServiceClient
	redisClient                   *redis.Client
	temporalClient                client.Client
	influxDBWriteClient           api.WriteAPI
}

// NewService initiates a service instance
func NewService(r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	c connectorPB.ConnectorPublicServiceClient,
	cPrivate connectorPB.ConnectorPrivateServiceClient,
	ct controllerPB.ControllerPrivateServiceClient,
	rc *redis.Client,
	t client.Client,
	i api.WriteAPI,
) Service {
	return &service{
		repository:                    r,
		mgmtPrivateServiceClient:      u,
		connectorPublicServiceClient:  c,
		connectorPrivateServiceClient: cPrivate,
		controllerClient:              ct,
		redisClient:                   rc,
		temporalClient:                t,
		influxDBWriteClient:           i,
	}
}

// GetMgmtPrivateServiceClient returns the management private service client
func (h *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return h.mgmtPrivateServiceClient
}

// GetRedisClient returns the redis client
func (h *service) GetRedisClient() *redis.Client {
	return h.redisClient
}

func (s *service) CreatePipeline(owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	// User desires to be active
	dbPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE)

	ownerPermalink := utils.GenOwnerPermalink(owner)
	dbPipeline.Owner = ownerPermalink

	recipePermalink, err := s.recipeNameToPermalink(owner, dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Recipe = recipePermalink

	resourceState, err := s.checkState(dbPipeline.Recipe)
	if err != nil {
		return nil, err
	}

	if err := s.repository.CreatePipeline(dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetPipelineByID(dbPipeline.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	rErr := s.includeResourceDetailInRecipe(dbCreatedPipeline.Recipe)
	if rErr != nil {
		return nil, rErr
	}
	createdCecipeRscName, err := s.recipePermalinkToName(dbCreatedPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// Add resource entry to controller to start checking components' state
	if err := s.UpdateResourceState(dbCreatedPipeline.UID, pipelinePB.Pipeline_State(resourceState), nil); err != nil {
		return nil, err
	}

	dbCreatedPipeline.Recipe = createdCecipeRscName

	return dbCreatedPipeline, nil
}

func (s *service) ListPipelines(owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error) {

	ownerPermalink := utils.GenOwnerPermalink(owner)
	dbPipelines, ps, pt, err := s.repository.ListPipelines(ownerPermalink, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	if !isBasicView {
		for idx := range dbPipelines {
			err := s.includeResourceDetailInRecipe(dbPipelines[idx].Recipe)
			if err != nil {
				return nil, 0, "", err
			}
			recipeRscName, err := s.recipePermalinkToName(dbPipelines[idx].Recipe)
			if err != nil {
				return nil, 0, "", status.Errorf(codes.Internal, err.Error())
			}
			dbPipelines[idx].Recipe = recipeRscName
		}
	}

	return dbPipelines, ps, pt, nil
}

func (s *service) ListPipelinesAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}
	if !isBasicView {
		for idx := range dbPipelines {
			err := s.includeResourceDetailInRecipe(dbPipelines[idx].Recipe)
			if err != nil {
				return nil, 0, "", err
			}
		}
	}

	return dbPipelines, ps, pt, nil
}

func (s *service) GetPipelineByID(id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error) {

	ownerPermalink := utils.GenOwnerPermalink(owner)

	dbPipeline, err := s.repository.GetPipelineByID(id, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	if !isBasicView {
		err := s.includeResourceDetailInRecipe(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
		recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		dbPipeline.Recipe = recipeRscName
	}

	return dbPipeline, nil
}

func (s *service) GetPipelineByUID(uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error) {

	ownerPermalink := utils.GenOwnerPermalink(owner)

	dbPipeline, err := s.repository.GetPipelineByUID(uid, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	if !isBasicView {
		err := s.includeResourceDetailInRecipe(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
		recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		dbPipeline.Recipe = recipeRscName
	}

	return dbPipeline, nil
}

func (s *service) GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(uid, isBasicView)
	if err != nil {
		return nil, err
	}
	if !isBasicView {
		err := s.includeResourceDetailInRecipe(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
	}

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(id string, owner *mgmtPB.User, toUpdPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	resourceState := toUpdPipeline.State

	if toUpdPipeline.Recipe != nil {

		// User desires to be active
		toUpdPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE)

		recipePermalink, err := s.recipeNameToPermalink(owner, toUpdPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		toUpdPipeline.Recipe = recipePermalink

		resourceState, err = s.checkState(toUpdPipeline.Recipe)
		if err != nil {
			return nil, err
		}
	}

	ownerPermalink := utils.GenOwnerPermalink(owner)

	toUpdPipeline.Owner = ownerPermalink

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetPipelineByID(id, ownerPermalink, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	if err := s.repository.UpdatePipeline(id, ownerPermalink, toUpdPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetPipelineByID(toUpdPipeline.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	// Update resource entry in controller to start checking components' state
	if err := s.UpdateResourceState(dbPipeline.UID, pipelinePB.Pipeline_State(resourceState), nil); err != nil {
		return nil, err
	}

	rErr := s.includeResourceDetailInRecipe(dbPipeline.Recipe)
	if rErr != nil {
		return nil, rErr
	}

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	dbPipeline.Recipe = recipeRscName

	return dbPipeline, nil
}

func (s *service) DeletePipeline(id string, owner *mgmtPB.User) error {
	ownerPermalink := utils.GenOwnerPermalink(owner)

	dbPipeline, err := s.repository.GetPipelineByID(id, ownerPermalink, false)
	if err != nil {
		return err
	}

	if err := s.DeleteResourceState(dbPipeline.UID); err != nil {
		return err
	}

	return s.repository.DeletePipeline(id, ownerPermalink)
}

func (s *service) UpdatePipelineState(id string, owner *mgmtPB.User, state datamodel.PipelineState) (*datamodel.Pipeline, error) {

	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED) {
		return nil, status.Errorf(codes.InvalidArgument, "State update with unspecified is not allowed")
	}

	ownerPermalink := utils.GenOwnerPermalink(owner)

	dbPipeline, err := s.repository.GetPipelineByID(id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	// user desires to be active or inactive, state stay the same
	// but update etcd storage with checkState
	var resourceState datamodel.PipelineState
	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		resourceState, err = s.checkState(dbPipeline.Recipe)
		if err != nil {
			return nil, err
		}
	} else {
		resourceState = datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE)
	}

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		mode, recipeErr := s.checkRecipe(owner, recipeRscName)

		if recipeErr != nil {
			return nil, recipeErr
		}

		if err := s.repository.UpdatePipelineMode(id, ownerPermalink, mode); err != nil {
			return nil, err
		}

	} else {
		if err := s.repository.UpdatePipelineMode(id, ownerPermalink, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED)); err != nil {
			return nil, err
		}
	}

	if err := s.repository.UpdatePipelineState(id, ownerPermalink, state); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetPipelineByID(id, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	// Update resource entry in controller to start checking components' state
	if err := s.UpdateResourceState(dbPipeline.UID, pipelinePB.Pipeline_State(resourceState), nil); err != nil {
		return nil, err
	}

	return s.GetPipelineByID(dbPipeline.ID, owner, true)

}

func (s *service) UpdatePipelineID(id string, owner *mgmtPB.User, newID string) (*datamodel.Pipeline, error) {

	ownerPermalink := utils.GenOwnerPermalink(owner)

	// Validation: Pipeline existence
	existingPipeline, _ := s.repository.GetPipelineByID(id, ownerPermalink, true)
	if existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	// if err := s.DeleteResourceState(existingPipeline.UID.String()); err != nil {
	// 	return nil, err
	// }

	if err := s.repository.UpdatePipelineID(id, ownerPermalink, newID); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetPipelineByID(newID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	// if err := s.UpdateResourceState(dbPipeline.UID.String(), pipelinePB.Pipeline_State(existingPipeline.State), nil); err != nil {
	// 	return nil, err
	// }

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbPipeline.Recipe = recipeRscName

	return dbPipeline, nil
}

func (s *service) preTriggerPipeline(dbPipeline *datamodel.Pipeline, pipelineInputs []*pipelinePB.PipelineDataPayload, expectedMode datamodel.PipelineMode) error {
	state, err := s.GetResourceState(dbPipeline.UID)
	if err != nil {
		return err
	}
	if *state != pipelinePB.Pipeline_STATE_ACTIVE {
		return status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	if dbPipeline.Mode != expectedMode {
		return status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not sync", dbPipeline.ID))
	}

	for idx := range pipelineInputs {
		pipelineInputs[idx].DataMappingIndex = ulid.Make().String()
	}
	return nil
}

func (s *service) TriggerSyncPipeline(ctx context.Context, req *pipelinePB.TriggerSyncPipelineRequest, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerSyncPipelineResponse, error) {

	logger, _ := logger.GetZapLogger(ctx)
	err := s.preTriggerPipeline(dbPipeline, req.Inputs, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC))
	if err != nil {
		return nil, err
	}

	pipelineInputs := req.Inputs
	// Download images
	var images [][]byte
	for idx := range pipelineInputs {
		for imageIdx := range pipelineInputs[idx].Images {
			switch pipelineInputs[idx].Images[imageIdx].UnstructuredData.(type) {
			case *pipelinePB.PipelineDataPayload_UnstructuredData_Blob:
				images = append(images, pipelineInputs[idx].Images[imageIdx].GetBlob())
			case *pipelinePB.PipelineDataPayload_UnstructuredData_Url:
				imageUrl := pipelineInputs[idx].Images[imageIdx].GetUrl()
				response, err := http.Get(imageUrl)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to download image at %v. %v", imageUrl, err))

					return nil, fmt.Errorf("unable to download image at %v", imageUrl)
				}
				defer response.Body.Close()

				buff := new(bytes.Buffer) // pointer
				_, err = buff.ReadFrom(response.Body)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to read content body from image at %v. %v", imageUrl, err))

					return nil, fmt.Errorf("unable to read content body from image at %v", imageUrl)
				}
				images = append(images, buff.Bytes())
			}
		}
	}

	var inputs []*connectorPB.DataPayload
	for idx := range pipelineInputs {
		inputs = append(inputs, &connectorPB.DataPayload{
			DataMappingIndex: pipelineInputs[idx].DataMappingIndex,
			Images:           images,
			Texts:            pipelineInputs[idx].Texts,
			StructuredData:   pipelineInputs[idx].StructuredData,
			Metadata:         pipelineInputs[idx].Metadata,
		})
	}

	componentIdMap := make(map[string]*datamodel.Component)

	for idx := range dbPipeline.Recipe.Components {
		componentIdMap[dbPipeline.Recipe.Components[idx].Id] = dbPipeline.Recipe.Components[idx]
	}

	dag := utils.NewDAG(dbPipeline.Recipe.Components)
	for _, component := range dbPipeline.Recipe.Components {
		parents, _, err := utils.ParseDependency(component.Dependencies)
		if err != nil {
			return nil,
				status.Errorf(codes.InvalidArgument, "dependencies error")
		}
		for idx := range parents {
			dag.AddEdge(componentIdMap[parents[idx]], component)
		}
	}
	orderedComp, err := dag.TopoloicalSort()
	if err != nil {
		return nil, err
	}

	cache := map[string][]*connectorPB.DataPayload{}
	cache[orderedComp[0].Id] = inputs

	for _, comp := range orderedComp[1:] {
		_, depMap, err := utils.ParseDependency(comp.Dependencies)
		if err != nil {
			return nil, err
		}
		inputs := worker.MergeData(cache, depMap, len(req.Inputs))
		resp, err := s.connectorPublicServiceClient.ExecuteConnector(
			utils.InjectOwnerToContextWithOwnerPermalink(ctx, utils.GenOwnerPermalink(owner)),
			&connectorPB.ExecuteConnectorRequest{
				Name:   comp.ResourceName,
				Inputs: inputs,
			},
		)
		if err != nil {
			return nil, err
		}
		cache[comp.Id] = resp.Outputs
	}

	outputs := cache[orderedComp[len(orderedComp)-1].Id]
	pipelineOutputs := []*pipelinePB.PipelineDataPayload{}
	for idx := range outputs {
		images := []*pipelinePB.PipelineDataPayload_UnstructuredData{}
		for imageIdx := range outputs[idx].Images {
			images = append(images, &pipelinePB.PipelineDataPayload_UnstructuredData{
				UnstructuredData: &pipelinePB.PipelineDataPayload_UnstructuredData_Blob{
					Blob: outputs[idx].Images[imageIdx],
				},
			})
		}
		pipelineOutput := &pipelinePB.PipelineDataPayload{
			DataMappingIndex: outputs[idx].DataMappingIndex,
			Images:           images,
			Texts:            outputs[idx].Texts,
			StructuredData:   outputs[idx].StructuredData,
			Metadata:         outputs[idx].Metadata,
		}
		pipelineOutputs = append(pipelineOutputs, pipelineOutput)
	}

	return &pipelinePB.TriggerSyncPipelineResponse{
		Outputs: pipelineOutputs,
	}, nil
}

func (s *service) TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest, pipelineTriggerID string, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerAsyncPipelineResponse, error) {

	inputs := req.Inputs
	err := s.preTriggerPipeline(dbPipeline, inputs, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC))
	if err != nil {
		return nil, err
	}
	logger, _ := logger.GetZapLogger(ctx)

	if err := s.excludeResourceDetailFromRecipe(dbPipeline.Recipe); err != nil {
		return nil, err
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	inputBlobRedisKeys := []string{}
	for idx, input := range inputs {
		inputJson, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		inputBlobRedisKey := fmt.Sprintf("async_pipeline_blob:%s:%d", pipelineTriggerID, idx)
		s.redisClient.Set(
			context.Background(),
			inputBlobRedisKey,
			inputJson,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerAsyncPipelineWorkflow",
		&worker.TriggerAsyncPipelineWorkflowRequest{
			PipelineInputBlobRedisKeys: inputBlobRedisKeys,
			Pipeline:                   dbPipeline,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	dataMappingIndices := []string{}
	for idx := range inputs {
		dataMappingIndices = append(dataMappingIndices, inputs[idx].DataMappingIndex)
	}

	return &pipelinePB.TriggerAsyncPipelineResponse{
		Operation: &longrunningpb.Operation{
			Name: fmt.Sprintf("operations/%s", pipelineTriggerID),
			Done: false,
		},
		DataMappingIndices: dataMappingIndices,
	}, nil

}
