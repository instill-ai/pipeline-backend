package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
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

	err = w.repository.UpdatePipelineRun(param.PipelineTriggerID, &datamodel.PipelineRun{Inputs: inputs})
	if err != nil {
		log.Error("failed to save pipeline run input data", zap.Error(err))
		return err
	}
	return nil
}

func (w *worker) UploadReceiptActivity(ctx context.Context, param *UploadReceiptActivityParam) error {
	log, _ := logger.GetZapLogger(ctx)
	log.Info("UploadReceiptActivity started", zap.String("PipelineTriggerID", param.PipelineTriggerID))

	url, minioObjectInfo, err := w.minioClient.UploadFile(ctx, param.ObjectName, param.Data, param.ContentType)
	if err != nil {
		log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	pipelineRun, err := w.repository.GetPipelineRunByUID(uuid.FromStringOrNil(param.PipelineTriggerID))
	if err != nil {
		log.Error("failed to fetch pipeline run for saving input data", zap.Error(err))
		return err
	}

	// Update the pipelineRun with the new information
	pipelineRun.RecipeSnapshot = datamodel.JSONB{{
		Name: minioObjectInfo.Key,
		Type: minioObjectInfo.ContentType,
		Size: minioObjectInfo.Size,
		URL:  url,
	}}

	// Log the updated pipeline run
	err = w.repository.UpsertPipelineRun(pipelineRun)
	if err != nil {
		log.Error("failed to log pipeline run with recipe snapshot", zap.Error(err))
	}
	return nil
}

func (w *worker) UploadOutputsWorkflow(ctx context.Context, param *UploadInputsToMinioActivityParam) error {
	return nil
}

func (w *worker) UploadComponentInputsActivity(ctx context.Context, param UploadComponentInputsParam) error {

	return nil
}
