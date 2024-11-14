// This file will be refactored soon
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/blobstorage"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	SystemVariables recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	Mode            mgmtpb.Mode
	TriggerFromAPI  bool
	WorkerUID       uuid.UUID
	// Namespace is the namespace of the requester of the pipeline.
	Namespace resource.Namespace
}

type SchedulePipelineWorkflowParam struct {
	Namespace          resource.Namespace
	PipelineID         string
	PipelineUID        uuid.UUID
	PipelineReleaseID  string
	PipelineReleaseUID uuid.UUID
	ExpiryRuleTag      string
}

type SchedulePipelineLoaderActivityParam struct {
	Namespace          resource.Namespace
	PipelineUID        uuid.UUID
	PipelineReleaseUID uuid.UUID
}

type SchedulePipelineLoaderActivityResult struct {
	ScheduleID string
	Pipeline   *datamodel.Pipeline
}

// ComponentActivityParam represents the parameters for TriggerActivity
type ComponentActivityParam struct {
	WorkflowID      string
	ID              string
	UpstreamIDs     []string
	Condition       string
	Type            string
	Task            string
	SystemVariables recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	Streaming       bool
	// Namespace is the namespace of the requester of the pipeline.
	Namespace resource.Namespace
}

type PreIteratorActivityParam struct {
	WorkflowID      string
	ID              string
	UpstreamIDs     []string
	Condition       string
	Input           string
	Range           any
	Index           string
	SystemVariables recipe.SystemVariables
}

type PreIteratorActivityResult struct {
	ChildWorkflowIDs []string
	ConditionMap     map[int]int
}

type PostIteratorActivityParam struct {
	WorkflowID      string
	ID              string
	ConditionMap    map[int]int
	OutputElements  map[string]string
	SystemVariables recipe.SystemVariables
}

type PreTriggerActivityParam struct {
	WorkflowID      string
	SystemVariables recipe.SystemVariables
}

type LoadDAGDataActivityParam struct {
	WorkflowID string
}

type LoadDAGDataActivityResult struct {
	Recipe    *datamodel.Recipe
	BatchSize int
}

type PostTriggerActivityParam struct {
	WorkflowID      string
	SystemVariables recipe.SystemVariables
}

type UpsertPipelineRunActivityParam struct {
	PipelineRun *datamodel.PipelineRun
}

type UpdatePipelineRunActivityParam struct {
	PipelineTriggerID string
	PipelineRun       *datamodel.PipelineRun
}

