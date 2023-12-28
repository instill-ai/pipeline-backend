package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type TriggerPipelineWorkflowRequest struct {
	PipelineInputBlobRedisKeys []string
	PipelineId                 string
	PipelineUid                uuid.UUID
	PipelineReleaseId          string
	PipelineReleaseUid         uuid.UUID
	PipelineRecipe             *datamodel.Recipe
	OwnerPermalink             string
	UserPermalink              string
	ReturnTraces               bool
	Mode                       mgmtPB.Mode
}

type TriggerPipelineWorkflowResponse struct {
	OutputBlobRedisKey string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteConnectorActivityRequest struct {
	Id                 string
	InputBlobRedisKeys []string
	DefinitionName     string
	ResourceName       string
	PipelineMetadata   PipelineMetadataStruct
	Task               string
}

type ExecuteConnectorActivityResponse struct {
	OutputBlobRedisKeys []string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteOperatorActivityRequest struct {
	Id                 string
	InputBlobRedisKeys []string
	DefinitionName     string
	PipelineMetadata   PipelineMetadataStruct
	Task               string
}

type ExecuteOperatorActivityResponse struct {
	OutputBlobRedisKeys []string
}

type PipelineMetadataStruct struct {
	Id         string
	Uid        string
	ReleaseId  string
	ReleaseUid string
	Owner      string
	TriggerId  string
	UserUid    string
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

func (w *worker) GetBlob(redisKeys []string) ([]*structpb.Struct, error) {
	payloads := []*structpb.Struct{}
	for idx := range redisKeys {
		blob, err := w.redisClient.Get(context.Background(), redisKeys[idx]).Bytes()
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
	return payloads, nil
}

func (w *worker) SetBlob(inputs []*structpb.Struct) ([]string, error) {
	id, _ := uuid.NewV4()
	blobRedisKeys := []string{}
	for idx, input := range inputs {
		inputJson, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		blobRedisKey := fmt.Sprintf("async_connector_blob:%s:%d", id.String(), idx)
		w.redisClient.Set(
			context.Background(),
			blobRedisKey,
			inputJson,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		blobRedisKeys = append(blobRedisKeys, blobRedisKey)
	}
	return blobRedisKeys, nil
}

// TriggerPipelineWorkflow is a pipeline trigger workflow definition.
func (w *worker) TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowRequest) (*TriggerPipelineWorkflowResponse, error) {

	startTime := time.Now()
	eventName := "TriggerPipelineWorkflow"

	sCtx, span := tracer.Start(context.Background(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(sCtx)
	logger.Info("TriggerPipelineWorkflow started")

	namespace := strings.Split(param.OwnerPermalink, "/")[0]
	var ownerType mgmtPB.OwnerType
	switch namespace {
	case "organizations":
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION
	case "users":
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           strings.Split(param.OwnerPermalink, "/")[1],
		OwnerType:          ownerType,
		UserUID:            strings.Split(param.UserPermalink, "/")[1],
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER, // TODO: currently only support /users type, will change after beta
		TriggerMode:        param.Mode,
		PipelineID:         param.PipelineId,
		PipelineUID:        param.PipelineUid.String(),
		PipelineReleaseID:  param.PipelineReleaseId,
		PipelineReleaseUID: param.PipelineReleaseUid.String(),
		PipelineTriggerUID: workflow.GetInfo(ctx).WorkflowExecution.ID,
		TriggerTime:        startTime.Format(time.RFC3339Nano),
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}

	// TODO: parallel
	dag, err := utils.GenerateDAG(param.PipelineRecipe.Components)
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = w.writeNewDataPoint(sCtx, dataPoint)
		return nil, err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = w.writeNewDataPoint(sCtx, dataPoint)
		return nil, err
	}
	var startCompId string
	for _, c := range orderedComp {
		if c.DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3" {
			startCompId = c.Id
		}
	}

	result := ExecuteConnectorActivityResponse{}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var inputs [][]byte
	pipelineInputs, err := w.GetBlob(param.PipelineInputBlobRedisKeys)
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = w.writeNewDataPoint(sCtx, dataPoint)
		return nil, err
	}
	batchSize := len(pipelineInputs)
	for idx := range pipelineInputs {
		inputStruct := structpb.NewStructValue(pipelineInputs[idx])

		input, err := protojson.Marshal(inputStruct)
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
			dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
			_ = w.writeNewDataPoint(sCtx, dataPoint)
			return nil, err
		}
		inputs = append(inputs, input)
	}

	memory := make([]map[string]interface{}, batchSize)
	statuses := make([]map[string]*utils.ComponentStatus, batchSize)
	computeTime := map[string]float32{}

	for idx := range inputs {
		memory[idx] = map[string]interface{}{}
		var inputStruct map[string]interface{}
		err := json.Unmarshal(inputs[idx], &inputStruct)
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
			dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
			_ = w.writeNewDataPoint(sCtx, dataPoint)
			return nil, err
		}
		memory[idx][startCompId] = inputStruct
		computeTime[startCompId] = 0

		memory[idx]["global"], err = utils.GenerateGlobalValue(param.PipelineUid, param.PipelineRecipe, param.OwnerPermalink)
		if err != nil {
			return nil, err
		}
		statuses[idx] = map[string]*utils.ComponentStatus{}
		statuses[idx][startCompId] = &utils.ComponentStatus{}
		statuses[idx][startCompId].Started = true
		statuses[idx][startCompId].Completed = true

	}

	responseCompId := ""
	for _, comp := range orderedComp {
		if comp.Id == startCompId {
			continue
		}

		for idx := 0; idx < batchSize; idx++ {
			statuses[idx][comp.Id] = &utils.ComponentStatus{}
		}

		var compInputs []*structpb.Struct

		idxMap := map[int]int{}

		for idx := 0; idx < batchSize; idx++ {
			memory[idx][comp.Id] = map[string]interface{}{
				"input":  map[string]interface{}{},
				"output": map[string]interface{}{},
				"status": map[string]interface{}{
					"started":   false,
					"completed": false,
					"skipped":   false,
				},
			}

			for _, ancestorID := range dag.GetAncestorIDs(comp.Id) {
				if statuses[idx][ancestorID].Skipped {
					memory[idx][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["skipped"] = true
					statuses[idx][comp.Id].Skipped = true
					break
				}
			}

			if !statuses[idx][comp.Id].Skipped {
				if comp.Configuration.Fields["condition"].GetStringValue() != "" {
					expr, err := parser.ParseExpr(comp.Configuration.Fields["condition"].GetStringValue())
					if err != nil {
						return nil, err
					}
					cond, err := utils.EvalCondition(expr, memory[idx])
					if err != nil {
						return nil, err
					}
					if cond == false {
						memory[idx][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["skipped"] = true
						statuses[idx][comp.Id].Skipped = true
					} else {
						memory[idx][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["started"] = true
						statuses[idx][comp.Id].Started = true
					}
				} else {
					memory[idx][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["started"] = true
					statuses[idx][comp.Id].Started = true
				}
			}

			if statuses[idx][comp.Id].Started {
				compInputTemplate := comp.Configuration

				// TODO: remove this hardcode injection
				// blockchain-numbers
				if comp.DefinitionName == "connector-definitions/70d8664a-d512-4517-a5e8-5d4da81756a7" {
					recipeByte, err := json.Marshal(param.PipelineRecipe)
					if err != nil {
						return nil, err
					}
					recipePb := &structpb.Struct{}
					err = protojson.Unmarshal(recipeByte, recipePb)
					if err != nil {
						return nil, err
					}

					// TODO: remove this hardcode injection
					if comp.DefinitionName == "connector-definitions/70d8664a-d512-4517-a5e8-5d4da81756a7" {
						metadata, err := structpb.NewValue(map[string]interface{}{
							"pipeline": map[string]interface{}{
								"uid":    "{global.pipeline.uid}",
								"recipe": "{global.pipeline.recipe}",
							},
							"owner": map[string]interface{}{
								"uid": "{global.owner.uid}",
							},
						})
						if err != nil {
							return nil, err
						}
						if compInputTemplate.Fields["input"].GetStructValue().Fields["custom"].GetStructValue() == nil {
							compInputTemplate.Fields["input"].GetStructValue().Fields["custom"] = structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}})
						}
						compInputTemplate.Fields["input"].GetStructValue().Fields["custom"].GetStructValue().Fields["metadata"] = metadata
					}
				}

				compInputTemplateJson, err := protojson.Marshal(compInputTemplate.Fields["input"].GetStructValue())
				if err != nil {
					span.SetStatus(1, err.Error())
					dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
					dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
					_ = w.writeNewDataPoint(sCtx, dataPoint)
					return nil, err
				}

				var compInputTemplateStruct interface{}
				err = json.Unmarshal(compInputTemplateJson, &compInputTemplateStruct)
				if err != nil {
					span.SetStatus(1, err.Error())
					dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
					dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
					_ = w.writeNewDataPoint(sCtx, dataPoint)
					return nil, err
				}

				compInputStruct, err := utils.RenderInput(compInputTemplateStruct, memory[idx])
				if err != nil {
					span.SetStatus(1, err.Error())
					dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
					dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
					_ = w.writeNewDataPoint(sCtx, dataPoint)
					return nil, err
				}
				compInputJson, err := json.Marshal(compInputStruct)
				if err != nil {
					span.SetStatus(1, err.Error())
					dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
					dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
					_ = w.writeNewDataPoint(sCtx, dataPoint)
					return nil, err
				}

				compInput := &structpb.Struct{}
				err = protojson.Unmarshal([]byte(compInputJson), compInput)
				if err != nil {
					span.SetStatus(1, err.Error())
					dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
					dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
					_ = w.writeNewDataPoint(sCtx, dataPoint)
					return nil, err
				}

				memory[idx][comp.Id].(map[string]interface{})["input"] = compInputStruct
				idxMap[len(compInputs)] = idx
				compInputs = append(compInputs, compInput)
			}
		}

		task := ""
		if comp.Configuration.Fields["task"] != nil {
			task = comp.Configuration.Fields["task"].GetStringValue()
		}

		// TODO: refactor
		if utils.IsConnectorDefinition(comp.DefinitionName) && comp.ResourceName != "" {
			inputBlobRedisKeys, err := w.SetBlob(compInputs)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return nil, err
			}
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
			}
			result := ExecuteConnectorActivityResponse{}
			ctx = workflow.WithActivityOptions(ctx, ao)

			start := time.Now()
			if err := workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ExecuteConnectorActivityRequest{
				Id:                 comp.Id,
				InputBlobRedisKeys: inputBlobRedisKeys,
				DefinitionName:     comp.DefinitionName,
				ResourceName:       comp.ResourceName,
				PipelineMetadata: PipelineMetadataStruct{
					Id:         param.PipelineId,
					Uid:        param.PipelineUid.String(),
					ReleaseId:  param.PipelineReleaseId,
					ReleaseUid: param.PipelineReleaseUid.String(),
					Owner:      param.OwnerPermalink,
					TriggerId:  workflow.GetInfo(ctx).WorkflowExecution.ID,
					UserUid:    strings.Split(param.UserPermalink, "/")[1],
				},
				Task: task,
			}).Get(ctx, &result); err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return nil, err
			}
			computeTime[comp.Id] = float32(time.Since(start).Seconds())
			outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
			}
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return nil, err
			}
			for compBatchIdx := range outputs {

				outputJson, err := protojson.Marshal(outputs[compBatchIdx])
				if err != nil {
					return nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, err
				}
				memory[idxMap[compBatchIdx]][comp.Id].(map[string]interface{})["output"] = outputStruct
				memory[idxMap[compBatchIdx]][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["completed"] = true
				statuses[idxMap[compBatchIdx]][comp.Id].Completed = true
			}

		} else if comp.DefinitionName == "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516" {
			responseCompId = comp.Id
			for compBatchIdx := range compInputs {
				memory[idxMap[compBatchIdx]][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["completed"] = true
				statuses[idxMap[compBatchIdx]][comp.Id].Completed = true
			}
			computeTime[comp.Id] = 0
		} else if utils.IsOperatorDefinition(comp.DefinitionName) {

			inputBlobRedisKeys, err := w.SetBlob(compInputs)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return nil, err
			}
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
			}
			result := ExecuteOperatorActivityResponse{}
			ctx = workflow.WithActivityOptions(ctx, ao)

			start := time.Now()
			if err := workflow.ExecuteActivity(ctx, w.OperatorActivity, &ExecuteOperatorActivityRequest{
				Id:                 comp.Id,
				InputBlobRedisKeys: inputBlobRedisKeys,
				DefinitionName:     comp.DefinitionName,
				PipelineMetadata: PipelineMetadataStruct{
					Id:         param.PipelineId,
					Uid:        param.PipelineUid.String(),
					ReleaseId:  param.PipelineReleaseId,
					ReleaseUid: param.PipelineReleaseUid.String(),
					Owner:      param.OwnerPermalink,
					TriggerId:  workflow.GetInfo(ctx).WorkflowExecution.ID,
					UserUid:    strings.Split(param.UserPermalink, "/")[1],
				},
				Task: task,
			}).Get(ctx, &result); err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return nil, err
			}
			computeTime[comp.Id] = float32(time.Since(start).Seconds())
			outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
			}
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return nil, err
			}
			for compBatchIdx := range outputs {

				outputJson, err := protojson.Marshal(outputs[compBatchIdx])
				if err != nil {
					return nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, err
				}
				memory[idxMap[compBatchIdx]][comp.Id].(map[string]interface{})["output"] = outputStruct
				memory[idxMap[compBatchIdx]][comp.Id].(map[string]interface{})["status"].(map[string]interface{})["completed"] = true
				statuses[idxMap[compBatchIdx]][comp.Id].Completed = true
			}

		}

	}

	pipelineOutputs := []*structpb.Struct{}
	if responseCompId == "" {
		for idx := 0; idx < batchSize; idx++ {
			pipelineOutputs = append(pipelineOutputs, &structpb.Struct{})
		}
	} else {
		for idx := 0; idx < batchSize; idx++ {
			pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}
			for key, value := range memory[idx][responseCompId].(map[string]interface{})["input"].(map[string]interface{}) {
				structVal, err := structpb.NewValue(value)
				if err != nil {
					return nil, err
				}
				pipelineOutput.Fields[key] = structVal

			}
			pipelineOutputs = append(pipelineOutputs, pipelineOutput)

		}
	}

	var traces map[string]*pipelinePB.Trace
	if param.ReturnTraces {
		traces, err = utils.GenerateTraces(orderedComp, memory, statuses, computeTime, batchSize)

		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
			dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
			_ = w.writeNewDataPoint(sCtx, dataPoint)
			return nil, err
		}
	}

	pipelineResp := &pipelinePB.TriggerUserPipelineResponse{
		Outputs: pipelineOutputs,
		Metadata: &pipelinePB.TriggerMetadata{
			Traces: traces,
		},
	}
	outputJson, err := protojson.Marshal(pipelineResp)
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = w.writeNewDataPoint(sCtx, dataPoint)
		return nil, err
	}
	blobRedisKey := fmt.Sprintf("async_pipeline_response:%s", workflow.GetInfo(ctx).WorkflowExecution.ID)
	w.redisClient.Set(
		context.Background(),
		blobRedisKey,
		outputJson,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED

	if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}
	logger.Info("TriggerPipelineWorkflow completed")
	return &TriggerPipelineWorkflowResponse{
		OutputBlobRedisKey: blobRedisKey,
	}, nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteConnectorActivityResponse, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ConnectorActivity started")

	compInputs, err := w.GetBlob(param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	con, err := w.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(param.DefinitionName, "/")[1]))
	if err != nil {
		return nil, err
	}

	dbConnector, err := w.repository.GetConnectorByUIDAdmin(ctx, uuid.FromStringOrNil(strings.Split(param.ResourceName, "/")[1]), false)
	if err != nil {
		return nil, err
	}

	configuration := func() *structpb.Struct {
		if dbConnector.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbConnector.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			// TODO: optimize this
			str.Fields["instill_user_uid"] = structpb.NewStringValue(param.PipelineMetadata.UserUid)
			str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
			return &str
		}
		str := structpb.Struct{Fields: make(map[string]*structpb.Value)}
		// TODO: optimize this
		str.Fields["instill_model_backend"] = structpb.NewStringValue(param.PipelineMetadata.UserUid)
		str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
		return nil
	}()

	// TODO
	execution, err := w.connector.CreateExecution(uuid.FromStringOrNil(con.Uid), param.Task, configuration, logger)
	if err != nil {
		return nil, err
	}
	compOutputs, err := execution.ExecuteWithValidation(compInputs)
	if err != nil {
		return nil, w.toApplicationError(err, param.Id, ConnectorActivityError)
	}

	outputBlobRedisKeys, err := w.SetBlob(compOutputs)
	if err != nil {
		return nil, err
	}

	logger.Info("ConnectorActivity completed")
	return &ExecuteConnectorActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
}

