package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
)

func (w *worker) UploadInputsToMinioActivity(ctx context.Context, param *UploadInputsToMinioActivityParam) error {
	log := w.log.With(zap.String("PipelineTriggerID", param.PipelineTriggerID))
	log.Info("UploadInputsToMinioActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.PipelineTriggerID)
	if err != nil {
		return err
	}

	pipelineData := make([]*structpb.Struct, wfm.GetBatchSize())

	for i := range wfm.GetBatchSize() {
		val, err := wfm.GetPipelineData(ctx, i, memory.PipelineVariable)
		if err != nil {
			return err
		}
		varStr, err := val.ToStructValue()
		if err != nil {
			return err
		}
		pipelineData[i] = varStr.GetStructValue()
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

	log.Info("UploadInputsToMinioActivity finished")
	return nil
}

func (w *worker) UploadRecipeToMinioActivity(ctx context.Context, param *UploadRecipeToMinioActivityParam) error {
	log := w.log.With(zap.String("PipelineTriggerUID", param.PipelineTriggerID))
	log.Info("UploadReceiptToMinioActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.PipelineTriggerID)
	if err != nil {
		return err
	}

	recipeForUpload := &datamodel.Recipe{
		Version:   wfm.GetRecipe().Version,
		On:        wfm.GetRecipe().On,
		Component: wfm.GetRecipe().Component,
		Variable:  wfm.GetRecipe().Variable,
		Output:    wfm.GetRecipe().Output,
	}
	b, err := json.Marshal(recipeForUpload)
	if err != nil {
		return err
	}

	url, minioObjectInfo, err := w.minioClient.UploadFileBytes(
		ctx,
		log,
		fmt.Sprintf("pipeline-runs/recipe/%s.json", param.PipelineTriggerID),
		b,
		constant.ContentTypeJSON,
	)
	if err != nil {
		log.Error("failed to upload pipeline run inputs to minio", zap.Error(err))
		return err
	}

	err = w.repository.UpdatePipelineRun(ctx, param.PipelineTriggerID, &datamodel.PipelineRun{RecipeSnapshot: datamodel.JSONB{{
		Name: minioObjectInfo.Key,
		Type: minioObjectInfo.ContentType,
		Size: minioObjectInfo.Size,
		URL:  url,
	}}})
	if err != nil {
		log.Error("failed to log pipeline run with recipe snapshot", zap.Error(err))
		return err
	}

	log.Info("UploadReceiptToMinioActivity finished")
	return nil
}

func (w *worker) UploadOutputsToMinioActivity(ctx context.Context, param *UploadOutputsToMinioActivityParam) error {
	eventName := "UploadOutputsToMinioActivity"
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

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, outputStructs, constant.ContentTypeJSON)
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

	objectName := fmt.Sprintf("component-runs/%s/input/%s.json", param.ID, pipelineTriggerID)

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, compInputs, constant.ContentTypeJSON)
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

	err = w.repository.UpdateComponentRun(ctx, pipelineTriggerID, param.ID, &datamodel.ComponentRun{Inputs: inputs})
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

	url, objectInfo, err := w.minioClient.UploadFile(ctx, objectName, compOutputs, constant.ContentTypeJSON)
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