type UpsertComponentRunActivityParam struct {
	ComponentRun *datamodel.ComponentRun
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

// WorkFlowSignal is used by sChan to signal the status of components in the Workflow.
type WorkFlowSignal struct {
	ID     string
	Status string
}

// TriggerPipelineWorkflow is a pipeline trigger workflow definition.
// The workflow is only responsible for orchestrating the DAG, not processing or reading/writing the data.
// All data processing should be done in activities.
func (w *worker) TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowParam) error {
	eventName := "TriggerPipelineWorkflow"
	startTime := time.Now()
	sCtx, span := tracer.Start(
		context.Background(),
		eventName,
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	logger, _ := logger.GetZapLogger(sCtx)
	logger.Info("TriggerPipelineWorkflow started")

	// Options for activity worker
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	// Options for MinIO activity worker
	mo := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if param.WorkerUID == uuid.Nil {
		ao.TaskQueue = w.workerUID.String()
		mo.TaskQueue = w.workerUID.String()
	} else {
		ao.TaskQueue = param.WorkerUID.String()
		mo.TaskQueue = fmt.Sprintf("%s-minio", param.WorkerUID.String())
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	minioCtx := workflow.WithActivityOptions(ctx, mo)

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
	if param.TriggerFromAPI {
		cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
		defer func() {
			if err := workflow.ExecuteActivity(
				cleanupCtx,
				w.ClosePipelineActivity,
				workflowID,
			).Get(cleanupCtx, nil); err != nil {
				logger.Error("Failed to clean up trigger workflow", zap.Error(err))
			}
		}()
	}

	var ownerType mgmtpb.OwnerType
	switch param.SystemVariables.PipelineOwnerType {
	case resource.Organization:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION
	case resource.User:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           param.SystemVariables.PipelineOwnerUID.String(),
		OwnerType:          ownerType,
		UserUID:            param.SystemVariables.PipelineUserUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		RequesterUID:       param.SystemVariables.PipelineRequesterUID.String(),
		RequesterType:      mgmtpb.OwnerType_OWNER_TYPE_USER,
		TriggerMode:        param.Mode,
		PipelineID:         param.SystemVariables.PipelineID,
		PipelineUID:        param.SystemVariables.PipelineUID.String(),
		PipelineReleaseID:  param.SystemVariables.PipelineReleaseID,
		PipelineReleaseUID: param.SystemVariables.PipelineReleaseUID.String(),
		PipelineTriggerUID: workflow.GetInfo(ctx).WorkflowExecution.ID,
		TriggerTime:        startTime.Format(time.RFC3339Nano),
	}

	// This is a simplistic check that relies on the only supported
	// namespace switch (user->organization). If other types of impersonation
	// are supported, the requester type should be provided in the system
	// variables.
	if dataPoint.UserUID != dataPoint.RequesterUID {
		dataPoint.RequesterType = mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION
	}

	if param.TriggerFromAPI {
		if err := workflow.ExecuteActivity(ctx, w.PreTriggerActivity, &PreTriggerActivityParam{
			WorkflowID:      workflowID,
			SystemVariables: param.SystemVariables,
		}).Get(ctx, nil); err != nil {
			return err
		}
	}

	_ = workflow.ExecuteActivity(minioCtx, w.UploadRecipeToMinioActivity, &UploadRecipeToMinioActivityParam{
		PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
		ExpiryRuleTag:     param.SystemVariables.ExpiryRuleTag,
	}).Get(ctx, nil)

	_ = workflow.ExecuteActivity(minioCtx, w.UploadInputsToMinioActivity, &UploadInputsToMinioActivityParam{
		PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
		ExpiryRuleTag:     param.SystemVariables.ExpiryRuleTag,
	}).Get(ctx, nil)

	dagData := &LoadDAGDataActivityResult{}
	_ = workflow.ExecuteActivity(ctx, w.LoadDAGDataActivity, &LoadDAGDataActivityParam{
		WorkflowID: workflowID,
	}).Get(ctx, dagData)

	dag, err := recipe.GenerateDAG(dagData.Recipe.Component)
	if err != nil {
		return err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return err
	}

	errs := []error{}
	componentRunFutures := []workflow.Future{}
	componentRunFailed := false
	var componentRunErrors []string
	// The components in the same group can be executed in parallel
	for group := range orderedComp {
		futures := []workflow.Future{}
		futureArgs := []*ComponentActivityParam{}
		for compID, comp := range orderedComp[group] {
			upstreamIDs := dag.GetUpstreamCompIDs(compID)

			switch comp.Type {
			default:
				componentRun := &datamodel.ComponentRun{
					PipelineTriggerUID: uuid.FromStringOrNil(param.SystemVariables.PipelineTriggerID),
					ComponentID:        compID,
					Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
					StartedTime:        time.Now(),
				}

				// adding the data row in advance in case that UploadComponentInputsActivity starts before ComponentActivity
				_ = workflow.ExecuteActivity(ctx, w.UpsertComponentRunActivity, &UpsertComponentRunActivityParam{
					ComponentRun: componentRun,
				}).Get(ctx, nil)

				args := &ComponentActivityParam{
					WorkflowID:      workflowID,
					ID:              compID,
					UpstreamIDs:     upstreamIDs,
					Type:            comp.Type,
					Task:            comp.Task,
					Condition:       comp.Condition,
					SystemVariables: param.SystemVariables,
				}

				componentRunFutures = append(componentRunFutures, workflow.ExecuteActivity(minioCtx, w.UploadComponentInputsActivity, args))

				futures = append(futures, workflow.ExecuteActivity(ctx, w.ComponentActivity, args))
				futureArgs = append(futureArgs, args)

			case datamodel.Iterator:
				// TODO: support intermediate result streaming for Iterator

				preIteratorResult := &PreIteratorActivityResult{}
				if err = workflow.ExecuteActivity(ctx, w.PreIteratorActivity, &PreIteratorActivityParam{
					WorkflowID:  workflowID,
					ID:          compID,
					UpstreamIDs: upstreamIDs,
					Input: func(c *datamodel.Component) string {
						if c.Input != nil {
							return c.Input.(string)
						}
						return ""
					}(comp),
					Range:           comp.Range,
					Condition:       comp.Condition,
					Index:           comp.Index,
					SystemVariables: param.SystemVariables,
				}).Get(ctx, &preIteratorResult); err != nil {
					errs = append(errs, err)
					continue
				}

				itFutures := []workflow.Future{}
				for iter := range preIteratorResult.ConditionMap {
					childWorkflowOptions := workflow.ChildWorkflowOptions{
						TaskQueue:                TaskQueue,
						WorkflowID:               preIteratorResult.ChildWorkflowIDs[iter],
						WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
						RetryPolicy: &temporal.RetryPolicy{
							MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
						},
					}

					itFutures = append(itFutures, workflow.ExecuteChildWorkflow(
						workflow.WithChildOptions(ctx, childWorkflowOptions),
						"TriggerPipelineWorkflow",
						&TriggerPipelineWorkflowParam{
							TriggerFromAPI:  false,
							SystemVariables: param.SystemVariables,
							Mode:            mgmtpb.Mode_MODE_SYNC,
							WorkerUID:       param.WorkerUID,
							// TODO: support streaming inside iterator.
							// IsStreaming:     param.IsStreaming,
						}))
				}
				for iter := 0; iter < len(itFutures); iter++ {
					err = itFutures[iter].Get(ctx, nil)
					if err != nil {
						errs = append(errs, err)
						continue
					}
				}

				if err = workflow.ExecuteActivity(ctx, w.PostIteratorActivity, &PostIteratorActivityParam{
					WorkflowID:      workflowID,
					ID:              compID,
					ConditionMap:    preIteratorResult.ConditionMap,
					OutputElements:  comp.OutputElements,
					SystemVariables: param.SystemVariables,
				}).Get(ctx, nil); err != nil {
					if err != nil {
						errs = append(errs, err)
						continue
					}
				}
			}

		}

		for idx := range futures {
			err = futures[idx].Get(ctx, nil)
			if err != nil {
				componentRunFailed = true
				componentRunErrors = append(componentRunErrors, fmt.Sprintf("component(ID: %s) run failed", futureArgs[idx].ID))
				errs = append(errs, err)
				continue
			}

			// ComponentActivity is responsible for returning a temporal
			// application error with the relevant information. Wrapping
			// the error here prevents the client from accessing the error
			// message from the activity.
			// return err
			componentRunFutures = append(componentRunFutures, workflow.ExecuteActivity(minioCtx, w.UploadComponentOutputsActivity, futureArgs[idx]))

		}

	}

	duration := time.Since(startTime)
	dataPoint.ComputeTimeDuration = duration.Seconds()
	dataPoint.Status = mgmtpb.Status_STATUS_COMPLETED

	if param.TriggerFromAPI {
		if err := workflow.ExecuteActivity(ctx, w.OutputActivity, &ComponentActivityParam{
			WorkflowID:      workflowID,
			Namespace:       param.Namespace,
			SystemVariables: param.SystemVariables,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if err := workflow.ExecuteActivity(minioCtx, w.UploadOutputsToMinioActivity, &UploadOutputsToMinioActivityParam{
			PipelineTriggerID: workflowID,
			ExpiryRuleTag:     param.SystemVariables.ExpiryRuleTag,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if err := workflow.ExecuteActivity(ctx, w.PostTriggerActivity, &PostTriggerActivityParam{
			WorkflowID:      workflowID,
			SystemVariables: param.SystemVariables,
		}).Get(ctx, nil); err != nil {
			return err
		}

		// TODO: we should check whether to collect failed component or not
		if err := workflow.ExecuteActivity(ctx, w.IncreasePipelineTriggerCountActivity, param.SystemVariables).Get(ctx, nil); err != nil {
			return fmt.Errorf("updating pipeline trigger count: %w", err)
		}

		if len(errs) > 0 {
			w.writeErrorDataPoint(sCtx, errs, span, startTime, &dataPoint)
		} else {
			if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
				logger.Warn(err.Error())
			}
		}

	}

	for _, f := range componentRunFutures {
		_ = f.Get(ctx, nil)
	}

	updatePipelineRunArgs := &UpdatePipelineRunActivityParam{
		PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
		PipelineRun: &datamodel.PipelineRun{
			CompletedTime: null.TimeFrom(time.Now()),
			Status:        datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED),
			TotalDuration: null.IntFrom(duration.Milliseconds()),
		},
	}
	if componentRunFailed {
		updatePipelineRunArgs.PipelineRun.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED)
		updatePipelineRunArgs.PipelineRun.Error = null.StringFrom(strings.Join(componentRunErrors, " / "))
	}

	_ = workflow.ExecuteActivity(ctx, w.UpdatePipelineRunActivity, updatePipelineRunArgs).Get(ctx, nil)

	logger.Info("TriggerPipelineWorkflow completed in", zap.Duration("duration", duration))

	return nil
}

func (w *worker) UpdatePipelineRunActivity(ctx context.Context, param *UpdatePipelineRunActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger = logger.With(zap.String("PipelineTriggerUID", param.PipelineTriggerID))
	logger.Info("UpdatePipelineRunActivity started")

	err := w.repository.UpdatePipelineRun(ctx, param.PipelineTriggerID, param.PipelineRun)
	if err != nil {
		logger.Error("failed to log completed pipeline run", zap.Error(err))
		// Note: We're not returning here because we want to complete the workflow even if logging fails
	}

	logger.Info("UpdatePipelineRunActivity completed")
	return nil
}

func (w *worker) UpsertComponentRunActivity(ctx context.Context, param *UpsertComponentRunActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger = logger.With(zap.String("PipelineTriggerUID", param.ComponentRun.PipelineTriggerUID.String()), zap.String("ComponentID", param.ComponentRun.ComponentID))
	logger.Info("UpsertComponentRunActivity started")
	err := w.repository.UpsertComponentRun(ctx, param.ComponentRun)
	if err != nil {
		logger.Error("failed to log component run start", zap.Error(err))
	}
	logger.Info("UpsertComponentRunActivity completed")
	return nil
}

func (w *worker) ComponentActivity(ctx context.Context, param *ComponentActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ComponentActivity started")

	startTime := time.Now()
	// this is component run actual start time
	err := w.repository.UpdateComponentRun(ctx, param.SystemVariables.PipelineTriggerID, param.ID, &datamodel.ComponentRun{StartedTime: startTime})
	if err != nil {
		logger.Error("failed to log component run start time", zap.Error(err))
	} else {
		defer func() {
			componentRun := &datamodel.ComponentRun{
				CompletedTime: null.TimeFrom(time.Now()),
				TotalDuration: null.IntFrom(time.Since(startTime).Milliseconds()),
			}
			if err != nil {
				componentRun.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED)
				componentRun.Error = null.StringFrom(err.Error())
			} else {
				componentRun.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED)
			}
			err = w.repository.UpdateComponentRun(ctx, param.SystemVariables.PipelineTriggerID, param.ID, componentRun)
			if err != nil {
				logger.Error("failed to log component run end time", zap.Error(err))
			}
		}()
	}

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
	}
	conditionMap, err := w.processCondition(ctx, wfm, param.ID, param.UpstreamIDs, param.Condition)
	if err != nil {
		return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
	}
	if len(conditionMap) > 0 {
		setups, err := NewSetupReader(wfm, param.ID, conditionMap).Read(ctx)
		if err != nil {
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
		}
		sysVars, err := recipe.GenerateSystemVariables(ctx, param.SystemVariables)
		if err != nil {
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
		}
		executionParams := componentstore.ExecutionParams{
			ComponentID:           param.ID,
			ComponentDefinitionID: param.Type,
			SystemVariables:       sysVars,

			// Note: currently, we assume that setup in the batch are all the same
			Setup: setups[0],
			Task:  param.Task,
		}

		execution, err := w.component.CreateExecution(executionParams)
		if err != nil {
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
		}

		jobs := make([]*componentbase.Job, len(conditionMap))
		for idx, originalIdx := range conditionMap {
			jobs[idx] = &componentbase.Job{
				Input:  NewInputReader(wfm, param.ID, originalIdx),
				Output: NewOutputWriter(wfm, param.ID, originalIdx, wfm.IsStreaming()),
				Error:  NewErrorHandler(wfm, param.ID, originalIdx),
			}
		}
		err = execution.Execute(
			ctx,
			jobs,
		)
		if err != nil {
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
		}

		for _, idx := range conditionMap {
			if e, err := wfm.GetComponentStatus(ctx, idx, param.ID, memory.ComponentStatusErrored); err == nil && !e {
				if err = wfm.SetComponentStatus(ctx, idx, param.ID, memory.ComponentStatusCompleted, true); err != nil {
					return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
				}
			}
		}
	}

	logger.Info("ComponentActivity completed")
	return nil
}

