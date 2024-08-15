package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"

	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type UploadToMinioActivityParam struct {
	ObjectName  string
	Data        []byte
	ContentType string
}

type UploadToMinioActivityResponse struct {
	URL        string
	ObjectInfo *minio.ObjectInfo
}

func (w *worker) UploadToMinioActivity(ctx context.Context, param UploadToMinioActivityParam) (string, error) {
	url, _, err := w.minioClient.UploadFileBytes(ctx, param.ObjectName, param.Data, param.ContentType)
	if err != nil {
		return "", err
	}
	return url, nil
}

type UploadInputsToMinioActivityParam struct {
	PipelineTriggerID string
	MemoryStorageKey  *recipe.BatchMemoryKey
}

func (w *worker) UploadInputsToMinioActivity(ctx context.Context, param UploadInputsToMinioActivityParam) error {
	log, _ := logger.GetZapLogger(ctx)
	log.Info("UploadInputsToMinioActivity started")

	batchMemory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		log.Error("failed to load pipeline inputs", zap.Error(err))
		return err
	}
	var pipelineData []*pipelinepb.TriggerData
	for _, memory := range batchMemory {
		jsonBytes, err := json.Marshal(memory.Variable)
		if err != nil {
			return err
		}

		data := &pipelinepb.TriggerData{}
		err = protojson.Unmarshal(jsonBytes, data)
		if err != nil {
			return err
		}
		pipelineData = append(pipelineData, data)
	}

	objectName := fmt.Sprintf("pipeline-runs/input/%s.json", param.PipelineTriggerID)

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, pipelineData, constant.ContentTypeJSON)
	if err != nil {
		return err
	}

	inputs := datamodel.JSONB{datamodel.FileReference{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	pipelineRun, err := w.repository.GetPipelineRunByUID(uuid.FromStringOrNil(param.PipelineTriggerID))
	if err != nil {
		return err
	}
	pipelineRun.Inputs = inputs

	return w.repository.UpsertPipelineRun(pipelineRun)
}
