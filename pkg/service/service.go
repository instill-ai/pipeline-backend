package service

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
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
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
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
	TriggerSyncPipeline(req *pipelinePB.TriggerSyncPipelineRequest, owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerSyncPipelineResponse, error)
	TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest, owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerAsyncPipelineResponse, error)
	TriggerSyncPipelineBinaryFileUpload(owner *mgmtPB.User, pipeline *datamodel.Pipeline, task modelPB.Model_Task, input interface{}) (*pipelinePB.TriggerSyncPipelineBinaryFileUploadResponse, error)
	TriggerAsyncPipelineBinaryFileUpload(ctx context.Context, owner *mgmtPB.User, pipeline *datamodel.Pipeline, task modelPB.Model_Task, input interface{}) (*pipelinePB.TriggerAsyncPipelineResponse, error)
	GetModelByName(owner *mgmtPB.User, modelName string) (*modelPB.Model, error)

	ListPipelinesAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)
	// Controller APIs
	GetResourceState(uid uuid.UUID) (*pipelinePB.Pipeline_State, error)
	UpdateResourceState(uid uuid.UUID, state pipelinePB.Pipeline_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error
}

type service struct {
	repository                    repository.Repository
	mgmtPrivateServiceClient      mgmtPB.MgmtPrivateServiceClient
	connectorPublicServiceClient  connectorPB.ConnectorPublicServiceClient
	connectorPrivateServiceClient connectorPB.ConnectorPrivateServiceClient
	modelPublicServiceClient      modelPB.ModelPublicServiceClient
	modelPrivateServiceClient     modelPB.ModelPrivateServiceClient
	controllerClient              controllerPB.ControllerPrivateServiceClient
	redisClient                   *redis.Client
	temporalClient                client.Client
}

// NewService initiates a service instance
func NewService(r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	c connectorPB.ConnectorPublicServiceClient,
	cPrivate connectorPB.ConnectorPrivateServiceClient,
	m modelPB.ModelPublicServiceClient,
	mPrivate modelPB.ModelPrivateServiceClient,
	ct controllerPB.ControllerPrivateServiceClient,
	rc *redis.Client,
	t client.Client,
) Service {
	return &service{
		repository:                    r,
		mgmtPrivateServiceClient:      u,
		connectorPublicServiceClient:  c,
		connectorPrivateServiceClient: cPrivate,
		modelPublicServiceClient:      m,
		modelPrivateServiceClient:     mPrivate,
		controllerClient:              ct,
		redisClient:                   rc,
		temporalClient:                t,
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

	mode, err := s.checkRecipe(owner, dbPipeline.Recipe)
	if err != nil {
		return nil, err
	}

	dbPipeline.Mode = mode

	// User desires to be active
	dbPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE)

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
		mode, err := s.checkRecipe(owner, toUpdPipeline.Recipe)
		if err != nil {
			return nil, err
		}

		toUpdPipeline.Mode = mode

		// User desires to be active
		toUpdPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE)

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

	mode, recipeErr := s.checkRecipe(owner, recipeRscName)
	if recipeErr != nil {
		return nil, err
	}

	if mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) && state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE) {
		return nil, status.Errorf(codes.InvalidArgument, "Pipeline %s is in the SYNC mode, which is always active", dbPipeline.ID)
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

	dbPipeline.Recipe = recipeRscName

	return dbPipeline, nil
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

func preTriggerPipeline(dbPipeline *datamodel.Pipeline, taskInputs []*modelPB.TaskInput, expectedMode datamodel.PipelineMode) ([]string, error) {
	if dbPipeline.Mode != expectedMode {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not sync", dbPipeline.ID))
	}
	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	var dataMappingIndices []string
	for range taskInputs {
		dataMappingIndices = append(dataMappingIndices, ulid.Make().String())
	}
	return dataMappingIndices, nil
}

func (s *service) TriggerSyncPipeline(req *pipelinePB.TriggerSyncPipelineRequest, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerSyncPipelineResponse, error) {

	dataMappingIndices, err := preTriggerPipeline(dbPipeline, req.TaskInputs, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC))
	if err != nil {
		return nil, err
	}

	var modelOutputs []*pipelinePB.ModelOutput
	for _, model := range utils.GetModelsFromRecipe(dbPipeline.Recipe) {
		modelOutput, err := worker.Trigger(s.modelPublicServiceClient, s.redisClient, req.TaskInputs, dataMappingIndices, model, dbPipeline.Owner)
		if err != nil {
			return nil, err
		}
		modelOutputs = append(modelOutputs, modelOutput)
	}

	if err != nil {
		return nil, err
	}

	return &pipelinePB.TriggerSyncPipelineResponse{
		DataMappingIndices: dataMappingIndices,
		ModelOutputs:       modelOutputs,
	}, nil
}