func (w *worker) OutputActivity(ctx context.Context, param *ComponentActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("OutputActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return temporal.NewApplicationErrorWithCause("loading pipeline memory", outputActivityErrorType, err)
	}

	for idx := range wfm.GetBatchSize() {
		outputTemplate, err := wfm.Get(ctx, idx, string(memory.PipelineOutputTemplate))
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline output", outputActivityErrorType, err)
		}
		output, err := recipe.Render(ctx, outputTemplate, idx, wfm, true)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline output", outputActivityErrorType, err)
		}

		updatedOutput := w.uploadFileAndReplaceURL(ctx, param, &output)

		err = wfm.SetPipelineData(ctx, idx, memory.PipelineOutput, updatedOutput)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline output", outputActivityErrorType, err)
		}
	}

	logger.Info("OutputActivity completed")
	return nil
}
func (w *worker) uploadFileAndReplaceURL(ctx context.Context, param *ComponentActivityParam, value *format.Value) format.Value {
	logger, _ := logger.GetZapLogger(ctx)
	if value == nil {
		return nil
	}
	switch v := (*value).(type) {
	case format.File:
		downloadURL, err := w.uploadBlobDataAndGetDownloadURL(ctx, param, &v)
		if err != nil || downloadURL == "" {
			logger.Warn("uploading blob data", zap.Error(err))
			return v
		}
		return data.NewString(downloadURL)
	case data.Array:
		newArray := make(data.Array, len(v))
		for i, item := range v {
			newArray[i] = w.uploadFileAndReplaceURL(ctx, param, &item)
		}
		return newArray
	case data.Map:
		newMap := make(data.Map)
		for k, v := range v {
			newMap[k] = w.uploadFileAndReplaceURL(ctx, param, &v)
		}
		return newMap
	default:
		return v
	}
}

