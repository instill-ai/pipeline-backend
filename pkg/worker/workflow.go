package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type TriggerAsyncPipelineWorkflowRequest struct {
	PipelineInputBlobRedisKeys []string
	Pipeline                   *datamodel.Pipeline
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteConnectorActivityRequest struct {
	InputBlobRedisKeys []string
	Name               string
	OwnerPermalink     string
}

type ExecuteConnectorActivityResponse struct {
	OutputBlobRedisKeys []string
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

func (w *worker) GetBlob(redisKeys []string) ([]*connectorPB.DataPayload, error) {
	payloads := []*connectorPB.DataPayload{}
	for idx := range redisKeys {
		blob, err := w.redisClient.Get(context.Background(), redisKeys[idx]).Bytes()
		if err != nil {
			return nil, err
		}
		payload := &connectorPB.DataPayload{}
		err = protojson.Unmarshal(blob, payload)
		if err != nil {
			return nil, err
		}

		payloads = append(payloads, payload)

	}
	return payloads, nil
}

func (w *worker) SetBlob(inputs []*connectorPB.DataPayload) ([]string, error) {
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

	dataPoint := utils.NewDataPoint(
		strings.Split(param.Pipeline.Owner, "/")[1],
		workflow.GetInfo(ctx).WorkflowExecution.ID,
		param.Pipeline,
		mgmtPB.Mode_MODE_ASYNC,
		startTime,
	)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}

	// TODO: parallel

	componentIdMap := make(map[string]*datamodel.Component)
	for idx := range param.Pipeline.Recipe.Components {
		componentIdMap[param.Pipeline.Recipe.Components[idx].Id] = param.Pipeline.Recipe.Components[idx]
	}
	dag := utils.NewDAG(param.Pipeline.Recipe.Components)
	for _, component := range param.Pipeline.Recipe.Components {
		parents, _, err := utils.ParseDependency(component.Dependencies)
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
			return err
		}
		for idx := range parents {
			dag.AddEdge(componentIdMap[parents[idx]], component)
		}
	}
	orderedComp, err := dag.TopoloicalSort()
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
		w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
		return err
	}

	result := ExecuteConnectorActivityResponse{}
	ctx = workflow.WithActivityOptions(ctx, ao)
	if err := workflow.ExecuteActivity(ctx, w.DownloadActivity, param).Get(ctx, &result); err != nil {
		span.SetStatus(1, err.Error())
		dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
		w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
		return err
	}

	cache := map[string][]*connectorPB.DataPayload{}
	outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
	for idx := range result.OutputBlobRedisKeys {
		defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
	}
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
		w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
		return err
	}
	cache[orderedComp[0].Id] = outputs

	responseCompId := ""

	for _, comp := range orderedComp[1:] {

		_, depMap, err := utils.ParseDependency(comp.Dependencies)
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
			return err
		}
		inputs := MergeData(cache, depMap, len(param.PipelineInputBlobRedisKeys), param.Pipeline, workflow.GetInfo(ctx).WorkflowExecution.ID)
		inputBlobRedisKeys, err := w.SetBlob(inputs)
		for idx := range result.OutputBlobRedisKeys {
			defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
		}
		result := ExecuteConnectorActivityResponse{}
		ctx = workflow.WithActivityOptions(ctx, ao)
		if err := workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ExecuteConnectorActivityRequest{
			InputBlobRedisKeys: inputBlobRedisKeys,
			Name:               comp.ResourceName,
			OwnerPermalink:     param.Pipeline.Owner,
		}).Get(ctx, &result); err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
			return err
		}

		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
			return err
		}
		outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
		for idx := range result.OutputBlobRedisKeys {
			defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
		}
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
			return err
		}
		cache[comp.Id] = outputs
		if comp.ResourceName == fmt.Sprintf("connectors/%s", constant.EndConnectorId) {
			responseCompId = comp.Id
		}
	}

	pipelineOutputs := []*pipelinePB.PipelineDataPayload{}
	if responseCompId == "" {
		for idx := range cache[orderedComp[0].Id] {
			pipelineOutput := &pipelinePB.PipelineDataPayload{
				DataMappingIndex: cache[orderedComp[0].Id][idx].DataMappingIndex,
			}
			pipelineOutputs = append(pipelineOutputs, pipelineOutput)
		}
	} else {
		outputs := cache[responseCompId]
		for idx := range outputs {
			pipelineOutput := &pipelinePB.PipelineDataPayload{
				DataMappingIndex: outputs[idx].DataMappingIndex,
				Images:           utils.DumpPipelineUnstructuredData(outputs[idx].Images),
				Audios:           utils.DumpPipelineUnstructuredData(outputs[idx].Audios),
				Texts:            outputs[idx].Texts,
				StructuredData:   outputs[idx].StructuredData,
				Metadata:         outputs[idx].Metadata,
			}
			pipelineOutputs = append(pipelineOutputs, pipelineOutput)
		}
	}

	for idx := range pipelineOutputs {
		outputJson, err := protojson.Marshal(pipelineOutputs[idx])
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_ERRORED.String()))
			return err
		}

		blobRedisKey := fmt.Sprintf("async_pipeline_response:%s:%d", workflow.GetInfo(ctx).WorkflowExecution.ID, idx)
		w.redisClient.Set(
			context.Background(),
			blobRedisKey,
			outputJson,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
	}

	dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
	w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", mgmtPB.Status_STATUS_COMPLETED.String()))
	logger.Info("TriggerAsyncPipelineWorkflow completed")
	return nil
}

