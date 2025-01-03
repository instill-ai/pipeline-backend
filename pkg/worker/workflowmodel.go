package worker

import (
	"github.com/minio/minio-go/v7"
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

type UploadOutputsToMinioActivityParam struct {
	PipelineTriggerID string
	ExpiryRuleTag     string
}

type UploadRecipeToMinioActivityParam struct {
	PipelineTriggerID string
	ExpiryRuleTag     string
	UploadToMinioActivityParam
}
