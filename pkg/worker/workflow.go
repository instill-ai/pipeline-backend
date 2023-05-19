package worker

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	modelv1alpha "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
)

type TriggerAsyncPipelineWorkflowParam struct {
	TaskInputRedisKeys []string
	DataMappingIndices []string
	DbPipeline         *datamodel.Pipeline
}

type TriggerAsyncPipelineByFileUploadWorkflowParam struct {
	TaskInputRedisKey  string
	DataMappingIndices []string
	Task               modelPB.Model_Task
	DbPipeline         *datamodel.Pipeline
}

// TriggerActivityParam represents the parameters for TriggerActivity
type TriggerActivityParam struct {
	TaskInputRedisKeys []string
	DataMappingIndices []string
	Model              string
	OwnerPermalink     string
}

// TriggerByFileUploadActivityParam represents the parameters for TriggerActivity
type TriggerByFileUploadActivityParam struct {
	TaskInputRedisKey  string
	DataMappingIndices []string
	Task               modelPB.Model_Task
	Model              string
	OwnerPermalink     string
}

// DestinationActivityParam represents the parameters for DestinationActivity
type DestinationActivityParam struct {
	DataMappingIndices []string
	ModelOutputs       [][]byte
	Destination        string
	DbPipeline         *datamodel.Pipeline
}