func (w *worker) DownloadActivity(ctx context.Context, param *TriggerAsyncPipelineWorkflowRequest) (*ExecuteConnectorActivityResponse, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("DownloadActivity started")
	var pipelineInputs []*pipelinePB.PipelineDataPayload

	for idx := range param.PipelineInputBlobRedisKeys {
		blob, err := w.redisClient.Get(context.Background(), param.PipelineInputBlobRedisKeys[idx]).Bytes()
		if err != nil {
			return nil, err
		}
		pipelineInput := &pipelinePB.PipelineDataPayload{}
		err = protojson.Unmarshal(blob, pipelineInput)
		if err != nil {
			return nil, err
		}

		pipelineInputs = append(pipelineInputs, pipelineInput)

	}

	for idx := range param.PipelineInputBlobRedisKeys {
		defer w.redisClient.Del(context.Background(), param.PipelineInputBlobRedisKeys[idx])
	}

	var connectorInputs []*connectorPB.DataPayload

	// Download images

	for idx := range pipelineInputs {
		images, err := utils.LoadPipelineUnstructuredData(pipelineInputs[idx].Images)
		if err != nil {
			return nil, err
		}
		audios, err := utils.LoadPipelineUnstructuredData(pipelineInputs[idx].Audios)
		if err != nil {
			return nil, err
		}
		connectorInputs = append(connectorInputs, &connectorPB.DataPayload{
			DataMappingIndex: pipelineInputs[idx].DataMappingIndex,
			Images:           images,
			Audios:           audios,
			Texts:            pipelineInputs[idx].Texts,
			StructuredData:   pipelineInputs[idx].StructuredData,
			Metadata:         pipelineInputs[idx].Metadata,
		})
	}
	outputBlobRedisKeys, err := w.SetBlob(connectorInputs)
	if err != nil {
		return nil, err
	}

	logger.Info("DownloadActivity completed")
	return &ExecuteConnectorActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteConnectorActivityResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ConnectorActivity started")

	inputs, err := w.GetBlob(param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	resp, err := w.connectorPublicServiceClient.ExecuteConnector(
		utils.InjectOwnerToContextWithOwnerPermalink(ctx, param.OwnerPermalink),
		&connectorPB.ExecuteConnectorRequest{
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

func MergeData(cache map[string][]*connectorPB.DataPayload, depMap map[string][]string, size int, pipeline *datamodel.Pipeline, pipelineTriggerId string) []*connectorPB.DataPayload {

	outputs := []*connectorPB.DataPayload{}
	for idx := 0; idx < size; idx++ {
		output := &connectorPB.DataPayload{}
		for _, imageDep := range depMap["images"] {
			imageDepName := strings.Split(imageDep, ".")[0]
			output.DataMappingIndex = cache[imageDepName][idx].DataMappingIndex
			output.Images = append(output.Images, cache[imageDepName][idx].Images...)
		}
		for _, audioDep := range depMap["audios"] {
			audioDepName := strings.Split(audioDep, ".")[0]
			output.DataMappingIndex = cache[audioDepName][idx].DataMappingIndex
			output.Audios = append(output.Audios, cache[audioDepName][idx].Audios...)
		}
		for _, textDep := range depMap["texts"] {

			textDepName := strings.Split(textDep, ".")[0]
			output.DataMappingIndex = cache[textDepName][idx].DataMappingIndex
			output.Texts = append(output.Texts, cache[textDepName][idx].Texts...)

		}
		for _, structuredDataDep := range depMap["structured_data"] {

			structuredDataDepName := strings.Split(structuredDataDep, ".")[0]
			output.DataMappingIndex = cache[structuredDataDepName][idx].DataMappingIndex
			for k, v := range cache[structuredDataDepName][idx].StructuredData.AsMap() {
				if output.StructuredData == nil {
					output.StructuredData = &structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}
				}

				val, _ := structpb.NewValue(v)

				output.StructuredData.GetFields()[k] = val
			}
		}
		for _, metadataDep := range depMap["metadata"] {
			metadataDepName := strings.Split(metadataDep, ".")[0]
			output.DataMappingIndex = cache[metadataDepName][idx].DataMappingIndex
			metadataDepField := strings.Split(metadataDep, ".")[1]
			if metadataDepField == "structured_data" {

				for k, v := range cache[metadataDepName][idx].StructuredData.AsMap() {

					if output.Metadata == nil {
						output.Metadata = &structpb.Struct{
							Fields: map[string]*structpb.Value{},
						}
					}
					val, _ := structpb.NewValue(v)

					output.Metadata.GetFields()[k] = val
				}
			}
			if metadataDepField == "metadata" {
				for k, v := range cache[metadataDepName][idx].Metadata.AsMap() {
					if output.Metadata == nil {
						output.Metadata = &structpb.Struct{
							Fields: map[string]*structpb.Value{},
						}
					}
					val, _ := structpb.NewValue(v)

					output.Metadata.GetFields()[k] = val
				}
			}
			if metadataDepField == "texts" {

				if output.Metadata == nil {
					output.Metadata = &structpb.Struct{
						Fields: map[string]*structpb.Value{},
					}
				}
				values := []*structpb.Value{}

				for textIdx := range cache[metadataDepName][idx].Texts {
					values = append(values, structpb.NewStringValue(cache[metadataDepName][idx].Texts[textIdx]))
				}
				output.Metadata.GetFields()["texts"] = structpb.NewListValue(&structpb.ListValue{Values: values})
			}
			if output.Metadata == nil {
				output.Metadata = &structpb.Struct{
					Fields: map[string]*structpb.Value{},
				}
			}

			pipelineVal, _ := structpb.NewValue(map[string]interface{}{
				"id":         pipeline.ID,
				"uid":        pipeline.BaseDynamic.UID.String(),
				"owner":      pipeline.Owner,
				"trigger_id": pipelineTriggerId,
			})

			output.Metadata.GetFields()["pipeline"] = pipelineVal

		}
		outputs = append(outputs, output)

	}
	return outputs
}