func (s *service) TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerAsyncPipelineResponse, error) {

	dataMappingIndices, err := preTriggerPipeline(dbPipeline, req.TaskInputs, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC))
	if err != nil {
		return nil, err
	}
	logger, _ := logger.GetZapLogger(ctx)

	if err := s.excludeResourceDetailFromRecipe(dbPipeline.Recipe); err != nil {
		return nil, err
	}

	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	var taskInputRedisKeys []string
	for idx := range req.TaskInputs {
		json, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(&modelPB.TaskInput{
			Input: req.TaskInputs[idx].GetInput(),
		})
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("async_pipeline_blob:%s:%s", id.String(), dataMappingIndices[idx])
		s.redisClient.Set(context.Background(), key, json, time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second)
		taskInputRedisKeys = append(taskInputRedisKeys, key)
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerAsyncPipelineWorkflow",
		&worker.TriggerAsyncPipelineWorkflowParam{
			TaskInputRedisKeys: taskInputRedisKeys,
			DataMappingIndices: dataMappingIndices,
			DbPipeline:         dbPipeline,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return &pipelinePB.TriggerAsyncPipelineResponse{
		Operation: &longrunningpb.Operation{
			Name: fmt.Sprintf("operations/%s", id),
			Done: false,
		},
		DataMappingIndices: dataMappingIndices,
	}, nil

}

func preTriggerPipelineBinaryFileUpload(dbPipeline *datamodel.Pipeline, task modelPB.Model_Task, input interface{}, expectedMode datamodel.PipelineMode) ([]string, error) {
	if dbPipeline.Mode != expectedMode {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not sync", dbPipeline.ID))
	}
	if dbPipeline.State != datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}

	batching := 1
	switch task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		inp := input.(*utils.ImageInput)
		batching = len(inp.FileNames)
	case modelPB.Model_TASK_TEXT_TO_IMAGE,
		modelPB.Model_TASK_TEXT_GENERATION:
		batching = 1
	}
	var dataMappingIndices []string
	for i := 0; i < batching; i++ {
		dataMappingIndices = append(dataMappingIndices, ulid.Make().String())
	}
	return dataMappingIndices, nil
}

func (s *service) TriggerSyncPipelineBinaryFileUpload(owner *mgmtPB.User, dbPipeline *datamodel.Pipeline, task modelPB.Model_Task, input interface{}) (*pipelinePB.TriggerSyncPipelineBinaryFileUploadResponse, error) {

	dataMappingIndices, err := preTriggerPipelineBinaryFileUpload(dbPipeline, task, input, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC))
	if err != nil {
		return nil, err
	}
	var modelOutputs []*pipelinePB.ModelOutput
	for _, model := range utils.GetModelsFromRecipe(dbPipeline.Recipe) {
		modelOutput, err := worker.TriggerBinaryFileUpload(s.modelPublicServiceClient, s.redisClient, task, input, dataMappingIndices, model, dbPipeline.Owner)
		if err != nil {
			return nil, err
		}
		modelOutputs = append(modelOutputs, modelOutput)
	}

	if err != nil {
		return nil, err
	}

	return &pipelinePB.TriggerSyncPipelineBinaryFileUploadResponse{
		DataMappingIndices: dataMappingIndices,
		ModelOutputs:       modelOutputs,
	}, nil

}

func (s *service) TriggerAsyncPipelineBinaryFileUpload(ctx context.Context, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline, task modelPB.Model_Task, input interface{}) (*pipelinePB.TriggerAsyncPipelineResponse, error) {
	logger, _ := logger.GetZapLogger(ctx)
	dataMappingIndices, err := preTriggerPipelineBinaryFileUpload(dbPipeline, task, input, datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC))
	if err != nil {
		return nil, err
	}
	if err := s.excludeResourceDetailFromRecipe(dbPipeline.Recipe); err != nil {
		return nil, err
	}

	// TODO: should refactor these switch cases
	var inputByte []byte
	switch task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION:

		var bytesBuffer bytes.Buffer
		enc := gob.NewEncoder(&bytesBuffer)
		err := enc.Encode(input.(*utils.ImageInput))
		if err != nil {
			return nil, err
		}
		inputByte = bytesBuffer.Bytes()
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		var bytesBuffer bytes.Buffer
		enc := gob.NewEncoder(&bytesBuffer)
		err := enc.Encode(input.(*utils.TextToImageInput))
		if err != nil {
			return nil, err
		}
		inputByte = bytesBuffer.Bytes()
	case modelPB.Model_TASK_TEXT_GENERATION:
		var bytesBuffer bytes.Buffer
		enc := gob.NewEncoder(&bytesBuffer)
		err := enc.Encode(input.(*utils.TextGenerationInput))
		if err != nil {
			return nil, err
		}
		inputByte = bytesBuffer.Bytes()
	}

	id, _ := uuid.NewV4()
	workflowOptions := client.StartWorkflowOptions{
		ID:                       id.String(),
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	taskInputRedisKey := fmt.Sprintf("async_pipeline_blob:%s", id.String())
	s.redisClient.Set(context.Background(), taskInputRedisKey, inputByte, time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second)

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerAsyncPipelineByFileUploadWorkflow",
		&worker.TriggerAsyncPipelineByFileUploadWorkflowParam{
			TaskInputRedisKey:  taskInputRedisKey,
			DataMappingIndices: dataMappingIndices,
			Task:               task,
			DbPipeline:         dbPipeline,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return &pipelinePB.TriggerAsyncPipelineResponse{
		Operation: &longrunningpb.Operation{
			Name: fmt.Sprintf("operations/%s", id),
			Done: false,
		},
		DataMappingIndices: dataMappingIndices}, nil

}

func (s *service) GetModelByName(owner *mgmtPB.User, modelName string) (*modelPB.Model, error) {
	modelResq, err := s.modelPublicServiceClient.GetModel(utils.InjectOwnerToContext(context.Background(), owner), &modelPB.GetModelRequest{
		Name: modelName,
	})
	if err != nil {
		return nil, err
	}
	return modelResq.Model, nil
}
