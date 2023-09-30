package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type TriggerAsyncPipelineWorkflowRequest struct {
	PipelineInputBlobRedisKeys []string
	PipelineId                 string
	PipelineUid                uuid.UUID
	PipelineReleaseId          string
	PipelineReleaseUid         uuid.UUID
	PipelineRecipe             *datamodel.Recipe
	OwnerPermalink             string
	ReturnTraces               bool
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteConnectorActivityRequest struct {
	InputBlobRedisKeys []string
	Name               string
	OwnerPermalink     string
	PipelineMetadata   PipelineMetadataStruct
}

type ExecuteConnectorActivityResponse struct {
	OutputBlobRedisKeys []string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteOperatorActivityRequest struct {
	InputBlobRedisKeys []string
	DefinitionName     string
	Configuration      *structpb.Struct
	PipelineMetadata   PipelineMetadataStruct
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

// TriggerAsyncPipelineWorkflow is a pipeline trigger workflow definition.
func (w *worker) TriggerAsyncPipelineWorkflow(ctx workflow.Context, param *TriggerAsyncPipelineWorkflowRequest) error {

	startTime := time.Now()
	eventName := "TriggerAsyncPipelineWorkflow"

	sCtx, span := tracer.Start(context.Background(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(sCtx)
	logger.Info("TriggerAsyncPipelineWorkflow started")

	dataPoint := utils.UsageMetricData{
		OwnerUID:           strings.Split(param.OwnerPermalink, "/")[1],
		TriggerMode:        mgmtPB.Mode_MODE_ASYNC,
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
		return err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = w.writeNewDataPoint(sCtx, dataPoint)
		return err
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
		return err
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
			return err
		}
		inputs = append(inputs, input)
	}

	memory := make([]map[string]interface{}, batchSize)
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
			return err
		}
		memory[idx][orderedComp[0].Id] = inputStruct
		computeTime[orderedComp[0].Id] = 0

		memory[idx]["global"], err = utils.GenerateGlobalValue(param.PipelineUid, param.PipelineRecipe, param.OwnerPermalink)
		if err != nil {
			return err
		}

	}

	responseCompId := ""

	for _, comp := range orderedComp[1:] {
		var compInputs []*structpb.Struct
		for idx := 0; idx < batchSize; idx++ {

			memory[idx][comp.Id] = map[string]interface{}{
				"input":  map[string]interface{}{},
				"output": map[string]interface{}{},
			}

			compInputTemplate := comp.Configuration

			// TODO: remove this hardcode injection
			if comp.DefinitionName == "connector-definitions/blockchain-numbers" {
				recipeByte, err := json.Marshal(param.PipelineRecipe)
				if err != nil {
					return err
				}
				recipePb := &structpb.Struct{}
				err = protojson.Unmarshal(recipeByte, recipePb)
				if err != nil {
					return err
				}

				// TODO: remove this hardcode injection
				if comp.DefinitionName == "connector-definitions/blockchain-numbers" {
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
						return err
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
				return err
			}

			var compInputTemplateStruct interface{}
			err = json.Unmarshal(compInputTemplateJson, &compInputTemplateStruct)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}

			compInputStruct, err := utils.RenderInput(compInputTemplateStruct, memory[idx])
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}
			compInputJson, err := json.Marshal(compInputStruct)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}

			compInput := &structpb.Struct{}
			err = protojson.Unmarshal([]byte(compInputJson), compInput)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}

			memory[idx][comp.Id].(map[string]interface{})["input"] = compInputStruct
			compInputs = append(compInputs, compInput)
		}

		// TODO: refactor
		if utils.IsConnectorDefinition(comp.DefinitionName) && comp.ResourceName != "" {
			inputBlobRedisKeys, err := w.SetBlob(compInputs)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
			}
			result := ExecuteConnectorActivityResponse{}
			ctx = workflow.WithActivityOptions(ctx, ao)

			start := time.Now()
			if err := workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ExecuteConnectorActivityRequest{
				InputBlobRedisKeys: inputBlobRedisKeys,
				Name:               comp.ResourceName,
				OwnerPermalink:     param.OwnerPermalink,
				PipelineMetadata: PipelineMetadataStruct{
					Id:         param.PipelineId,
					Uid:        param.PipelineUid.String(),
					ReleaseId:  param.PipelineReleaseId,
					ReleaseUid: param.PipelineReleaseUid.String(),
					Owner:      param.OwnerPermalink,
					TriggerId:  workflow.GetInfo(ctx).WorkflowExecution.ID,
				},
			}).Get(ctx, &result); err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
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
				return err
			}
			for idx := range outputs {

				outputJson, err := protojson.Marshal(outputs[idx])
				if err != nil {
					return err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return err
				}
				memory[idx][comp.Id].(map[string]interface{})["output"] = outputStruct
			}

		} else if comp.DefinitionName == "operator-definitions/op-end" {
			responseCompId = comp.Id
			computeTime[comp.Id] = 0
		} else if utils.IsOperatorDefinition(comp.DefinitionName) {

			inputBlobRedisKeys, err := w.SetBlob(compInputs)
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
			}
			result := ExecuteConnectorActivityResponse{}
			ctx = workflow.WithActivityOptions(ctx, ao)

			start := time.Now()
			if err := workflow.ExecuteActivity(ctx, w.OperatorActivity, &ExecuteOperatorActivityRequest{
				InputBlobRedisKeys: inputBlobRedisKeys,
				DefinitionName:     comp.DefinitionName,
				Configuration:      comp.Configuration,
				PipelineMetadata: PipelineMetadataStruct{
					Id:         param.PipelineId,
					Uid:        param.PipelineUid.String(),
					ReleaseId:  param.PipelineReleaseId,
					ReleaseUid: param.PipelineReleaseUid.String(),
					Owner:      param.OwnerPermalink,
					TriggerId:  workflow.GetInfo(ctx).WorkflowExecution.ID,
				},
			}).Get(ctx, &result); err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}
			computeTime[comp.Id] = float32(time.Since(start).Seconds())
			if err != nil {
				return err
			}
			outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
			for idx := range result.OutputBlobRedisKeys {
				defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
			}
			if err != nil {
				span.SetStatus(1, err.Error())
				dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
				dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
				_ = w.writeNewDataPoint(sCtx, dataPoint)
				return err
			}
			for idx := range outputs {

				outputJson, err := protojson.Marshal(outputs[idx])
				if err != nil {
					return err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return err
				}
				memory[idx][comp.Id].(map[string]interface{})["output"] = outputStruct
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
					return err
				}
				pipelineOutput.Fields[key] = structVal

			}
			pipelineOutputs = append(pipelineOutputs, pipelineOutput)

		}
	}

	var traces map[string]*pipelinePB.Trace
	if param.ReturnTraces {
		traces, err = utils.GenerateTraces(orderedComp, memory, computeTime, batchSize)

		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
			dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
			_ = w.writeNewDataPoint(sCtx, dataPoint)
			return err
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
		return err
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
	logger.Info("TriggerAsyncPipelineWorkflow completed")
	return nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteConnectorActivityResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ConnectorActivity started")

	inputs, err := w.GetBlob(param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	resp, err := w.connectorPublicServiceClient.ExecuteUserConnectorResource(
		utils.InjectOwnerToContextWithOwnerPermalink(
			metadata.AppendToOutgoingContext(ctx,
				"id", param.PipelineMetadata.Id,
				"uid", param.PipelineMetadata.Uid,
				"release_id", param.PipelineMetadata.ReleaseId,
				"release_uid", param.PipelineMetadata.ReleaseUid,
				"owner", param.PipelineMetadata.Owner,
				"trigger_id", param.PipelineMetadata.TriggerId,
			),
			param.OwnerPermalink),
		&connectorPB.ExecuteUserConnectorResourceRequest{
			Name:   param.Name,
			Inputs: inputs,
		},
	)
	if err != nil {
		logger.Error(fmt.Sprintf("[connector-backend] Error %s at connector %s: %v", "ExecuteConnector", param.Name, err.Error()))
		return nil, err
	}

	outputBlobRedisKeys, err := w.SetBlob(resp.Outputs)
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

	op, err := w.operator.GetOperatorDefinitionById(strings.Split(param.DefinitionName, "/")[1])
	if err != nil {
		return nil, err
	}

	execution, err := w.operator.CreateExecution(uuid.FromStringOrNil(op.Uid), param.Configuration, logger)
	if err != nil {
		return nil, err
	}
	compOutputs, err := execution.Execute(compInputs)
	if err != nil {
		return nil, err
	}

	outputBlobRedisKeys, err := w.SetBlob(compOutputs)
	if err != nil {
		return nil, err
	}

	logger.Info("OperatorActivity completed")
	return &ExecuteOperatorActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
}