func (w *worker) uploadBlobDataAndGetDownloadURL(ctx context.Context, param *ComponentActivityParam, value *format.File) (string, error) {
	artifactClient := w.artifactPublicServiceClient
	ns := param.Namespace

	jsonBytes, err := json.Marshal(param.SystemVariables)
	if err != nil {
		return "", fmt.Errorf("marshal system variables: %w", err)
	}

	var sysVarJSON map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &sysVarJSON); err != nil {
		return "", fmt.Errorf("unmarshal system variables: %w", err)
	}

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(sysVarJSON))

	objectName := fmt.Sprintf("%s/%s", ns.NsID, uuid.Must(uuid.NewV4()).String())
	resp, err := artifactClient.GetObjectUploadURL(ctx, &artifactpb.GetObjectUploadURLRequest{
		NamespaceId:      ns.NsID,
		ObjectName:       objectName,
		ObjectExpireDays: 0,
	})

	if err != nil {
		return "", fmt.Errorf("get upload url: %w", err)
	}

	uploadURL := resp.GetUploadUrl()

	err = uploadBlobData(ctx, uploadURL, value)
	if err != nil {
		return "", fmt.Errorf("upload blob data: %w", err)
	}

	respDownloadURL, err := artifactClient.GetObjectDownloadURL(ctx, &artifactpb.GetObjectDownloadURLRequest{
		NamespaceId: ns.NsID,
		ObjectUid:   resp.GetObject().GetUid(),
	})
	if err != nil {
		return "", fmt.Errorf("get object download url: %w", err)
	}

	return respDownloadURL.GetDownloadUrl(), nil
}

func uploadBlobData(ctx context.Context, uploadURL string, value *format.File) error {
	if uploadURL == "" {
		return fmt.Errorf("empty upload URL provided")
	}

	parsedURL, err := url.Parse(uploadURL)
	if err != nil {
		return fmt.Errorf("parsing upload URL: %w", err)
	}
	parsedURL.Scheme = "http"
	parsedURL.Host = fmt.Sprintf("%s:%d", config.Config.APIGateway.Host, config.Config.APIGateway.PublicPort)
	fullURL := parsedURL.String()
	contentType := (*value).ContentType().String()
	fileBytes, err := (*value).Binary()

	if err != nil {
		return fmt.Errorf("getting file bytes: %w", err)
	}

	err = blobstorage.UploadFile(ctx, fullURL, fileBytes.ByteArray(), contentType)

	if err != nil {
		return fmt.Errorf("uploading blob: %w", err)
	}

	return nil
}

