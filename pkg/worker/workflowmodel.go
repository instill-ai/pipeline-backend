package worker

import (
	"github.com/minio/minio-go/v7"
	"google.golang.org/protobuf/types/known/structpb"

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

type UploadOutputsWorkflowParam struct {
	PipelineTriggerID string
	Outputs           []*structpb.Struct
}

type UploadRecipeToMinioActivityParam struct {
	PipelineTriggerID string
	UploadToMinioActivityParam
}
