package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
)

const (
	UploadOutputsWorkflow = "UploadOutputsToMinioWorkflow"
)

func (w *worker) UploadToMinioActivity(ctx context.Context, param *UploadToMinioActivityParam) (string, error) {
	url, _, err := w.minioClient.UploadFileBytes(ctx, param.ObjectName, param.Data, param.ContentType)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (w *worker) UploadInputsToMinioActivity(ctx context.Context, param *UploadInputsToMinioActivityParam) error {
	log, _ := logger.GetZapLogger(ctx)
	log.Info("UploadInputsToMinioActivity started")

	batchMemory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		log.Error("failed to load pipeline run inputs", zap.Error(err))
		return err
	}
	var pipelineData []*structpb.Struct
	for _, memory := range batchMemory {
		jsonBytes, err := json.Marshal(memory.Variable)
		if err != nil {
			log.Error("failed to marshal memory variable to json", zap.Error(err))
			return err
		}

		data := &structpb.Struct{}
		err = protojson.Unmarshal(jsonBytes, data)
		if err != nil {
			log.Error("failed to unmarshal memory variable to trigger data", zap.Error(err))
			return err
		}
		pipelineData = append(pipelineData, data)
	}

	objectName := fmt.Sprintf("pipeline-runs/input/%s.json", param.PipelineTriggerID)

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, pipelineData, constant.ContentTypeJSON)
	if err != nil {
		log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	inputs := datamodel.JSONB{{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	err = w.repository.UpdatePipelineRun(ctx, param.PipelineTriggerID, &datamodel.PipelineRun{Inputs: inputs})
	if err != nil {
		log.Error("failed to save pipeline run input data", zap.Error(err))
		return err
	}
	return nil
}

func (w *worker) UploadRecipeToMinioActivity(ctx context.Context, param *UploadRecipeToMinioActivityParam) error {
	log, _ := logger.GetZapLogger(ctx)
	log.Info("UploadReceiptToMinioActivity started", zap.String("PipelineTriggerID", param.PipelineTriggerID))

	url, minioObjectInfo, err := w.minioClient.UploadFileBytes(ctx, param.ObjectName, param.Data, param.ContentType)
	if err != nil {
		w.log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	err = w.repository.UpdatePipelineRun(ctx, param.PipelineTriggerID, &datamodel.PipelineRun{RecipeSnapshot: datamodel.JSONB{{
		Name: minioObjectInfo.Key,
		Type: minioObjectInfo.ContentType,
		Size: minioObjectInfo.Size,
		URL:  url,
	}}})
	if err != nil {
		w.log.Error("failed to log pipeline run with recipe snapshot", zap.Error(err))
		return err
	}
	return nil
}

func (w *worker) UploadOutputsToMinioWorkflow(ctx workflow.Context, param *UploadOutputsWorkflowParam) error {
	eventName := "UploadOutputsToMinioWorkflow"
	w.log.Info(fmt.Sprintf("%s started", eventName), zap.String("PipelineTriggerID", param.PipelineTriggerID))

	objectName := fmt.Sprintf("pipeline-runs/output/%s.json", param.PipelineTriggerID)
	sCtx, span := tracer.Start(context.Background(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	url, objectInfo, err := w.minioClient.UploadFile(sCtx, objectName, param.Outputs, constant.ContentTypeJSON)
	if err != nil {
		w.log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	outputs := datamodel.JSONB{{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	err = w.repository.UpdatePipelineRun(sCtx, param.PipelineTriggerID, &datamodel.PipelineRun{Outputs: outputs})
	if err != nil {
		w.log.Error("failed to save pipeline run output data", zap.Error(err))
		return err
	}

	w.log.Info(fmt.Sprintf("%s finished", eventName), zap.String("PipelineTriggerID", param.PipelineTriggerID))
	return nil
}

func (w *worker) UploadComponentInputsActivity(ctx context.Context, param *ComponentActivityParam) error {
	pipelineTriggerID := param.SystemVariables.PipelineTriggerID
	w.log.Info("UploadComponentInputsActivity started", zap.String("PipelineTriggerID", pipelineTriggerID))

	batchMemory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		return err
	}

	compInputs, _, err := w.processInput(batchMemory, param.ID, param.UpstreamIDs, param.Condition, param.Input)
	if err != nil {
		return err
	}
	objectName := fmt.Sprintf("component-runs/%s/input/%s.json", param.ID, pipelineTriggerID)

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, compInputs, constant.ContentTypeJSON)
	if err != nil {
		w.log.Error("failed to upload component run inputs to minio", zap.Error(err))
		return err
	}

	inputs := datamodel.JSONB{{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	err = w.repository.UpdateComponentRun(ctx, pipelineTriggerID, param.ID, &datamodel.ComponentRun{Inputs: inputs})
	if err != nil {
		w.log.Error("failed to save pipeline run input data", zap.Error(err))
		return err
	}
	return nil
}

func (w *worker) UploadComponentOutputsActivity(ctx context.Context, param *ComponentActivityResult) error {
	pipelineTriggerID := param.SystemVariables.PipelineTriggerID
	w.log.Info("UploadComponentOutputsActivity started", zap.String("PipelineTriggerID", pipelineTriggerID))

	objectName := fmt.Sprintf("component-runs/%s/output/%s.json", pipelineTriggerID, param.ID)

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, param.Outputs, constant.ContentTypeJSON)
	if err != nil {
		w.log.Error("failed to upload component run outputs to minio", zap.Error(err))
		return err
	}

	outputs := datamodel.JSONB{{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	err = w.repository.UpdateComponentRun(ctx, pipelineTriggerID, param.ID, &datamodel.ComponentRun{Outputs: outputs})
	if err != nil {
		w.log.Error("failed to save pipeline run output data", zap.Error(err))
		return err
	}
	return nil
}