// TODO: complete iterator
// PreIteratorActivity generate the trigger memory for each iteration.
func (w *worker) PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreIteratorActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}
	conditionMap, err := w.processCondition(ctx, wfm, param.ID, param.UpstreamIDs, param.Condition)
	if err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}

	result := &PreIteratorActivityResult{}

	childWorkflowIDs := make([]string, len(conditionMap))

	for idx, originalIdx := range conditionMap {
		if err = wfm.SetComponentStatus(ctx, originalIdx, param.ID, memory.ComponentStatusStarted, true); err != nil {
			return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
		}
		childWorkflowID := fmt.Sprintf("%s:%d:%s:%s:%s", param.WorkflowID, originalIdx, constant.SegComponent, param.ID, constant.SegIteration)
		childWorkflowIDs[idx] = childWorkflowID

		// If `input` is provided, the iteration will be performed over it;
		// otherwise, the iteration will be based on the `range` setup.
		useInput := param.Input != ""

		var indexes []int
		var elems []format.Value
		if useInput {
			input, err := recipe.Render(ctx, data.NewString(param.Input), originalIdx, wfm, false)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}
			elems = input.(data.Array)
			indexes = make([]int, len(elems))
		} else {

			// We offer two syntax options for defining `range`.

			// The first is the **array representation**:
			// ```
			// range: [0, 5, 2]
			// ---
			// range:
			//   - 0
			//   - 5
			//   - 2
			// ```

			// The second is the **map representation**, which is the
			// recommended approach for using references in range values:
			// ```
			// range:
			//   start: 0
			//   stop: ${variable.top-k}
			//   step: 1
			// ```

			rangeParam, err := data.NewValue(param.Range)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error"), preIteratorActivityErrorType, param.ID)
			}
			useArrayRange := false
			switch rangeParam.(type) {
			case data.Array:
				useArrayRange = true
			case data.Map:
				useArrayRange = false
			default:
				return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error"), preIteratorActivityErrorType, param.ID)
			}

			renderedRangeParam, err := recipe.Render(ctx, rangeParam, originalIdx, wfm, false)
			if err != nil {
				return nil, err
			}

			var start, stop, step int

			withStep := false
			if useArrayRange {
				if l := len(rangeParam.(data.Array)); l < 2 || l > 3 {
					return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, must be in the form [start, stop[, step]]"), preIteratorActivityErrorType, param.ID)
				} else if l == 3 {
					withStep = true
				}
				start = renderedRangeParam.(data.Array)[0].(format.Number).Integer()
				stop = renderedRangeParam.(data.Array)[1].(format.Number).Integer()
				if withStep {
					step = renderedRangeParam.(data.Array)[2].(format.Number).Integer()
				}
			} else {
				if _, ok := renderedRangeParam.(data.Map)[rangeStart]; ok {
					start = renderedRangeParam.(data.Map)[rangeStart].(format.Number).Integer()
				} else {
					return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, `start` is missing"), preIteratorActivityErrorType, param.ID)
				}

				if _, ok := renderedRangeParam.(data.Map)[rangeStop]; ok {
					stop = renderedRangeParam.(data.Map)[rangeStop].(format.Number).Integer()
				} else {
					return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, `stop` is missing"), preIteratorActivityErrorType, param.ID)
				}

				if _, ok := renderedRangeParam.(data.Map)[rangeStep]; ok {
					withStep = true
					step = renderedRangeParam.(data.Map)[rangeStep].(format.Number).Integer()
				}

			}

			if !withStep {
				if start > stop {
					return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, the `stop` should be larger then `start`"), preIteratorActivityErrorType, param.ID)
				}
				indexes = make([]int, stop-start)
				for i, j := 0, start; j < stop; i, j = i+1, j+1 {
					indexes[i] = j
				}

			} else {
				if step == 0 {
					return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, the `step` should not be zero"), preIteratorActivityErrorType, param.ID)
				}
				if start > stop {
					if step > 0 {
						return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, the `step` should be negative"), preIteratorActivityErrorType, param.ID)
					}
					for j := start; j > stop; j = j + step {
						indexes = append(indexes, j)
					}
				}
				if start < stop {
					if step < 0 {
						return nil, componentActivityError(ctx, wfm, fmt.Errorf("iterator range error, the `step` should be positive"), preIteratorActivityErrorType, param.ID)
					}
					for j := start; j < stop; j = j + step {
						indexes = append(indexes, j)
					}
				}
			}
		}

		iteratorRecipe := &datamodel.Recipe{
			Component: wfm.GetRecipe().Component[param.ID].Component,
		}

		childWFM, err := w.memoryStore.NewWorkflowMemory(ctx, childWorkflowIDs[idx], iteratorRecipe, len(indexes))
		if err != nil {
			return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
		}

		// When iterating over `input`, each element in the array is processed
		// and stored in memory.
		if useInput {
			for e := range len(indexes) {
				iteratorElem := data.Map{
					"element": elems[e],
				}
				err = childWFM.Set(ctx, e, param.ID, iteratorElem)
				if err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
			}
		} else {
			for e, rangeIndex := range indexes {
				identifier := defaultRangeIdentifier
				if param.Index != "" {
					identifier = param.Index
				}
				err = childWFM.Set(ctx, e, identifier, data.NewNumberFromInteger(rangeIndex))
				if err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
			}
		}

		for e, rangeIndex := range indexes {
			variable, err := wfm.Get(ctx, originalIdx, constant.SegVariable)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}
			secret, err := wfm.Get(ctx, originalIdx, constant.SegSecret)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}
			connection, err := wfm.Get(ctx, originalIdx, constant.SegConnection)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}
			err = childWFM.SetPipelineData(ctx, e, memory.PipelineVariable, variable)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}
			err = childWFM.SetPipelineData(ctx, e, memory.PipelineSecret, secret)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}
			err = childWFM.SetPipelineData(ctx, e, memory.PipelineConnection, connection)
			if err != nil {
				return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
			}

			for _, id := range param.UpstreamIDs {
				component, err := wfm.Get(ctx, originalIdx, id)
				if err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
				err = childWFM.Set(ctx, e, id, component)
				if err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
			}
			for compID, comp := range iteratorRecipe.Component {
				inputVal, err := data.NewValue(comp.Input)
				if err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
				setupVal, err := data.NewValue(comp.Setup)
				if err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
				childWFM.InitComponent(ctx, e, compID)

				inputVal = setIteratorIndex(inputVal, param.Index, rangeIndex)
				if err := childWFM.SetComponentData(ctx, e, compID, memory.ComponentDataInput, inputVal); err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
				if err := childWFM.SetComponentData(ctx, e, compID, memory.ComponentDataSetup, setupVal); err != nil {
					return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
				}
			}
		}

	}
	result.ChildWorkflowIDs = childWorkflowIDs
	result.ConditionMap = conditionMap
	logger.Info("PreIteratorActivity completed")
	return result, nil
}

// PostIteratorActivity merges the trigger memory from each iteration.
func (w *worker) PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PostIteratorActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
	}

	for _, originalIdx := range param.ConditionMap {
		childWorkflowID := fmt.Sprintf("%s:%d:%s:%s:%s", param.WorkflowID, originalIdx, constant.SegComponent, param.ID, constant.SegIteration)
		childWFM, err := w.memoryStore.GetWorkflowMemory(ctx, childWorkflowID)
		if err != nil {
			return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
		}

		output := data.Map{}
		for k, v := range param.OutputElements {
			elemVals := data.Array{}

			for elemIdx := range childWFM.GetBatchSize() {
				elemVal, err := recipe.Render(ctx, data.NewString(v), elemIdx, childWFM, false)
				if err != nil {
					return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
				}
				elemVals = append(elemVals, elemVal)

			}
			output[k] = elemVals
		}
		if err = wfm.SetComponentData(ctx, originalIdx, param.ID, memory.ComponentDataOutput, output); err != nil {
			return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
		}

		if err = wfm.SetComponentStatus(ctx, originalIdx, param.ID, memory.ComponentStatusCompleted, true); err != nil {
			return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
		}
	}

	logger.Info("PostIteratorActivity completed")
	return nil
}

func (w *worker) LoadDAGDataActivity(ctx context.Context, param *LoadDAGDataActivityParam) (*LoadDAGDataActivityResult, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("LoadDAGDataActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return nil, err
	}

	logger.Info("LoadDAGDataActivity completed")
	return &LoadDAGDataActivityResult{
		Recipe:    wfm.GetRecipe(),
		BatchSize: wfm.GetBatchSize(),
	}, nil
}

