package worker

import (
	"github.com/gofrs/uuid"
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
	UserUID           uuid.UUID
	PipelineTriggerID string
	ExpiryRuleTag     string
}

type UploadRecipeToMinioActivityParam struct {
	UserUID           uuid.UUID
	PipelineTriggerID string
	ExpiryRuleTag     string
	UploadToMinioActivityParam
}
