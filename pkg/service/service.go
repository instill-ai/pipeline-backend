package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/operator"
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
	GetConnectorPrivateServiceClient() connectorPB.ConnectorPrivateServiceClient
	GetConnectorPublicServiceClient() connectorPB.ConnectorPublicServiceClient
	GetRedisClient() *redis.Client
	GetOperator() *operator.Operator

	CreatePipeline(owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	ListPipelines(owner *mgmtPB.User, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByID(id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUID(uid uuid.UUID, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, owner *mgmtPB.User, updatedPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error)
	DeletePipeline(id string, owner *mgmtPB.User) error
	UpdatePipelineState(id string, owner *mgmtPB.User, state datamodel.PipelineState) (*datamodel.Pipeline, error)
	UpdatePipelineID(id string, owner *mgmtPB.User, newID string) (*datamodel.Pipeline, error)
	TriggerPipeline(ctx context.Context, req *pipelinePB.TriggerPipelineRequest, owner *mgmtPB.User, pipeline *datamodel.Pipeline, pipelineTriggerId string) (*pipelinePB.TriggerPipelineResponse, error)
	TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest, pipelineTriggerID string, owner *mgmtPB.User, pipeline *datamodel.Pipeline) (*pipelinePB.TriggerAsyncPipelineResponse, error)

	ListPipelinesAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)

	// Controller APIs
	GetResourceState(uid uuid.UUID) (*pipelinePB.Pipeline_State, error)
	UpdateResourceState(uid uuid.UUID, state pipelinePB.Pipeline_State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error
	// Influx API
	WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData) error

	GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error)
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
	operator                      operator.Operator
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
		operator:                      operator.InitOperator(),
	}
}

// GetMgmtPrivateServiceClient returns the management private service client
func (h *service) GetMgmtPrivateServiceClient() mgmtPB.MgmtPrivateServiceClient {
	return h.mgmtPrivateServiceClient
}

func (h *service) GetConnectorPrivateServiceClient() connectorPB.ConnectorPrivateServiceClient {
	return h.connectorPrivateServiceClient
}

func (h *service) GetConnectorPublicServiceClient() connectorPB.ConnectorPublicServiceClient {
	return h.connectorPublicServiceClient
}

func (h *service) GetOperator() *operator.Operator {
	return &h.operator
}

// GetRedisClient returns the redis client
func (h *service) GetRedisClient() *redis.Client {
	return h.redisClient
}