// PreTriggerActivity clone the trigger memory from Redis to MemoryStore.
func (w *worker) PreTriggerActivity(ctx context.Context, param *PreTriggerActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreTriggerActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return fmt.Errorf("loading pipeline memory: %w", err)
	}

	preTriggerErr := func(err error) error {
		if wfm.IsStreaming() {
			updateTime := time.Now()
			for batchIdx := range wfm.GetBatchSize() {
				if err := w.memoryStore.SendWorkflowStatusEvent(
					ctx,
					param.WorkflowID,
					memory.Event{
						Event: string(memory.PipelineStatusUpdated),
						Data: memory.PipelineStatusUpdatedEventData{
							PipelineEventData: memory.PipelineEventData{
								UpdateTime: updateTime,
								BatchIndex: batchIdx,
								Status: map[memory.PipelineStatusType]bool{
									memory.PipelineStatusStarted:   true,
									memory.PipelineStatusErrored:   true,
									memory.PipelineStatusCompleted: false,
								},
							},
						},
					},
				); err != nil {
					return fmt.Errorf("sending error event: %s", err)
				}
			}

		}

		if msg := errmsg.Message(err); msg != "" {
			return temporal.NewApplicationErrorWithCause(msg, preTriggerActivityErrorType, err)
		}

		return err
	}

	var triggerRecipe *datamodel.Recipe
	if param.SystemVariables.PipelineReleaseUID.IsNil() {
		pipeline, err := w.repository.GetPipelineByUIDAdmin(ctx, param.SystemVariables.PipelineUID, false, false)
		if err != nil {
			return preTriggerErr(fmt.Errorf("loading pipeline recipe: %w", err))
		}
		triggerRecipe = pipeline.Recipe
	} else {
		release, err := w.repository.GetPipelineReleaseByUIDAdmin(ctx, param.SystemVariables.PipelineReleaseUID, false)
		if err != nil {
			return preTriggerErr(fmt.Errorf("loading pipeline recipe: %w", err))
		}
		triggerRecipe = release.Recipe
	}

	wfm.SetRecipe(triggerRecipe)

	// Loading secrets and connections into memory.
	pt := ""
	var nsSecrets []*datamodel.Secret
	ownerPermalink := fmt.Sprintf("%s/%s", param.SystemVariables.PipelineOwnerType, param.SystemVariables.PipelineOwnerUID)
	for {
		var secrets []*datamodel.Secret
		secrets, _, pt, err = w.repository.ListNamespaceSecrets(ctx, ownerPermalink, 100, pt, filtering.Filter{})
		if err != nil {
			return preTriggerErr(fmt.Errorf("loading pipeline secret memory: %w", err))
		}

		for _, secret := range secrets {
			if secret.Value != nil {
				nsSecrets = append(nsSecrets, secret)
			}
		}
		if pt == "" {
			break
		}
	}

	connections := data.Map{}
	for _, comp := range triggerRecipe.Component {
		if connRef, ok := comp.Setup.(string); ok {
			connID, err := recipe.ConnectionIDFromReference(connRef)
			if err != nil {
				return preTriggerErr(fmt.Errorf("resolving connection reference: %w", err))
			}

			if _, connAlreadyLoaded := connections[connID]; connAlreadyLoaded {
				continue
			}

			nsUID, err := resource.GetRscPermalinkUID(ownerPermalink)
			if err != nil {
				return preTriggerErr(fmt.Errorf("extracting owner UID: %w", err))
			}

			conn, err := w.repository.GetNamespaceConnectionByID(ctx, nsUID, connID)
			if err != nil {
				if errors.Is(err, errdomain.ErrNotFound) {
					err = errmsg.AddMessage(err, fmt.Sprintf("Connection %s doesn't exist.", connID))
				}

				return preTriggerErr(fmt.Errorf("fetching connection: %w", err))
			}

			var setup map[string]any
			if err := json.Unmarshal(conn.Setup, &setup); err != nil {
				return preTriggerErr(fmt.Errorf("unmarshalling setup: %w", err))
			}

			setupVal, err := data.NewValue(setup)
			if err != nil {
				return preTriggerErr(fmt.Errorf("transforming connection setup to value: %w", err))
			}

			connections[connID] = setupVal
		}
		if comp.Type == datamodel.Iterator {
			for _, nestedComp := range comp.Component {
				if connRef, ok := nestedComp.Setup.(string); ok {
					connID, err := recipe.ConnectionIDFromReference(connRef)
					if err != nil {
						return preTriggerErr(fmt.Errorf("resolving connection reference: %w", err))
					}

					if _, connAlreadyLoaded := connections[connID]; connAlreadyLoaded {
						continue
					}

					nsUID, err := resource.GetRscPermalinkUID(ownerPermalink)
					if err != nil {
						return preTriggerErr(fmt.Errorf("extracting owner UID: %w", err))
					}

					conn, err := w.repository.GetNamespaceConnectionByID(ctx, nsUID, connID)
					if err != nil {
						if errors.Is(err, errdomain.ErrNotFound) {
							err = errmsg.AddMessage(err, fmt.Sprintf("Connection %s doesn't exist.", connID))
						}

						return preTriggerErr(fmt.Errorf("fetching connection: %w", err))
					}

					var setup map[string]any
					if err := json.Unmarshal(conn.Setup, &setup); err != nil {
						return preTriggerErr(fmt.Errorf("unmarshalling setup: %w", err))
					}

					setupVal, err := data.NewValue(setup)
					if err != nil {
						return preTriggerErr(fmt.Errorf("transforming connection setup to value: %w", err))
					}

					connections[connID] = setupVal
				}
			}
		}
	}

	for idx := range wfm.GetBatchSize() {
		pipelineSecrets, err := wfm.Get(ctx, idx, constant.SegSecret)
		if err != nil {
			return preTriggerErr(fmt.Errorf("loading pipeline secret memory: %w", err))
		}

		for _, secret := range nsSecrets {
			if _, ok := pipelineSecrets.(data.Map)[secret.ID]; !ok {
				pipelineSecrets.(data.Map)[secret.ID] = data.NewString(*secret.Value)
			}
		}

		if err := wfm.Set(ctx, idx, constant.SegConnection, connections); err != nil {
			return preTriggerErr(fmt.Errorf("setting connections in memory: %w", err))
		}

		// Init component template
		for compID, comp := range triggerRecipe.Component {
			wfm.InitComponent(ctx, idx, compID)

			inputVal, err := data.NewValue(comp.Input)
			if err != nil {
				return preTriggerErr(fmt.Errorf("initializing pipeline input memory: %w", err))
			}
			if err := wfm.SetComponentData(ctx, idx, compID, memory.ComponentDataInput, inputVal); err != nil {
				return preTriggerErr(fmt.Errorf("initializing pipeline input memory: %w", err))
			}

			setupVal, err := data.NewValue(comp.Setup)
			if err != nil {
				return preTriggerErr(fmt.Errorf("initializing pipeline setup memory: %w", err))
			}
			if err := wfm.SetComponentData(ctx, idx, compID, memory.ComponentDataSetup, setupVal); err != nil {
				return preTriggerErr(fmt.Errorf("initializing pipeline setup memory: %w", err))
			}
		}
		output := data.Map{}

		for key, o := range triggerRecipe.Output {
			output[key] = data.NewString(o.Value)
		}
		err = wfm.SetPipelineData(ctx, idx, memory.PipelineOutputTemplate, output)
		if err != nil {
			return preTriggerErr(fmt.Errorf("initializing pipeline memory: %w", err))
		}
	}

	if wfm.IsStreaming() {
		for batchIdx := range wfm.GetBatchSize() {
			err = w.memoryStore.SendWorkflowStatusEvent(
				ctx,
				param.WorkflowID,
				memory.Event{
					Event: string(memory.PipelineStatusUpdated),
					Data: memory.PipelineStatusUpdatedEventData{
						PipelineEventData: memory.PipelineEventData{
							UpdateTime: time.Now(),
							BatchIndex: batchIdx,
							Status: map[memory.PipelineStatusType]bool{
								memory.PipelineStatusStarted:   true,
								memory.PipelineStatusErrored:   false,
								memory.PipelineStatusCompleted: false,
							},
						},
					},
				},
			)
			if err != nil {
				return preTriggerErr(fmt.Errorf("sending event: %w", err))
			}
		}
	}

	logger.Info("PreTriggerActivity completed")
	return nil
}

