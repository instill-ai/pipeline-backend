package worker

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
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
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

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
		startTime,
	)

	var pipelineInputs []*pipelinePB.PipelineDataPayload

	for idx := range param.PipelineInputBlobRedisKeys {
		blob, err := w.redisClient.Get(context.Background(), param.PipelineInputBlobRedisKeys[idx]).Bytes()
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", "errored"))
			return err
		}
		pipelineInput := &pipelinePB.PipelineDataPayload{}
		err = protojson.Unmarshal(blob, pipelineInput)
		if err != nil {
			span.SetStatus(1, err.Error())
			dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
			w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", "errored"))
			return err
		}

		pipelineInputs = append(pipelineInputs, pipelineInput)

	}

	for idx := range param.PipelineInputBlobRedisKeys {
		defer w.redisClient.Del(context.Background(), param.PipelineInputBlobRedisKeys[idx])
	}

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
					logger.Error(fmt.Sprintf("logUnable to download image at %v. %v", imageUrl, err))
					span.SetStatus(1, err.Error())
					dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
					w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", "errored"))
					return fmt.Errorf("unable to download image at %v", imageUrl)
				}
				defer response.Body.Close()

				buff := new(bytes.Buffer) // pointer
				_, err = buff.ReadFrom(response.Body)
				if err != nil {
					logger.Error(fmt.Sprintf("Unable to read content body from image at %v. %v", imageUrl, err))
					span.SetStatus(1, err.Error())
					dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
					w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", "errored"))
					return fmt.Errorf("unable to read content body from image at %v", imageUrl)
				}
				images = append(images, buff.Bytes())
			}
		}
	}

	var connectorInputs []*connectorPB.DataPayload
	for idx := range pipelineInputs {
		connectorInputs = append(connectorInputs, &connectorPB.DataPayload{
			DataMappingIndex: pipelineInputs[idx].DataMappingIndex,
			Images:           images,
			Texts:            pipelineInputs[idx].Texts,
			StructuredData:   pipelineInputs[idx].StructuredData,
			Metadata:         pipelineInputs[idx].Metadata,
		})
	}

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
			return err
		}
		for idx := range parents {
			dag.AddEdge(componentIdMap[parents[idx]], component)
		}
	}
	orderedComp, err := dag.TopoloicalSort()
	if err != nil {
		return err
	}

	cache := map[string][]*connectorPB.DataPayload{}
	cache[orderedComp[0].Id] = connectorInputs

	for _, comp := range orderedComp[1:] {

		_, depMap, err := utils.ParseDependency(comp.Dependencies)
		if err != nil {
			return err
		}
		inputs := MergeData(cache, depMap, len(param.PipelineInputBlobRedisKeys))
		inputBlobRedisKeys, err := w.SetBlob(inputs)
		result := ExecuteConnectorActivityResponse{}
		ctx = workflow.WithActivityOptions(ctx, ao)
		if err := workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ExecuteConnectorActivityRequest{
			InputBlobRedisKeys: inputBlobRedisKeys,
			Name:               comp.ResourceName,
			OwnerPermalink:     param.Pipeline.Owner,
		}).Get(ctx, &result); err != nil {
			return err
		}

		for idx := range result.OutputBlobRedisKeys {
			defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
		}

		if err != nil {
			return err
		}
		outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
		if err != nil {
			return err
		}
		cache[comp.Id] = outputs
	}

	dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
	w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", "completed"))

	logger.Info("TriggerAsyncPipelineWorkflow completed")
	return nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteConnectorActivityResponse, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("ConnectorActivity started")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

func MergeData(cache map[string][]*connectorPB.DataPayload, depMap map[string][]string, size int) []*connectorPB.DataPayload {

	outputs := []*connectorPB.DataPayload{}
	for idx := 0; idx < size; idx++ {
		output := &connectorPB.DataPayload{}
		for _, imageDep := range depMap["images"] {
			imageDepName := strings.Split(imageDep, ".")[0]
			output.DataMappingIndex = cache[imageDepName][idx].DataMappingIndex
			output.Images = append(output.Images, cache[imageDepName][idx].Images...)

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

		}
		outputs = append(outputs, output)

	}
	return outputs
}