func (w *worker) OperatorActivity(ctx context.Context, param *ExecuteOperatorActivityRequest) (*ExecuteOperatorActivityResponse, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("OperatorActivity started")

	compInputs, err := w.GetBlob(param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	op, err := w.operator.GetOperatorDefinitionByUID(uuid.FromStringOrNil(strings.Split(param.DefinitionName, "/")[1]))
	if err != nil {
		return nil, err
	}

	execution, err := w.operator.CreateExecution(uuid.FromStringOrNil(op.Uid), param.Task, nil, logger)
	if err != nil {
		return nil, err
	}
	compOutputs, err := execution.ExecuteWithValidation(compInputs)
	if err != nil {
		return nil, w.toApplicationError(err, param.Id, OperatorActivityError)
	}

	outputBlobRedisKeys, err := w.SetBlob(compOutputs)
	if err != nil {
		return nil, err
	}

	logger.Info("OperatorActivity completed")
	return &ExecuteOperatorActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
}

// toApplicationError wraps a temporal task error in a temporal.Application
// error, adding end-user information that can be extracted by the temporal
// client.
func (w *worker) toApplicationError(err error, componentID, errType string) error {
	details := EndUserErrorDetails{
		// If no end-user message is present in the error, MessageOrErr will
		// return the string version of the error. For an end user, this extra
		// information is more actionable than no information at all.
		Message: fmt.Sprintf("Component %s failed to execute. %s", componentID, errmsg.MessageOrErr(err)),
	}
	// return fault.Wrap(err, fmsg.WithDesc("component failed to execute", issue))
	return temporal.NewApplicationErrorWithCause("component failed to execute", errType, err, details)
}

// The following constants help temporal clients to trace the origin of an
// execution error. They can be leveraged to e.g. define retry policy rules.
// This may evolve in the future to values that have more to do with the
// business domain (e.g. VendorError (non billable), InputDataError (billable),
// etc.).
const (
	ConnectorActivityError = "ConnectorActivityError"
	OperatorActivityError  = "OperatorActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}