func (s *service) CreatePipeline(owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	dbPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE)

	ownerPermalink := utils.GenOwnerPermalink(owner)
	dbPipeline.Owner = ownerPermalink

	recipePermalink, err := s.recipeNameToPermalink(owner, dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	dbPipeline.Recipe = recipePermalink

	if err := s.repository.CreatePipeline(dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetPipelineByID(dbPipeline.ID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	createdCecipeRscName, err := s.recipePermalinkToName(dbCreatedPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	// Add resource entry to controller
	if err := s.UpdateResourceState(dbCreatedPipeline.UID, pipelinePB.Pipeline_STATE_INACTIVE, nil); err != nil {
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

	return dbPipelines, ps, pt, nil
}

func (s *service) GetPipelineByID(id string, owner *mgmtPB.User, isBasicView bool) (*datamodel.Pipeline, error) {

	ownerPermalink := utils.GenOwnerPermalink(owner)

	dbPipeline, err := s.repository.GetPipelineByID(id, ownerPermalink, isBasicView)
	if err != nil {
		return nil, err
	}

	if !isBasicView {
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

	return dbPipeline, nil
}

func (s *service) UpdatePipeline(id string, owner *mgmtPB.User, toUpdPipeline *datamodel.Pipeline) (*datamodel.Pipeline, error) {

	if toUpdPipeline.Recipe != nil {

		toUpdPipeline.State = datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE)

		recipePermalink, err := s.recipeNameToPermalink(owner, toUpdPipeline.Recipe)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		toUpdPipeline.Recipe = recipePermalink

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

	// Update resource entry in controller
	if err := s.UpdateResourceState(dbPipeline.UID, pipelinePB.Pipeline_STATE_INACTIVE, nil); err != nil {
		return nil, err
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

	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if state == datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
		recipeErr := s.checkRecipe(owner, dbPipeline.Recipe)

		if recipeErr != nil {
			return nil, recipeErr
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

	if err := s.repository.UpdatePipelineID(id, ownerPermalink, newID); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetPipelineByID(newID, ownerPermalink, false)
	if err != nil {
		return nil, err
	}

	recipeRscName, err := s.recipePermalinkToName(dbPipeline.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	dbPipeline.Recipe = recipeRscName

	return dbPipeline, nil
}

func (s *service) preTriggerPipeline(dbPipeline *datamodel.Pipeline, pipelineInputs []*structpb.Struct) error {
	state, err := s.GetResourceState(dbPipeline.UID)
	if err != nil {
		return err
	}
	if *state != pipelinePB.Pipeline_STATE_ACTIVE {
		return status.Error(codes.FailedPrecondition, fmt.Sprintf("The pipeline %s is not active", dbPipeline.ID))
	}
	typeMap := map[string]string{}
	for _, comp := range dbPipeline.Recipe.Components {
		if comp.DefinitionName == "operator-definitions/start-operator" {
			for key, value := range comp.Configuration.Fields["body"].GetStructValue().Fields {
				typeMap[key] = value.GetStructValue().Fields["type"].GetStringValue()
			}
		}
	}
	for idx := range pipelineInputs {
		for key, val := range pipelineInputs[idx].Fields {
			switch typeMap[key] {
			case "integer":
				v, err := strconv.ParseInt(val.GetStringValue(), 10, 64)
				if err != nil {
					return err
				}
				pipelineInputs[idx].Fields[key] = structpb.NewNumberValue(float64(v))
			case "number":
				v, err := strconv.ParseFloat(val.GetStringValue(), 64)
				if err != nil {
					return err
				}
				pipelineInputs[idx].Fields[key] = structpb.NewNumberValue(v)
			case "boolean":
				v, err := strconv.ParseBool(val.GetStringValue())
				if err != nil {
					return err
				}
				pipelineInputs[idx].Fields[key] = structpb.NewBoolValue(v)
			case "text", "image", "audio", "video":
			case "integer_array", "number_array", "boolean_array", "text_array", "image_array", "audio_array", "video_array":
				if val.GetListValue() == nil {
					return fmt.Errorf("%s should be a array", key)
				}

				switch typeMap[key] {
				case "integer_array":
					vals := []interface{}{}
					for _, val := range val.GetListValue().AsSlice() {
						n, err := strconv.ParseInt(val.(string), 10, 64)
						if err != nil {
							return err
						}
						vals = append(vals, n)
					}
					structVal, err := structpb.NewList(vals)
					if err != nil {
						return err
					}
					pipelineInputs[idx].Fields[key] = structpb.NewListValue(structVal)

				case "number_array":
					vals := []interface{}{}
					for _, val := range val.GetListValue().AsSlice() {
						n, err := strconv.ParseFloat(val.(string), 64)
						if err != nil {
							return err
						}
						vals = append(vals, n)
					}
					structVal, err := structpb.NewList(vals)
					if err != nil {
						return err
					}
					pipelineInputs[idx].Fields[key] = structpb.NewListValue(structVal)
				case "boolean_array":
					vals := []interface{}{}
					for _, val := range val.GetListValue().AsSlice() {
						n, err := strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
						vals = append(vals, n)
					}
					structVal, err := structpb.NewList(vals)
					if err != nil {
						return err
					}
					pipelineInputs[idx].Fields[key] = structpb.NewListValue(structVal)

				}
			}

		}
	}

	return nil
}

func (s *service) TriggerPipeline(ctx context.Context, req *pipelinePB.TriggerPipelineRequest, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline, pipelineTriggerId string) (*pipelinePB.TriggerPipelineResponse, error) {

	err := s.preTriggerPipeline(dbPipeline, req.Inputs)
	if err != nil {
		return nil, err
	}

	pipelineInputs := req.Inputs

	var inputs [][]byte

	batchSize := len(pipelineInputs)

	for idx := range pipelineInputs {
		inputStruct := &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		}
		inputStruct.Fields["body"] = structpb.NewStructValue(pipelineInputs[idx])

		input, err := protojson.Marshal(inputStruct)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}

	dag, err := utils.GenerateDAG(dbPipeline.Recipe.Components)
	if err != nil {
		return nil, err
	}

	orderedComp, err := dag.TopoloicalSort()
	if err != nil {
		return nil, err
	}

	cache := make([]map[string]interface{}, batchSize)

	for idx := range inputs {
		cache[idx] = map[string]interface{}{}
		var inputStruct map[string]interface{}
		err := json.Unmarshal(inputs[idx], &inputStruct)
		if err != nil {
			return nil, err
		}

		cache[idx][orderedComp[0].Id] = inputStruct

	}

	responseCompId := ""
	for _, comp := range orderedComp[1:] {
		var compInputs []*structpb.Struct

		for idx := 0; idx < batchSize; idx++ {
			compInputTemplate := comp.Configuration
			compInputTemplateJson, err := protojson.Marshal(compInputTemplate)
			if err != nil {
				return nil, err
			}

			var compInputTemplateStruct interface{}
			err = json.Unmarshal(compInputTemplateJson, &compInputTemplateStruct)
			if err != nil {
				return nil, err
			}

			compInputStruct, err := utils.RenderInput(compInputTemplateStruct, cache[idx])
			if err != nil {
				return nil, err
			}
			compInputJson, err := json.Marshal(compInputStruct)
			if err != nil {
				return nil, err
			}

			compInput := &structpb.Struct{}
			err = protojson.Unmarshal([]byte(compInputJson), compInput)
			if err != nil {
				return nil, err
			}

			compInputs = append(compInputs, compInput)
		}

		if comp.ResourceName != "" {
			resp, err := s.connectorPublicServiceClient.ExecuteConnector(
				utils.InjectOwnerToContextWithOwnerPermalink(
					metadata.AppendToOutgoingContext(ctx,
						"id", dbPipeline.ID,
						"uid", dbPipeline.BaseDynamic.UID.String(),
						"owner", dbPipeline.Owner,
						"trigger_id", pipelineTriggerId,
					),
					utils.GenOwnerPermalink(owner)),
				&connectorPB.ExecuteConnectorRequest{
					Name:   comp.ResourceName,
					Inputs: compInputs,
				},
			)
			if err != nil {
				return nil, err
			}
			for idx := range resp.Outputs {

				outputJson, err := protojson.Marshal(resp.Outputs[idx])
				if err != nil {
					return nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, err
				}
				cache[idx][comp.Id] = outputStruct
			}

		}

		if comp.DefinitionName == "operator-definitions/end-operator" {
			responseCompId = comp.Id
			for idx := range compInputs {
				outputJson, err := protojson.Marshal(compInputs[idx])
				if err != nil {
					return nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, err
				}
				cache[idx][comp.Id] = outputStruct
			}

		}

	}

	pipelineOutputs := []*structpb.Struct{}
	for idx := 0; idx < batchSize; idx++ {
		pipelineOutputJson, err := json.Marshal(cache[idx][responseCompId].(map[string]interface{})["body"])
		if err != nil {
			return nil, err
		}
		pipelineOutput := &structpb.Struct{}
		err = protojson.Unmarshal(pipelineOutputJson, pipelineOutput)
		if err != nil {
			return nil, err
		}
		pipelineOutputs = append(pipelineOutputs, pipelineOutput)

	}

	return &pipelinePB.TriggerPipelineResponse{
		Outputs: pipelineOutputs,
	}, nil
}

func (s *service) TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest, pipelineTriggerID string, owner *mgmtPB.User, dbPipeline *datamodel.Pipeline) (*pipelinePB.TriggerAsyncPipelineResponse, error) {

	inputs := req.Inputs
	err := s.preTriggerPipeline(dbPipeline, inputs)
	if err != nil {
		return nil, err
	}
	logger, _ := logger.GetZapLogger(ctx)

	inputBlobRedisKeys := []string{}
	for idx, input := range inputs {
		inputJson, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		inputBlobRedisKey := fmt.Sprintf("async_pipeline_request:%s:%d", pipelineTriggerID, idx)
		s.redisClient.Set(
			context.Background(),
			inputBlobRedisKey,
			inputJson,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
	}
	memo := map[string]interface{}{}
	memo["number_of_data"] = len(inputBlobRedisKeys)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
		Memo: memo,
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

	return &pipelinePB.TriggerAsyncPipelineResponse{
		Operation: &longrunningpb.Operation{
			Name: fmt.Sprintf("operations/%s", pipelineTriggerID),
			Done: false,
		},
	}, nil

}

func (s *service) GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowId, "")

	if err != nil {
		return nil, err
	}
	return s.getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		payloads := []*structpb.Struct{}
		numberOfData := 0
		err := converter.GetDefaultDataConverter().FromPayload(workflowExecutionInfo.Memo.GetFields()["number_of_data"], &numberOfData)
		if err != nil {
			return nil, err
		}

		for idx := 0; idx < numberOfData; idx++ {
			blobRedisKey := fmt.Sprintf("async_pipeline_response:%s:%d", workflowExecutionInfo.Execution.WorkflowId, idx)
			blob, err := s.redisClient.Get(context.Background(), blobRedisKey).Bytes()
			if err != nil {
				return nil, err
			}
			payload := &structpb.Struct{}
			err = protojson.Unmarshal(blob, payload)
			if err != nil {
				return nil, err
			}

			payloads = append(payloads, payload)

		}
		pipelineResp := pipelinePB.TriggerPipelineResponse{
			Outputs: payloads,
		}

		resp, err := anypb.New(&pipelineResp)
		if err != nil {
			return nil, err
		}
		resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1alpha.TriggerPipelineResponse"
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: resp,
			},
		}
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		operation = longrunningpb.Operation{
			Done: false,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{},
			},
		}
	default:
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &rpcStatus.Status{
					Code:    int32(workflowExecutionInfo.Status),
					Details: []*anypb.Any{},
					Message: "",
				},
			},
		}
	}

	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, nil
}
