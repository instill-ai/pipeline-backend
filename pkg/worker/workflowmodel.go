package worker

import (
	"github.com/minio/minio-go/v7"

	"github.com/instill-ai/pipeline-backend/pkg/recipe"
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

type UploadInputsToMinioActivityParam struct {
	PipelineTriggerID string
	MemoryStorageKey  *recipe.BatchMemoryKey
}

type UploadComponentInputsParam struct {
	PipelineTriggerID string
	ComponentID       string
	MemoryStorageKey  *recipe.BatchMemoryKey
}

type UploadReceiptToMinioActivityParam struct {
	PipelineTriggerID string
	UploadToMinioActivityParam
}
