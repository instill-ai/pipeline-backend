package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/memory"

	constantx "github.com/instill-ai/x/constant"
	miniox "github.com/instill-ai/x/minio"
)

// UploadRecipeToMinIOParam contains the information to upload a pipeline
// recipe to MinIO.
type UploadRecipeToMinIOParam struct {
	Recipe   *datamodel.Recipe
	Metadata MinIOUploadMetadata
}

func (w *worker) UploadRecipeToMinIOActivity(ctx context.Context, param UploadRecipeToMinIOParam) error {
	log := w.log.With(zap.String("PipelineTriggerUID", param.Metadata.PipelineTriggerID))
	log.Info("UploadRecipeToMinIOActivity started")

	recipeForUpload := &datamodel.Recipe{
		Version:   param.Recipe.Version,
		On:        param.Recipe.On,
		Component: param.Recipe.Component,
		Variable:  param.Recipe.Variable,
		Output:    param.Recipe.Output,
	}
	b, err := json.Marshal(recipeForUpload)
	if err != nil {
		return err
	}

	url, minioObjectInfo, err := w.minioClient.WithLogger(log).UploadFileBytes(
		ctx,
		&miniox.UploadFileBytesParam{
			UserUID:       param.Metadata.UserUID,
			FilePath:      fmt.Sprintf("pipeline-runs/recipe/%s.json", param.Metadata.PipelineTriggerID),
			FileBytes:     b,
			FileMimeType:  constantx.ContentTypeJSON,
			ExpiryRuleTag: param.Metadata.ExpiryRuleTag,
		},
	)
	if err != nil {
		log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	err = w.repository.UpdatePipelineRun(ctx, param.Metadata.PipelineTriggerID, &datamodel.PipelineRun{RecipeSnapshot: datamodel.JSONB{{
		Name: minioObjectInfo.Key,
		Type: minioObjectInfo.ContentType,
		Size: minioObjectInfo.Size,
		URL:  url,
	}}})
	if err != nil {
		log.Error("failed to save pipeline run recipe snapshot", zap.Error(err))
		return err
	}

	log.Info("UploadRecipeToMinIOActivity completed")
	return nil
}

// MinIOUploadMetadata contains information needed to upload an object to
// MinIO.
type MinIOUploadMetadata struct {
	UserUID           uuid.UUID
	PipelineTriggerID string
	ExpiryRuleTag     string
}

func (w *worker) UploadOutputsToMinIOActivity(ctx context.Context, param *MinIOUploadMetadata) error {
	eventName := "UploadOutputsToMinIOActivity"
	log := w.log.With(zap.String("PipelineTriggerUID", param.PipelineTriggerID))
	log.Info(fmt.Sprintf("%s started", eventName))

	objectName := fmt.Sprintf("pipeline-runs/output/%s.json", param.PipelineTriggerID)

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.PipelineTriggerID)
	if err != nil {
		return err
	}

	outputStructs := make([]*structpb.Struct, wfm.GetBatchSize())

	for idx := range wfm.GetBatchSize() {

		outputVal, err := wfm.GetPipelineData(ctx, idx, memory.PipelineOutput)
		if err != nil {
			return err
		}
		outputValStr, err := outputVal.ToStructValue()
		if err != nil {
			return err
		}

		outputStructs[idx] = outputValStr.GetStructValue()
	}

	url, objectInfo, err := w.minioClient.WithLogger(log).UploadFile(
		ctx,
		&miniox.UploadFileParam{
			UserUID:       param.UserUID,
			FilePath:      objectName,
			FileContent:   outputStructs,
			FileMimeType:  constantx.ContentTypeJSON,
			ExpiryRuleTag: param.ExpiryRuleTag,
		},
	)
	if err != nil {
		log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	outputs := datamodel.JSONB{{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	err = w.repository.UpdatePipelineRun(ctx, param.PipelineTriggerID, &datamodel.PipelineRun{Outputs: outputs})
	if err != nil {
		log.Error("failed to save pipeline run output data", zap.Error(err))
		return err
	}

	log.Info(fmt.Sprintf("%s finished", eventName))
	return nil
}

func (w *worker) UploadComponentInputsActivity(ctx context.Context, param *ComponentActivityParam) error {
	pipelineTriggerID := param.SystemVariables.PipelineTriggerID
	log := w.log.With(zap.String("PipelineTriggerUID", pipelineTriggerID), zap.String("ComponentID", param.ID))

	log.Info("UploadComponentInputsActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return err
	}

	compInputs := make([]*structpb.Struct, wfm.GetBatchSize())

	for i := range wfm.GetBatchSize() {
		val, err := wfm.GetComponentData(ctx, i, param.ID, memory.ComponentDataInput)
		if err != nil {
			return err
		}
		varStr, err := val.ToStructValue()
		if err != nil {
			return err
		}
		compInputs[i] = varStr.GetStructValue()
	}

	// Process unstructured data in the inputs and upload to MinIO
	compInputs, err = w.processAndUploadUnstructuredData(ctx, compInputs, param)
	if err != nil {
		return err
	}

	objectName := fmt.Sprintf("component-runs/%s/input/%s.json", param.ID, pipelineTriggerID)

	url, objectInfo, err := w.minioClient.WithLogger(log).UploadFile(
		ctx,
		&miniox.UploadFileParam{
			UserUID:       param.SystemVariables.PipelineUserUID,
			FilePath:      objectName,
			FileContent:   compInputs,
			FileMimeType:  constantx.ContentTypeJSON,
			ExpiryRuleTag: param.SystemVariables.ExpiryRule.Tag,
		},
	)
	if err != nil {
		log.Error("failed to upload component run inputs to minio", zap.Error(err))
		return err
	}

	inputs := datamodel.JSONB{{
		Name: objectInfo.Key,
		Type: objectInfo.ContentType,
		Size: objectInfo.Size,
		URL:  url,
	}}

	componentRunUpdate := &datamodel.ComponentRun{
		Inputs: inputs,
	}

	if param.SystemVariables.ExpiryRule.ExpirationDays > 0 {
		blobExpiration := time.Now().UTC().AddDate(0, 0, param.SystemVariables.ExpiryRule.ExpirationDays)
		componentRunUpdate.BlobDataExpirationTime = null.TimeFrom(blobExpiration)
	}
	err = w.repository.UpdateComponentRun(ctx, pipelineTriggerID, param.ID, componentRunUpdate)
	if err != nil {
		log.Error("failed to save pipeline run input data", zap.Error(err))
		return err
	}
	return nil
}

func (w *worker) UploadComponentOutputsActivity(ctx context.Context, param *ComponentActivityParam) error {
	pipelineTriggerID := param.SystemVariables.PipelineTriggerID
	log := w.log.With(zap.String("PipelineTriggerUID", pipelineTriggerID), zap.String("ComponentID", param.ID))

	log.Info("UploadComponentOutputsActivity started")

	objectName := fmt.Sprintf("component-runs/%s/output/%s.json", pipelineTriggerID, param.ID)

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return err
	}

	compOutputs := make([]*structpb.Struct, wfm.GetBatchSize())

	for i := range wfm.GetBatchSize() {
		val, err := wfm.GetComponentData(ctx, i, param.ID, memory.ComponentDataOutput)
		if err != nil {
			return err
		}
		varStr, err := val.ToStructValue()
		if err != nil {
			return err
		}
		compOutputs[i] = varStr.GetStructValue()
	}

	// Process unstructured data in the outputs and upload to MinIO
	compOutputs, err = w.processAndUploadUnstructuredData(ctx, compOutputs, param)
	if err != nil {
		return err
	}

	url, objectInfo, err := w.minioClient.WithLogger(log).UploadFile(
		ctx,
		&miniox.UploadFileParam{
			UserUID:       param.SystemVariables.PipelineUserUID,
			FilePath:      objectName,
			FileContent:   compOutputs,
			FileMimeType:  constantx.ContentTypeJSON,
			ExpiryRuleTag: param.SystemVariables.ExpiryRule.Tag,
		},
	)
	if err != nil {
		log.Error("failed to upload component run outputs to minio", zap.Error(err))
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
		log.Error("failed to save pipeline run output data", zap.Error(err))
		return err
	}

	return nil
}

// processAndUploadUnstructuredData processes unstructured data in structs and uploads to MinIO
func (w *worker) processAndUploadUnstructuredData(ctx context.Context, dataStructs []*structpb.Struct, param *ComponentActivityParam) ([]*structpb.Struct, error) {
	updatedDataStructs := make([]*structpb.Struct, len(dataStructs))
	for i, dataStruct := range dataStructs {
		updatedDataStruct, err := processStructUnstructuredData(ctx, dataStruct, w.uploadUnstructuredDataToMinIO, param)
		if err != nil {
			// Note: we don't want to fail the whole process if one of the data structs fails to upload.
			updatedDataStructs[i] = dataStruct
		} else {
			updatedDataStructs[i] = updatedDataStruct
		}
	}
	return updatedDataStructs, nil
}

// uploadUnstructuredDataToMinIO uploads unstructured data to MinIO and returns a download URL
func (w *worker) uploadUnstructuredDataToMinIO(ctx context.Context, data string, param *ComponentActivityParam) (string, error) {
	// Generate unique object name
	uid, err := uuid.NewV4()
	if err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}

	// Get MIME type
	mimeType, err := getMimeType(data)
	if err != nil {
		return "", fmt.Errorf("get mime type: %w", err)
	}

	// Remove prefix and decode base64 data
	base64Data := removePrefix(data)
	decodedData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("decode base64 data: %w", err)
	}

	// Generate object name with extension
	objectName := fmt.Sprintf("%s/%s%s", param.SystemVariables.PipelineRequesterUID.String(), uid.String(), getFileExtension(mimeType))

	// Upload to MinIO
	uploadParam := miniox.UploadFileBytesParam{
		UserUID:       param.SystemVariables.PipelineRequesterUID,
		FilePath:      objectName,
		FileBytes:     decodedData,
		FileMimeType:  mimeType,
		ExpiryRuleTag: param.SystemVariables.ExpiryRule.Tag,
	}

	downloadURL, _, err := w.minioClient.UploadFileBytes(ctx, &uploadParam)
	if err != nil {
		return "", fmt.Errorf("upload to MinIO: %w", err)
	}

	return downloadURL, nil
}

// getMimeType extracts or detects the MIME type from data
func getMimeType(data string) (string, error) {
	var mimeType string
	if strings.HasPrefix(data, "data:") {
		contentType := strings.TrimPrefix(data, "data:")
		parts := strings.SplitN(contentType, ";", 2)
		if len(parts) == 0 {
			return "", fmt.Errorf("invalid data url")
		}
		mimeType = parts[0]
	} else {
		b, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return "", fmt.Errorf("decode base64 string: %w", err)
		}
		mimeType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	}
	return mimeType, nil
}

// getFileExtension maps MIME types to file extensions
func getFileExtension(mimeType string) string {
	ext, err := mime.ExtensionsByType(mimeType)
	if err != nil {
		return ""
	}
	if len(ext) == 0 {
		return ""
	}
	return ext[0]
}

// removePrefix removes the data URI prefix and returns the base64 data
func removePrefix(data string) string {
	if strings.HasPrefix(data, "data:") {
		parts := strings.SplitN(data, ",", 2)
		if len(parts) == 0 {
			return ""
		}
		return parts[1]
	}
	return data
}