// TriggerAsyncPipelineWorkflow is a pipeline trigger workflow definition.
func (w *worker) TriggerAsyncPipelineWorkflow(ctx workflow.Context, param *TriggerAsyncPipelineWorkflowParam) ([][]byte, error) {

	logger := workflow.GetLogger(ctx)
	logger.Info("TriggerAsyncPipelineWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var results [][]byte
	var modelActivities []workflow.Future
	for _, model := range utils.GetModelsFromRecipe(param.DbPipeline.Recipe) {

		modelActivities = append(modelActivities, workflow.ExecuteActivity(ctx, w.TriggerActivity, &TriggerActivityParam{
			TaskInputRedisKeys: param.TaskInputRedisKeys,
			DataMappingIndices: param.DataMappingIndices,
			Model:              model,
			OwnerPermalink:     param.DbPipeline.Owner,
		}))
	}

	for idx := range modelActivities {
		var result []byte
		if err := modelActivities[idx].Get(ctx, &result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	var destinationActivities []workflow.Future
	for _, destination := range utils.GetDestinationsFromRecipe(param.DbPipeline.Recipe) {
		destinationActivities = append(destinationActivities, workflow.ExecuteActivity(ctx, w.DestinationActivity, &DestinationActivityParam{
			Destination:        destination,
			DataMappingIndices: param.DataMappingIndices,
			ModelOutputs:       results,
			DbPipeline:         param.DbPipeline,
		}))

	}
	for idx := range destinationActivities {
		var result []byte
		if err := destinationActivities[idx].Get(ctx, &result); err != nil {
			return nil, err
		}
	}

	for _, key := range param.TaskInputRedisKeys {
		w.redisClient.Del(context.Background(), key)
	}
	logger.Info("TriggerAsyncPipelineWorkflow completed")
	return results, nil
}

// TriggerAsyncPipelineWorkflow is a pipeline trigger workflow definition.
func (w *worker) TriggerAsyncPipelineByFileUploadWorkflow(ctx workflow.Context, param *TriggerAsyncPipelineByFileUploadWorkflowParam) ([][]byte, error) {

	logger := workflow.GetLogger(ctx)
	logger.Info("TriggerAsyncPipelineByFileUploadWorkflow started")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var results [][]byte
	var modelActivities []workflow.Future
	for _, model := range utils.GetModelsFromRecipe(param.DbPipeline.Recipe) {

		modelActivities = append(modelActivities, workflow.ExecuteActivity(ctx, w.TriggerByFileUploadActivity, &TriggerByFileUploadActivityParam{
			TaskInputRedisKey:  param.TaskInputRedisKey,
			DataMappingIndices: param.DataMappingIndices,
			Task:               param.Task,
			Model:              model,
			OwnerPermalink:     param.DbPipeline.Owner,
		}))
	}

	for idx := range modelActivities {
		var result []byte
		if err := modelActivities[idx].Get(ctx, &result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	var destinationActivities []workflow.Future
	for _, destination := range utils.GetDestinationsFromRecipe(param.DbPipeline.Recipe) {
		destinationActivities = append(destinationActivities, workflow.ExecuteActivity(ctx, w.DestinationActivity, &DestinationActivityParam{
			Destination:        destination,
			DataMappingIndices: param.DataMappingIndices,
			ModelOutputs:       results,
			DbPipeline:         param.DbPipeline,
		}))

	}
	for idx := range destinationActivities {
		var result []byte
		if err := destinationActivities[idx].Get(ctx, &result); err != nil {
			return nil, err
		}
	}

	w.redisClient.Del(context.Background(), param.TaskInputRedisKey)

	logger.Info("TriggerAsyncPipelineByFileUploadWorkflow completed")

	return results, nil
}

// TriggerActivity is a pipeline trigger activity definition.
func (w *worker) TriggerActivity(ctx context.Context, param *TriggerActivityParam) ([]byte, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("TriggerActivity started")

	var inputs []*modelv1alpha.TaskInput
	for idx := range param.TaskInputRedisKeys {
		var input modelv1alpha.TaskInput
		json, err := w.redisClient.Get(context.Background(), param.TaskInputRedisKeys[idx]).Bytes()
		if err != nil {
			return nil, err
		}
		if err := protojson.Unmarshal(json, &input); err != nil {
			return nil, err
		}
		inputs = append(inputs, &input)
	}
	modelOutput, err := Trigger(w.modelPublicServiceClient, w.redisClient, inputs, param.DataMappingIndices, param.Model, param.OwnerPermalink)
	if err != nil {
		return nil, err
	}
	modelOutputJson, err := protojson.Marshal(modelOutput)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	logger.Info("TriggerActivity completed")

	return modelOutputJson, nil

}

// TriggerActivity is a pipeline trigger activity definition.
func (w *worker) TriggerByFileUploadActivity(ctx context.Context, param *TriggerByFileUploadActivityParam) ([]byte, error) {

	logger := activity.GetLogger(ctx)
	logger.Info("TriggerByFileUploadActivity started")

	json, err := w.redisClient.Get(context.Background(), param.TaskInputRedisKey).Bytes()
	if err != nil {
		return nil, err
	}
	bytesBuffer := bytes.NewBuffer(json)
	dec := gob.NewDecoder(bytesBuffer)

	var input interface{}
	switch param.Task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		var imageInput *utils.ImageInput
		err := dec.Decode(&imageInput)
		if err != nil {
			return nil, err
		}
		input = imageInput

	case modelPB.Model_TASK_TEXT_TO_IMAGE:

		var imageInput *utils.TextToImageInput
		err := dec.Decode(&imageInput)
		if err != nil {
			return nil, err
		}
		input = imageInput
	case modelPB.Model_TASK_TEXT_GENERATION:
		var imageInput *utils.TextGenerationInput
		err := dec.Decode(&imageInput)
		if err != nil {
			return nil, err
		}
		input = imageInput

	}

	modelOutput, err := TriggerBinaryFileUpload(w.modelPublicServiceClient, w.redisClient, param.Task, input, param.DataMappingIndices, param.Model, param.OwnerPermalink)
	if err != nil {
		return nil, err
	}
	modelOutputJson, err := protojson.Marshal(modelOutput)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	logger.Info("TriggerByFileUploadActivity completed")

	return modelOutputJson, nil

}

func (w *worker) DestinationActivity(ctx context.Context, param *DestinationActivityParam) ([]byte, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("DestinationActivity started")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var modelOutputs []*pipelinePB.ModelOutput

	for idx := range param.ModelOutputs {
		var modelOutput pipelinePB.ModelOutput
		err := protojson.Unmarshal(param.ModelOutputs[idx], &modelOutput)
		if err != nil {
			return nil, err
		}
		modelOutputs = append(modelOutputs, &modelOutput)
	}
	_, err := w.connectorPublicServiceClient.WriteDestinationConnector(utils.InjectOwnerToContextWithOwnerPermalink(ctx, param.DbPipeline.Owner), &connectorPB.WriteDestinationConnectorRequest{
		Name:                param.Destination,
		SyncMode:            connectorPB.SupportedSyncModes_SUPPORTED_SYNC_MODES_FULL_REFRESH,
		DestinationSyncMode: connectorPB.SupportedDestinationSyncModes_SUPPORTED_DESTINATION_SYNC_MODES_APPEND,
		Pipeline:            fmt.Sprintf("pipelines/%s", param.DbPipeline.ID),
		DataMappingIndices:  param.DataMappingIndices,
		ModelOutputs:        modelOutputs,
		Recipe: func() *pipelinePB.Recipe {
			if param.DbPipeline.Recipe != nil {
				b, err := json.Marshal(param.DbPipeline.Recipe)
				if err != nil {
					logger.Error(err.Error())
					return nil
				}
				pbRecipe := pipelinePB.Recipe{}
				err = json.Unmarshal(b, &pbRecipe)
				if err != nil {
					logger.Error(err.Error())
					return nil
				}
				return &pbRecipe
			}
			return nil
		}(),
	})
	if err != nil {
		logger.Error(fmt.Sprintf("[connector-backend] Error %s at destination %s: %v", "WriteDestinationConnector", param.Destination, err.Error()))
	}

	logger.Info("DestinationActivity completed")
	return nil, nil
}