// PostTriggerActivity copy the trigger memory from MemoryStore to Redis.
func (w *worker) PostTriggerActivity(ctx context.Context, param *PostTriggerActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PostTriggerActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return temporal.NewApplicationErrorWithCause("loading pipeline memory", postTriggerActivityErrorType, err)
	}

	for batchIdx := range wfm.GetBatchSize() {
		output, err := wfm.GetPipelineData(ctx, batchIdx, memory.PipelineOutput)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline memory", postTriggerActivityErrorType, err)
		}
		// TODO: optimize the struct conversion
		outputStruct, err := output.ToStructValue()
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline memory", postTriggerActivityErrorType, err)
		}
		b, err := protojson.Marshal(outputStruct)
		if err != nil {
			return err
		}
		var data map[string]any
		err = json.Unmarshal(b, &data)
		if err != nil {
			return err
		}

		if wfm.IsStreaming() {
			err = w.memoryStore.SendWorkflowStatusEvent(
				ctx,
				param.WorkflowID,
				memory.Event{
					Event: string(memory.PipelineStatusUpdated),
					Data: memory.PipelineStatusUpdatedEventData{
						PipelineEventData: memory.PipelineEventData{
							UpdateTime: time.Now(),
							BatchIndex: batchIdx,
							Status: map[memory.PipelineStatusType]bool{
								memory.PipelineStatusStarted:   true,
								memory.PipelineStatusErrored:   false,
								memory.PipelineStatusCompleted: true,
							},
						},
					},
				},
			)
			if err != nil {
				return temporal.NewApplicationErrorWithCause("sending event", postTriggerActivityErrorType, err)
			}
		}
	}

	logger.Info("PostTriggerActivity completed")
	return nil
}

func (w *worker) IncreasePipelineTriggerCountActivity(ctx context.Context, sv recipe.SystemVariables) error {
	l, _ := logger.GetZapLogger(ctx)
	l = l.With(zap.Reflect("systemVariables", sv))
	l.Info("IncreasePipelineTriggerCountActivity started")

	if err := w.repository.AddPipelineRuns(ctx, sv.PipelineUID); err != nil {
		l.With(zap.Error(err)).Error("Couldn't update number of pipeline runs.")
	}

	l.Info("IncreasePipelineTriggerCountActivity completed")
	return nil
}

