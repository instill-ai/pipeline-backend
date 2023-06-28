package worker

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type TriggerAsyncPipelineWorkflowRequest struct {
	PipelineInputBlobRedisKey string
	Pipeline                  *datamodel.Pipeline
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteConnectorActivityRequest struct {
	InputBlobRedisKey string
	Name              string
	OwnerPermalink    string
}

type ExecuteConnectorActivityResponse struct {
	OutpurBlobRedisKey string
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

func GetBlob(redisClient *redis.Client, redisKey string, output interface{}) error {
	blob, err := redisClient.Get(context.Background(), redisKey).Bytes()
	if err != nil {
		return err
	}
	bytesBuffer := bytes.NewBuffer(blob)
	dec := gob.NewDecoder(bytesBuffer)

	err = dec.Decode(output)
	if err != nil {
		return err
	}
	return nil
}

func SetBlob(redisClient *redis.Client, input interface{}) (string, error) {

	id, _ := uuid.NewV4()
	redisKey := ""

	switch input.(type) {
	case []*pipelinePB.PipelineDataPayload:
		redisKey = fmt.Sprintf("async_pipeline_blob:%s", id.String())
	case []*connectorPB.DataPayload:
		redisKey = fmt.Sprintf("async_connector_blob:%s", id.String())
	}

	var bytesBuffer bytes.Buffer
	enc := gob.NewEncoder(&bytesBuffer)
	err := enc.Encode(input)
	if err != nil {
		return "", err
	}

	redisClient.Set(
		context.Background(),
		redisKey,
		bytesBuffer.Bytes(),
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)
	return redisKey, nil
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
	err := GetBlob(w.redisClient, param.PipelineInputBlobRedisKey, &pipelineInputs)
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint = dataPoint.AddField("compute_time_duration", time.Since(startTime).Seconds())
		w.influxDBWriteClient.WritePoint(dataPoint.AddTag("status", "errored"))
		return err
	}

	defer w.redisClient.Del(context.Background(), param.PipelineInputBlobRedisKey)

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
			Images:         images,
			Texts:          pipelineInputs[idx].Texts,
			StructuredData: pipelineInputs[idx].StructuredData,
			Metadata:       pipelineInputs[idx].Metadata,
		})
	}

	fmt.Println(connectorInputs)

	// TODO: DAG
	// var destinationActivities []workflow.Future
	// for _, destinationName := range utils.GetResourceFromRecipe(param.Pipeline.Recipe, connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION) {

	// 	inputBlobRedisKey, err := SetBlob(w.redisClient, connectorInputs)
	// 	if err != nil {
	// 		return err
	// 	}

	// ao := workflow.ActivityOptions{
	// 	StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
	// 	RetryPolicy: &temporal.RetryPolicy{
	// 		MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
	// 	},
	// }
	// ctx = workflow.WithActivityOptions(ctx, ao)
	// 	destinationActivities = append(destinationActivities, workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ExecuteConnectorActivityRequest{
	// 		InputBlobRedisKey: inputBlobRedisKey,
	// 		Name:              destinationName,
	// 		OwnerPermalink:    param.Pipeline.Owner,
	// 	}))

	// }
	// for idx := range destinationActivities {
	// 	var result []byte
	// 	if err := destinationActivities[idx].Get(ctx, &result); err != nil {
	// 		return err
	// 	}
	// }

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

	var inputs []*connectorPB.DataPayload
	err := GetBlob(w.redisClient, param.InputBlobRedisKey, &inputs)
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

	outputBlobRedisKey, err := SetBlob(w.redisClient, resp.Outputs)
	if err != nil {

		return nil, err
	}

	logger.Info("ConnectorActivity completed")
	return &ExecuteConnectorActivityResponse{OutpurBlobRedisKey: outputBlobRedisKey}, nil
}