func (w *worker) processCondition(ctx context.Context, wfm memory.WorkflowMemory, id string, UpstreamIDs []string, condition string) (map[int]int, error) {
	conditionMap := map[int]int{}

	ptr := 0
	for idx := range wfm.GetBatchSize() {

		for _, upstreamID := range UpstreamIDs {
			if s, err := wfm.GetComponentStatus(ctx, idx, upstreamID, memory.ComponentStatusSkipped); err == nil && s {
				if err = wfm.SetComponentStatus(ctx, idx, id, memory.ComponentStatusSkipped, true); err != nil {
					return nil, err
				}
			}
			if s, err := wfm.GetComponentStatus(ctx, idx, upstreamID, memory.ComponentStatusErrored); err == nil && s {
				if err = wfm.SetComponentStatus(ctx, idx, id, memory.ComponentStatusSkipped, true); err != nil {
					return nil, err
				}
			}
		}
		if s, err := wfm.GetComponentStatus(ctx, idx, id, memory.ComponentStatusSkipped); err == nil && s {
			continue
		}

		if condition != "" {
			// TODO: these code should be refactored and shared some common functions with Render
			condStr := condition
			var varMapping map[string]string
			condStr, _, varMapping = recipe.SanitizeCondition(condStr)

			expr, err := parser.ParseExpr(condStr)
			if err != nil {
				return nil, err
			}

			allMemory, err := wfm.Get(ctx, idx, "")
			if err != nil {
				return nil, err
			}
			condMemoryForConditionStruct, err := allMemory.ToStructValue()
			if err != nil {
				return nil, err
			}
			b, _ := protojson.Marshal(condMemoryForConditionStruct)
			condMemoryForCondition := map[string]any{}
			_ = json.Unmarshal(b, &condMemoryForCondition)

			sanitizedCondMemoryForCondition := map[string]any{}
			for k, v := range condMemoryForCondition {
				sanitizedCondMemoryForCondition[varMapping[k]] = v
			}

			cond, err := recipe.EvalCondition(expr, sanitizedCondMemoryForCondition)
			if err != nil {
				return nil, err
			}

			if cond == false {
				if err = wfm.SetComponentStatus(ctx, idx, id, memory.ComponentStatusSkipped, true); err != nil {
					return nil, err
				}
			}
		}

		if s, err := wfm.GetComponentStatus(ctx, idx, id, memory.ComponentStatusSkipped); err == nil && !s {
			if err = wfm.SetComponentStatus(ctx, idx, id, memory.ComponentStatusStarted, true); err != nil {
				return nil, err
			}
			conditionMap[ptr] = idx
			ptr += 1
		}
	}
	return conditionMap, nil
}

// writeErrorDataPoint is a helper function that writes the error data point to
// the usage metrics table.
func (w *worker) writeErrorDataPoint(ctx context.Context, errs []error, span trace.Span, startTime time.Time, dataPoint *utils.PipelineUsageMetricData) {
	errStrs := make([]string, len(errs))
	for i, e := range errs {
		errStrs[i] = e.Error()
	}
	span.SetStatus(1, "workflow error")
	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtpb.Status_STATUS_ERRORED
	_ = w.writeNewDataPoint(ctx, *dataPoint)
}

// componentActivityError transforms an error with (potentially) an end-user
// message into a Temporal application error. Temporal clients can extract the
// message and propagate it to the end user.
func componentActivityError(ctx context.Context, wfm memory.WorkflowMemory, err error, errType, componentID string) error {

	if wfm == nil {
		return fmt.Errorf("workflow memory is empty")
	}

	// TODO: huitang
	// Currently, if any data in the batch has an error, we treat the entire
	// batch as errored. In the future, we should allow partial errors within a
	// batch.
	for batchIdx := range wfm.GetBatchSize() {
		if wfmErr := wfm.SetComponentStatus(ctx, batchIdx, componentID, memory.ComponentStatusErrored, true); wfmErr != nil {
			return wfmErr
		}
		if wfmErr := wfm.SetComponentErrorMessage(ctx, batchIdx, componentID, errmsg.MessageOrErr(err)); wfmErr != nil {
			return wfmErr
		}
	}

	// If no end-user message is present in the error, MessageOrErr will return
	// the string version of the error. For an end user, this extra information
	// is more actionable than no information at all.
	msg := fmt.Sprintf("Component %s failed to execute. %s", componentID, errmsg.MessageOrErr(err))
	return temporal.NewApplicationErrorWithCause(msg, errType, err)
}

// The following constants help temporal clients to trace the origin of an
// execution error. They can be leveraged to e.g. define retry policy rules.
// This may evolve in the future to values that have more to do with the
// business domain (e.g. VendorError (non billable), InputDataError (billable),
// etc.).
const (
	componentActivityErrorType    = "ComponentActivityError"
	outputActivityErrorType       = "OutputActivityError"
	preIteratorActivityErrorType  = "PreIteratorActivityError"
	postIteratorActivityErrorType = "PostIteratorActivityError"
	preTriggerActivityErrorType   = "PreTriggerActivityError"
	loadDAGDataActivityErrorType  = "LoadDAGDataActivityError"
	postTriggerActivityErrorType  = "PostTriggerActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}

func (w *worker) SchedulePipelineWorkflow(wfctx workflow.Context, param *SchedulePipelineWorkflowParam) error {

	scheduleID := fmt.Sprintf("%s_%s_schedule", param.PipelineUID, param.PipelineReleaseUID)

	// TODO: huitang - Handle pipeline release as well.
	triggerParam := &TriggerPipelineWorkflowParam{
		SystemVariables: recipe.SystemVariables{
			PipelineTriggerID:    scheduleID,
			PipelineID:           param.PipelineID,
			PipelineUID:          param.PipelineUID,
			PipelineReleaseID:    param.PipelineReleaseID,
			PipelineReleaseUID:   param.PipelineReleaseUID,
			PipelineOwnerType:    param.Namespace.NsType,
			PipelineOwnerUID:     param.Namespace.NsUID,
			PipelineUserUID:      param.Namespace.NsUID,
			PipelineRequesterUID: param.Namespace.NsUID,
			ExpiryRuleTag:        param.ExpiryRuleTag,
		},
		Mode: mgmtpb.Mode_MODE_ASYNC,
	}

	wfUID, _ := uuid.NewV4()
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue:                w.workerUID.String(),
		WorkflowID:               wfUID.String(),
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	_ = workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(wfctx, childWorkflowOptions),
		"TriggerPipelineWorkflow",
		triggerParam,
	).Get(wfctx, nil)

	return nil
}

// ClosePipelineActivity is the last step when triggering a workflow. The activity:
//   - Sends a PipelineClosed event if the trigger is streamed. If this fails,
//     the error is saved in order not to block the execution of the next step.
//   - Purges the workflow memory.
func (w *worker) ClosePipelineActivity(ctx context.Context, workflowID string) error {
	var errEvent, errPurge error
	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, workflowID)
	if err != nil {
		return err
	}

	if wfm.IsStreaming() {
		evt := memory.Event{
			Event: string(memory.PipelineClosed),
		}

		if err := w.memoryStore.SendWorkflowStatusEvent(ctx, workflowID, evt); err != nil {
			errEvent = fmt.Errorf("sending PipelineClosed event: %w", err)
		}
	}

	return errors.Join(errEvent, errPurge)
}
