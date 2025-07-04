// This file will be refactored soon
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/component/generic/scheduler/v0"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	SystemVariables recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	Recipe          *datamodel.Recipe

	Streaming bool
	Mode      mgmtpb.Mode

	// If the pipeline trigger is from an iterator, these fields will be set.
	ParentWorkflowID  *string
	ParentCompID      *string
	ParentOriginalIdx *int
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
	WorkflowID        string
	ID                string
	UpstreamIDs       []string
	ProcessedBatchIDs []int
	Type              string
	Task              string
	SystemVariables   recipe.SystemVariables

	// If the component belongs to an iterator, these fields will be set
	ParentWorkflowID  *string
	ParentCompID      *string
	ParentOriginalIdx *int
}

// ChildPipelineTriggerParams contains the information to execute a child
// pipeline trigger in the context of the execution of an iterator component.
type ChildPipelineTriggerParams struct {
	// BatchIdx refers to the element within the batch that will be triggering
	// the child pipeline executions. Some batch elements might be skipped due
	// to the component condition, so we need this info to build the whole
	// batch result later.
	BatchIdx int

	// WorkflowIDs contains the IDs of the workflows that an iterator will
	// trigger (one per element in the iterator) for a given batch element.
	WorkflowIDs []string
}

// PostIteratorActivityParam contains the parameters to wrap up an iterator
// component execution.
type PostIteratorActivityParam struct {
	WorkflowID            string
	ID                    string
	ChildPipelineTriggers []ChildPipelineTriggerParams
	OutputElements        map[string]string
	SystemVariables       recipe.SystemVariables
}

// InitComponentsActivityParam ...
type InitComponentsActivityParam struct {
	WorkflowID      string
	SystemVariables recipe.SystemVariables
	Recipe          *datamodel.Recipe
}

type UpdatePipelineRunActivityParam struct {
	PipelineTriggerID string
	PipelineRun       *datamodel.PipelineRun
}

type UpsertComponentRunActivityParam struct {
	ComponentRun *datamodel.ComponentRun
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

func (w *worker) SchedulePipelineWorkflow(wfctx workflow.Context, param *scheduler.SchedulePipelineWorkflowParam) error {
	eventName := "SchedulePipelineWorkflow"
	sCtx, span := tracer.Start(
		context.Background(),
		eventName,
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	msg := scheduleEventMessage{
		UID:         param.UID.String(),
		TriggeredAt: time.Now().Format(time.RFC3339),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	structPayload := &structpb.Struct{}
	err = protojson.Unmarshal(payload, structPayload)
	if err != nil {
		return err
	}

	_, err = w.pipelinePublicServiceClient.DispatchPipelineWebhookEvent(sCtx, &pb.DispatchPipelineWebhookEventRequest{
		WebhookType: "scheduler",
		Message:     structPayload,
	})
	if err != nil {
		return err
	}

	return nil
}

// CleanupMemoryWorkflow removes the committed workflow memory data from the
// external datastore. It is mainly meant for async triggers, where we need to
// hold de data for a while so clients can request the status of the operation.
func (w *worker) CleanupMemoryWorkflow(ctx workflow.Context, userUID uuid.UUID, workflowID string) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	})

	return workflow.ExecuteActivity(ctx, w.CleanupWorkflowMemoryActivity, userUID, workflowID).Get(ctx, nil)
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
	logger = logger.With(
		zap.String("pipelineUID", param.SystemVariables.PipelineUID.String()),
		zap.String("pipelineTriggerID", param.SystemVariables.PipelineTriggerID),
	)

	logger.Info("TriggerPipelineWorkflow started")

	// Options for activity worker
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	sessionOptions := &workflow.SessionOptions{
		CreationTimeout:  time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		ExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		HeartbeatTimeout: 2 * time.Minute,
	}

	ctx, err := workflow.CreateSession(ctx, sessionOptions)
	if err != nil {
		logger.Error("Failed to create session", zap.Error(err))
		return err
	}
	defer workflow.CompleteSession(ctx)

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	// Due to binary data types, the workflow memory data might be too large to
	// be communicated as a workflow param. For client-worker communication,
	// the data is stored in an external datastore. Then, the workflow loads
	// this in-memory. This means that the workflow history can't be replayed
	// (activities modify these in-memory structures without returning them to
	// the workflow), which implies that all the activities must be executed in
	// the same worker process.
	// TODO [INS-7456]: Remove the in-memory dependency so activities can be
	// executed by different processes. This can be achieved by loading and
	// committing the memory in every activity, or by removing the data blobs
	// from the memory and hodling only a reference that can be used to pull
	// the data.
	loadWFMParam := LoadWorkflowMemoryActivityParam{
		WorkflowID: workflowID,
		UserUID:    param.SystemVariables.PipelineUserUID,
		Streaming:  param.Streaming,
	}
	err = workflow.ExecuteActivity(ctx, w.LoadWorkflowMemoryActivity, loadWFMParam).Get(ctx, nil)
	if err != nil {
		return err
	}

	cleanupCtx, _ := workflow.NewDisconnectedContext(ctx)
	defer func() {
		err := workflow.ExecuteActivity(cleanupCtx, w.PurgeWorkflowMemoryActivity, workflowID).Get(cleanupCtx, nil)
		if err != nil {
			logger.Error("Failed to purge workflow memory", zap.Error(err))
		}
	}()

	// Iterator components are implemented as pipeline-in-pipeline triggers. In
	// such cases there are tasks we WON'T need to perform, such as sending the
	// workflow streaming events or the pipeline run data (e.g. recipe).
	isParentPipeline := param.ParentWorkflowID == nil
	if isParentPipeline {
		defer func() {
			err := workflow.ExecuteActivity(cleanupCtx, w.ClosePipelineActivity, workflowID).Get(cleanupCtx, nil)
			if err != nil {
				logger.Error("Failed to clean up trigger workflow", zap.Error(err))
			}
		}()

		uploadParam := UploadRecipeToMinIOParam{
			Recipe: param.Recipe,
			Metadata: MinIOUploadMetadata{
				UserUID:           param.SystemVariables.PipelineUserUID,
				PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
				ExpiryRuleTag:     param.SystemVariables.ExpiryRule.Tag,
			},
		}
		err := workflow.ExecuteActivity(ctx, w.UploadRecipeToMinIOActivity, uploadParam).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to upload pipeline run recipe", zap.Error(err))
		}

		if err := workflow.ExecuteActivity(ctx, w.InitComponentsActivity, &InitComponentsActivityParam{
			WorkflowID:      workflowID,
			SystemVariables: param.SystemVariables,
			Recipe:          param.Recipe,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if param.Streaming {
			if err := workflow.ExecuteActivity(ctx, w.SendStartedEventActivity, workflowID).Get(ctx, nil); err != nil {
				return err
			}
		}
	}

	dag, err := recipe.GenerateDAG(param.Recipe.Component)
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
groupLoop:
	for group := range orderedComp {
		futures := []workflow.Future{}
		futureArgs := []*ComponentActivityParam{}
		for compID, comp := range orderedComp[group] {
			upstreamIDs := dag.GetUpstreamCompIDs(compID)

			var processedBatchIDs []int
			err := workflow.ExecuteActivity(ctx, w.ProcessBatchConditionsActivity, ProcessBatchConditionsActivityParam{
				WorkflowID:  workflowID,
				ComponentID: compID,
				Condition:   comp.Condition,
				UpstreamIDs: upstreamIDs,
			}).Get(ctx, &processedBatchIDs)
			if err != nil {
				return err
			}

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
					WorkflowID:        workflowID,
					ID:                compID,
					UpstreamIDs:       upstreamIDs,
					ProcessedBatchIDs: processedBatchIDs,
					Type:              comp.Type,
					Task:              comp.Task,
					SystemVariables:   param.SystemVariables,
					ParentWorkflowID:  param.ParentWorkflowID,
					ParentCompID:      param.ParentCompID,
					ParentOriginalIdx: param.ParentOriginalIdx,
				}

				futures = append(futures, workflow.ExecuteActivity(ctx, w.ComponentActivity, args))
				futureArgs = append(futureArgs, args)

			case datamodel.Iterator:
				// TODO: support intermediate result streaming for Iterator

				iteratorRecipe := &datamodel.Recipe{
					Component: param.Recipe.Component[compID].Component,
				}

				childTriggers := make([]ChildPipelineTriggerParams, 0, len(processedBatchIDs))
				itFutures := []workflow.Future{}
				for _, processedBatchIdx := range processedBatchIDs {
					var childTrigger ChildPipelineTriggerParams

					err := workflow.ExecuteActivity(ctx, w.PreIteratorActivity, &PreIteratorActivityParam{
						WorkflowID:  workflowID,
						ID:          compID,
						UpstreamIDs: upstreamIDs,
						BatchIdx:    processedBatchIdx,
						Input: func(c *datamodel.Component) string {
							if c.Input != nil {
								return c.Input.(string)
							}
							return ""
						}(comp),
						Range:           comp.Range,
						Index:           comp.Index,
						SystemVariables: param.SystemVariables,
						IteratorRecipe:  iteratorRecipe,
					}).Get(ctx, &childTrigger)
					if err != nil {
						errs = append(errs, err)
						continue groupLoop
					}

					childTriggers = append(childTriggers, childTrigger)
					for _, childWorkflowID := range childTrigger.WorkflowIDs {
						defer func() {
							err := workflow.ExecuteActivity(
								cleanupCtx,
								w.CleanupWorkflowMemoryActivity,
								param.SystemVariables.PipelineUserUID,
								childWorkflowID,
							).Get(cleanupCtx, nil)
							if err != nil {
								// This isn't considered an error as the workflow
								// memory might not exist at this point. E.g., if a
								// failure occurred before the data was committed.
								logger.Info("Failed to clean up child trigger workflow", zap.Error(err))
							}
						}()

						childWorkflowOptions := workflow.ChildWorkflowOptions{
							TaskQueue:                TaskQueue,
							WorkflowID:               childWorkflowID,
							WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
							RetryPolicy: &temporal.RetryPolicy{
								MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
							},
						}

						itFutures = append(itFutures, workflow.ExecuteChildWorkflow(
							workflow.WithChildOptions(ctx, childWorkflowOptions),
							w.TriggerPipelineWorkflow,
							&TriggerPipelineWorkflowParam{
								SystemVariables:   param.SystemVariables,
								Mode:              mgmtpb.Mode_MODE_SYNC,
								Recipe:            iteratorRecipe,
								ParentWorkflowID:  &workflowID,
								ParentCompID:      &compID,
								ParentOriginalIdx: &childTrigger.BatchIdx,
								Streaming:         param.Streaming,
							},
						))
					}
				}

				for _, itFuture := range itFutures {
					err = itFuture.Get(ctx, nil)
					if err != nil {
						errs = append(errs, err)
						continue
					}
				}

				if err = workflow.ExecuteActivity(ctx, w.PostIteratorActivity, &PostIteratorActivityParam{
					WorkflowID:            workflowID,
					ID:                    compID,
					ChildPipelineTriggers: childTriggers,
					OutputElements:        comp.OutputElements,
					SystemVariables:       param.SystemVariables,
				}).Get(ctx, nil); err != nil {
					errs = append(errs, err)
					continue
				}
			}
		}

		for idx, future := range futures {
			err = future.Get(ctx, nil)
			if err != nil {
				componentRunFailed = true
				componentRunErrors = append(componentRunErrors, fmt.Sprintf("component(ID: %s) run failed", futureArgs[idx].ID))
				errs = append(errs, err)

				continue
			}
			componentRunFutures = append(componentRunFutures, workflow.ExecuteActivity(ctx, w.UploadComponentOutputsActivity, futureArgs[idx]))
		}

		for idx := range futures {
			// There is time difference between the workflow memory update and upload component inputs activity.
			// If we upload the inputs before the component activity, some of the input will not be set in the workflow memory.
			// So, we have to execute this worker activity after the component activity.
			componentRunFutures = append(componentRunFutures, workflow.ExecuteActivity(ctx, w.UploadComponentInputsActivity, futureArgs[idx]))
		}
	}

	duration := time.Since(startTime)
	if isParentPipeline {
		if err := workflow.ExecuteActivity(ctx, w.OutputActivity, &ComponentActivityParam{
			WorkflowID:      workflowID,
			SystemVariables: param.SystemVariables,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if err := workflow.ExecuteActivity(ctx, w.UploadOutputsToMinIOActivity, &MinIOUploadMetadata{
			UserUID:           param.SystemVariables.PipelineUserUID,
			PipelineTriggerID: workflowID,
			ExpiryRuleTag:     param.SystemVariables.ExpiryRule.Tag,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if param.Streaming {
			if err := workflow.ExecuteActivity(ctx, w.SendCompletedEventActivity, workflowID).Get(ctx, nil); err != nil {
				return err
			}
		}

		// TODO: we should check whether to collect failed component or not
		if err := workflow.ExecuteActivity(ctx, w.IncreasePipelineTriggerCountActivity, param.SystemVariables).Get(ctx, nil); err != nil {
			return err
		}

		dataPoint := w.pipelineTriggerDataPoint(workflowID, param.SystemVariables, param.Mode)
		dataPoint.TriggerTime = startTime.Format(time.RFC3339Nano)
		dataPoint.ComputeTimeDuration = duration.Seconds()
		dataPoint.Status = mgmtpb.Status_STATUS_COMPLETED

		if len(errs) > 0 {
			span.SetStatus(1, "workflow error")
			dataPoint.Status = mgmtpb.Status_STATUS_ERRORED
		}

		if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
			logger.Warn(err.Error())
		}
	}

	if err := workflow.ExecuteActivity(ctx, w.CommitWorkflowMemoryActivity, workflowID, param.SystemVariables).Get(ctx, nil); err != nil {
		return err
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
		// If a component has failed, we consider the whole pipeline as failed.
		// TODO jvallesm: this is a simplistic approach we might want to
		// challenge in the future. E.g., a pipeline might be designed so a
		// component runs when another one fails. We need to provide a
		// mechanism to consider this scenario as a completed trigger.
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
	}

	defer func() {
		componentRun := &datamodel.ComponentRun{
			Status:        datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED),
			CompletedTime: null.TimeFrom(time.Now()),
			TotalDuration: null.IntFrom(time.Since(startTime).Milliseconds()),
		}

		if err != nil {
			componentRun.Status = datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED)
			componentRun.Error = null.StringFrom(err.Error())
		}

		err = w.repository.UpdateComponentRun(ctx, param.SystemVariables.PipelineTriggerID, param.ID, componentRun)
		if err != nil {
			logger.Error("failed to log component run end time", zap.Error(err))
		}
	}()

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
	}

	if len(param.ProcessedBatchIDs) == 0 {
		return nil
	}

	sr := &setupReader{
		memoryStore:       w.memoryStore,
		workflowID:        param.WorkflowID,
		compID:            param.ID,
		processedBatchIDs: param.ProcessedBatchIDs,
	}
	setups, err := sr.Read(ctx)
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

	jobs := make([]*componentbase.Job, len(param.ProcessedBatchIDs))
	for idx, originalIdx := range param.ProcessedBatchIDs {
		jobs[idx] = &componentbase.Job{
			Input:  newInputReader(w.memoryStore, param.WorkflowID, param.ID, originalIdx, w.binaryFetcher),
			Output: newOutputWriter(w.memoryStore, param.WorkflowID, param.ID, originalIdx, wfm.IsStreaming()),
			Error:  newErrorHandler(w.memoryStore, param.WorkflowID, param.ID, originalIdx, param.ParentWorkflowID, param.ParentCompID, param.ParentOriginalIdx),
		}
	}
	err = execution.Execute(
		ctx,
		jobs,
	)
	if err != nil {
		return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
	}

	isFailedExecution := false
	for _, idx := range param.ProcessedBatchIDs {
		isFailedExecution, err = wfm.GetComponentStatus(ctx, idx, param.ID, memory.ComponentStatusErrored)
		if err != nil {
			err = fmt.Errorf("checking component execution error: %w", err)
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
		}

		// If any of the jobs failed, we consider the component failed to
		// execute and return an error.
		// TODO jvallesm: in the future we might not want to break the
		// execution and detect only if an element in the batch has failed.
		if isFailedExecution {
			// The batch element will contain an error message in memory. We
			// use it as the component activity error.
			var msg string
			msg, err = wfm.GetComponentErrorMessage(ctx, idx, param.ID)
			if err != nil {
				err = fmt.Errorf("extracting component execution error: %w", err)
				return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
			}

			err = fmt.Errorf("%s", msg)
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
		}

		if err := wfm.SetComponentStatus(ctx, idx, param.ID, memory.ComponentStatusCompleted, true); err != nil {
			return componentActivityError(ctx, wfm, err, componentActivityErrorType, param.ID)
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

		updatedOutput := w.uploadFileAndReplaceWithURL(ctx, param, &output)

		err = wfm.SetPipelineData(ctx, idx, memory.PipelineOutput, updatedOutput)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline output", outputActivityErrorType, err)
		}
	}

	logger.Info("OutputActivity completed")
	return nil
}

// ProcessBatchConditionsActivityParam ...
type ProcessBatchConditionsActivityParam struct {
	WorkflowID  string
	ComponentID string
	Condition   string
	UpstreamIDs []string
}

// ProcessBatchConditionsActivity computes the batch IDs for which a component
// should be executed.
func (w *worker) ProcessBatchConditionsActivity(ctx context.Context, param ProcessBatchConditionsActivityParam) ([]int, error) {
	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		err := fmt.Errorf("fetching workflow memory: %w", err)
		return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
	}

	processedIDs := make([]int, 0, wfm.GetBatchSize())
	for idx := range wfm.GetBatchSize() {
		for _, upstreamID := range param.UpstreamIDs {
			if s, err := wfm.GetComponentStatus(ctx, idx, upstreamID, memory.ComponentStatusSkipped); err == nil && s {
				if err := wfm.SetComponentStatus(ctx, idx, param.ComponentID, memory.ComponentStatusSkipped, true); err != nil {
					return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
				}
			}
			if s, err := wfm.GetComponentStatus(ctx, idx, upstreamID, memory.ComponentStatusErrored); err == nil && s {
				if err := wfm.SetComponentStatus(ctx, idx, param.ComponentID, memory.ComponentStatusSkipped, true); err != nil {
					return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
				}
			}
		}
		if s, err := wfm.GetComponentStatus(ctx, idx, param.ComponentID, memory.ComponentStatusSkipped); err == nil && s {
			continue
		}

		if param.Condition != "" {
			allMemory, err := wfm.Get(ctx, idx, "")
			if err != nil {
				err := fmt.Errorf("fetching memory: %w", err)
				return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
			}

			cond, err := recipe.Eval(param.Condition, allMemory)
			if err != nil {
				err := fmt.Errorf("evaluating param.Condition: %w", err)
				return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
			}

			if cond == false {
				if err = wfm.SetComponentStatus(ctx, idx, param.ComponentID, memory.ComponentStatusSkipped, true); err != nil {
					return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
				}
			}
		}

		if s, err := wfm.GetComponentStatus(ctx, idx, param.ComponentID, memory.ComponentStatusSkipped); err == nil && !s {
			if err = wfm.SetComponentStatus(ctx, idx, param.ComponentID, memory.ComponentStatusStarted, true); err != nil {
				return nil, componentActivityError(ctx, wfm, err, processBatchConditionsActivityErrorType, param.ComponentID)
			}
			processedIDs = append(processedIDs, idx)
		}
	}

	return processedIDs, nil
}

// PreIteratorActivityParam ...
type PreIteratorActivityParam struct {
	WorkflowID      string
	ID              string
	UpstreamIDs     []string
	BatchIdx        int
	Input           string
	Range           any
	Index           string
	SystemVariables recipe.SystemVariables
	IteratorRecipe  *datamodel.Recipe
}

// iteratorComponentData is used to hold the component data in an iterator
// element before building its workflow memory.
type iteratorComponentData struct {
	input format.Value
	setup format.Value
}

// PreIteratorActivity generates the workflow memory for each element in an
// iteration. In order to execute iterator components concurrently, each
// element in the iterator triggers a TriggerPipelineWorkflow.
func (w *worker) PreIteratorActivity(ctx context.Context, param PreIteratorActivityParam) (*ChildPipelineTriggerParams, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreIteratorActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}

	if err = wfm.SetComponentStatus(ctx, param.BatchIdx, param.ID, memory.ComponentStatusStarted, true); err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}

	baseWorkflowID := fmt.Sprintf("%s:%d:%s:%s:%s", param.WorkflowID, param.BatchIdx, constant.SegComponent, param.ID, constant.SegIteration)

	// If `input` is provided, the iteration will be performed over it;
	// otherwise, the iteration will be based on the `range` setup.
	useInput := param.Input != ""

	var indexes []int
	var elems []format.Value
	if useInput {
		input, err := recipe.Render(ctx, data.NewString(param.Input), param.BatchIdx, wfm, false)
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

		renderedRangeParam, err := recipe.Render(ctx, rangeParam, param.BatchIdx, wfm, false)
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

	// Get common workflow memory data.
	variable, err := wfm.Get(ctx, param.BatchIdx, constant.SegVariable)
	if err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}
	secret, err := wfm.Get(ctx, param.BatchIdx, constant.SegSecret)
	if err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}
	connection, err := wfm.Get(ctx, param.BatchIdx, constant.SegConnection)
	if err != nil {
		return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
	}

	upstreamComponents := map[string]format.Value{}
	for _, id := range param.UpstreamIDs {
		component, err := wfm.Get(ctx, param.BatchIdx, id)
		if err != nil {
			err := fmt.Errorf("fetching upstream component data: %w", err)
			return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
		}

		upstreamComponents[id] = component
	}

	iteratorComponents := map[string]iteratorComponentData{}
	for compID, comp := range param.IteratorRecipe.Component {
		input, err := data.NewValue(comp.Input)
		if err != nil {
			err := fmt.Errorf("converting component input to format.Value: %w", err)
			return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
		}
		setup, err := data.NewValue(comp.Setup)
		if err != nil {
			err := fmt.Errorf("converting component setup to format.Value: %w", err)
			return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
		}

		iteratorComponents[compID] = iteratorComponentData{
			input: input,
			setup: setup,
		}
	}

	// Each element in the iterator generates a workflow memory (and a
	// triggered workflow). Therefore, batch size is 1 and we'll set and access
	// the data at the 0 index.
	// The following function extracts the common code to initialize, commit
	// and purge the workflow memory of an element.
	commitChildWFM := func(e, rangeIndex int) (workflowID string, err error) {
		elemWFID := fmt.Sprintf("%s:%d", baseWorkflowID, e)
		elemWFM, err := w.memoryStore.NewWorkflowMemory(ctx, elemWFID, 1)
		if err != nil {
			return "", fmt.Errorf("initializing workflow memory: %w", err)
		}

		defer w.memoryStore.PurgeWorkflowMemory(elemWFID)

		// Set input.
		var key string
		var elem format.Value

		// When iterating over `input`, each element in the array is
		// processed and stored in memory.
		if useInput {
			elem = data.Map{"element": elems[e]}
			key = param.ID
		} else {
			elem = data.NewNumberFromInteger(rangeIndex)

			key = param.Index
			if key == "" {
				key = defaultRangeIdentifier
			}
		}

		if err := elemWFM.Set(ctx, 0, key, elem); err != nil {
			return "", fmt.Errorf("setting element input in workflow memory: %w", err)
		}

		// The following code doesn't depend on the iterator element.
		// Therefore, we could squeeze some performance by generating a
		// single workflow memory and committing it with different IDs.
		// However, this would mean exposing internal fields of the workflow
		// memory like the ID and potentially misusing this field. For now, the
		// performance gain of having one workflow per iterator element is
		// enough.

		// Set pipeline data
		if err := elemWFM.SetPipelineData(ctx, 0, memory.PipelineVariable, variable); err != nil {
			return "", fmt.Errorf("setting variable in workflow memory: %w", err)
		}
		if err := elemWFM.SetPipelineData(ctx, 0, memory.PipelineSecret, secret); err != nil {
			return "", fmt.Errorf("setting secret in workflow memory: %w", err)
		}
		if err := elemWFM.SetPipelineData(ctx, 0, memory.PipelineConnection, connection); err != nil {
			return "", fmt.Errorf("setting connection in workflow memory: %w", err)
		}

		for id, component := range upstreamComponents {
			if err := elemWFM.Set(ctx, 0, id, component); err != nil {
				return "", fmt.Errorf("setting upstream component data: %w", err)
			}
		}

		for compID, compData := range iteratorComponents {
			elemWFM.InitComponent(ctx, 0, compID)

			inputVal := setIteratorIndex(compData.input, param.Index, rangeIndex)
			if err := elemWFM.SetComponentData(ctx, 0, compID, memory.ComponentDataInputTemplate, inputVal); err != nil {
				return "", fmt.Errorf("setting component input: %w", err)
			}

			if err := elemWFM.SetComponentData(ctx, 0, compID, memory.ComponentDataSetupTemplate, compData.setup); err != nil {
				return "", fmt.Errorf("setting component setup: %w", err)
			}
		}

		if err := w.memoryStore.CommitWorkflowData(ctx, param.SystemVariables.PipelineUserUID, elemWFM); err != nil {
			return "", fmt.Errorf("committing workflow memory: %w", err)
		}

		return elemWFID, nil
	}

	childWorkflowIDs := make([]string, len(indexes))
	for e, rangeIndex := range indexes {
		elemWFID, err := commitChildWFM(e, rangeIndex)
		if err != nil {
			return nil, componentActivityError(ctx, wfm, err, preIteratorActivityErrorType, param.ID)
		}

		childWorkflowIDs[e] = elemWFID
	}

	logger.Info("PreIteratorActivity completed")
	return &ChildPipelineTriggerParams{
		BatchIdx:    param.BatchIdx,
		WorkflowIDs: childWorkflowIDs,
	}, nil
}

// PostIteratorActivity merges the trigger memory from each iteration.
func (w *worker) PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PostIteratorActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
	}

	for _, childTrigger := range param.ChildPipelineTriggers {
		output := data.Map{}

		for _, childWorkflowID := range childTrigger.WorkflowIDs {
			// The fetched child workflow memory doesn't contain the streaming
			// flag. If we used it in this activity, we'd need to receive it as
			// a param and set it after fetching it.
			childWFM, err := w.memoryStore.FetchWorkflowMemory(ctx, param.SystemVariables.PipelineUserUID, childWorkflowID)
			if err != nil {
				err := fmt.Errorf("fetching workflow memory: %w", err)
				return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
			}

			defer w.memoryStore.PurgeWorkflowMemory(childWorkflowID)

			errored, err := wfm.GetComponentStatus(ctx, 0, param.ID, memory.ComponentStatusErrored)
			if err != nil {
				return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
			}

			// If one element in the iteration has failed, we consider the
			// component as failed and we won't build the output.
			if errored {
				return nil
			}

			for k, v := range param.OutputElements {
				if _, hasValues := output[k]; !hasValues {
					output[k] = make(data.Array, 0, len(childTrigger.WorkflowIDs))
				}

				elemVals := output[k].(data.Array)

				elemVal, err := recipe.Render(ctx, data.NewString(v), 0, childWFM, false)
				if err != nil {
					return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
				}

				elemVals = append(elemVals, elemVal)
				output[k] = elemVals
			}
		}

		if err = wfm.SetComponentData(ctx, childTrigger.BatchIdx, param.ID, memory.ComponentDataOutput, output); err != nil {
			return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
		}

		if err = wfm.SetComponentStatus(ctx, childTrigger.BatchIdx, param.ID, memory.ComponentStatusCompleted, true); err != nil {
			return componentActivityError(ctx, wfm, err, postIteratorActivityErrorType, param.ID)
		}
	}

	logger.Info("PostIteratorActivity completed")
	return nil
}

// preTriggerErr returns a function that handles errors that happen during the
// trigger workflow setup, i.e., before the components start to be executed.
// If the trigger is streamed, it will send an event to halt the execution.
func (w *worker) preTriggerErr(ctx context.Context, workflowID string, wfm *memory.WorkflowMemory) func(error) error {
	return func(err error) error {
		if msg := errmsg.Message(err); msg != "" {
			err = temporal.NewApplicationErrorWithCause(msg, preTriggerErrorType, err)
		}

		if !wfm.IsStreaming() {
			return err
		}

		updateTime := time.Now()
		for batchIdx := range wfm.GetBatchSize() {
			if err := w.memoryStore.SendWorkflowStatusEvent(
				ctx,
				workflowID,
				pubsub.Event{
					Name: string(memory.PipelineStatusUpdated),
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

		return err
	}
}

func (w *worker) fetchConnectionAsValue(ctx context.Context, requesterUID uuid.UUID, connectionID string) (format.Value, error) {
	conn, err := w.repository.GetNamespaceConnectionByID(ctx, requesterUID, connectionID)
	if err != nil {
		if errors.Is(err, errdomain.ErrNotFound) {
			return nil, errmsg.AddMessage(err, fmt.Sprintf("Connection %s doesn't exist.", connectionID))
		}

		return nil, fmt.Errorf("fetching connection: %w", err)
	}

	var setup map[string]any
	if err := json.Unmarshal(conn.Setup, &setup); err != nil {
		return nil, fmt.Errorf("unmarshaling connection setup: %w", err)
	}

	v, err := data.NewValue(setup)
	if err != nil {
		return nil, fmt.Errorf("transforming connection setup to value: %w", err)
	}

	return v, nil
}

// loadConnectionFromComponent looks for a connection references in a component
// and, when one is found, fetches the connection from the requester's
// namespace and loads it to the connection map.
func (w *worker) loadConnectionFromComponent(
	ctx context.Context,
	requesterUID uuid.UUID,
	component *datamodel.Component,
	connections data.Map,
) error {
	// We're only looking for connection references, so we skip components
	// whose setup is defined explicitly in the recipe.
	connRef, hasConnRef := component.Setup.(string)
	if !hasConnRef {
		return nil
	}

	connID, err := recipe.ConnectionIDFromReference(connRef)
	if err != nil {
		return fmt.Errorf("resolving connection reference: %w", err)
	}

	if _, connAlreadyLoaded := connections[connID]; connAlreadyLoaded {
		return nil
	}

	conn, err := w.fetchConnectionAsValue(ctx, requesterUID, connID)
	if err != nil {
		if !errors.Is(err, errdomain.ErrNotFound) {
			return err
		}

		// The connection ID might not exist in the requester's namespace, but
		// they can still provide it it in the trigger params.
		conn = data.NewNull()
	}

	connections[connID] = conn
	return nil
}

// mergeInputConnections returns the connections that will be used in an
// execution batch. If the trigger data references a connection in that batch,
// the connection value is overwritten.
func (w *worker) mergeInputConnections(
	ctx context.Context,
	wfm *memory.WorkflowMemory,
	idx int,
	requesterUID uuid.UUID,
	pipelineConnections data.Map,
	inputConnections data.Map,
) (data.Map, error) {
	connRefsInMem, err := wfm.Get(ctx, idx, constant.SegConnection)
	if err != nil {
		return nil, fmt.Errorf("loading pipeline connection memory: %w", err)
	}

	connRefs, ok := connRefsInMem.(data.Map)
	if !ok {
		return nil, fmt.Errorf("invalid connection references in batch memory")
	}

	batchConns := data.Map{}
	for connID, conn := range pipelineConnections {
		ref, override := connRefs[connID]
		if !override {
			// The connection isn't referenced in the trigger data, so the
			// connection referenced in the recipe must exist in the
			// requester's namespace.
			if conn.Equal(data.NewNull()) {
				return nil, errmsg.AddMessage(
					fmt.Errorf("connection doesn't exist"),
					fmt.Sprintf("Connection %s doesn't exist.", connID),
				)
			}

			batchConns[connID] = conn
			continue
		}

		// Fetch referenced connection and override current value.
		inputConnID := ref.String()
		inputConn, alreadyFetched := inputConnections[inputConnID]
		if !alreadyFetched {
			inputConn, err = w.fetchConnectionAsValue(ctx, requesterUID, inputConnID)
			if err != nil {
				return nil, err
			}

			// Cache connection in case other batches reference it, too.
			inputConnections[inputConnID] = inputConn
		}

		batchConns[connID] = inputConn
	}

	return batchConns, nil
}

// InitComponentsActivity sets up the component information and loads it into
// memory.
//   - Secrets and connections are resolved and loaded into memory.
//   - Initializes the pipeline template, wiring the pipeline and component input
//     and outputs.
func (w *worker) InitComponentsActivity(ctx context.Context, param *InitComponentsActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("InitComponentsActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return fmt.Errorf("loading pipeline memory: %w", err)
	}

	handleErr := w.preTriggerErr(ctx, param.WorkflowID, wfm)

	// Load secrets and connections
	pt := ""
	var nsSecrets []*datamodel.Secret
	ownerPermalink := fmt.Sprintf("%s/%s", param.SystemVariables.PipelineOwner.NsType, param.SystemVariables.PipelineOwner.NsUID)
	for {
		var secrets []*datamodel.Secret
		secrets, _, pt, err = w.repository.ListNamespaceSecrets(ctx, ownerPermalink, 100, pt, filtering.Filter{})
		if err != nil {
			return handleErr(fmt.Errorf("loading pipeline secret memory: %w", err))
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

	requesterUID := param.SystemVariables.PipelineRequesterUID

	// inputConns will contain the connections referenced in the trigger data,
	// as several batches might reference the same connection.
	inputConns := data.Map{}
	connections := data.Map{}
	for _, comp := range param.Recipe.Component {
		if err := w.loadConnectionFromComponent(ctx, requesterUID, comp, connections); err != nil {
			return handleErr(fmt.Errorf("loading connections: %w", err))
		}

		if comp.Type == datamodel.Iterator {
			for _, nestedComp := range comp.Component {
				if err := w.loadConnectionFromComponent(ctx, requesterUID, nestedComp, connections); err != nil {
					return handleErr(fmt.Errorf("loading connections: %w", err))
				}
			}
		}
	}

	// Secrets may be overwritten per batch, so we need to resolve them within
	// the batch loop.
	for idx := range wfm.GetBatchSize() {
		pipelineSecrets, err := wfm.Get(ctx, idx, constant.SegSecret)
		if err != nil {
			return handleErr(fmt.Errorf("loading pipeline secret memory: %w", err))
		}

		for _, secret := range nsSecrets {
			if _, ok := pipelineSecrets.(data.Map)[secret.ID]; !ok {
				pipelineSecrets.(data.Map)[secret.ID] = data.NewString(*secret.Value)
			}
		}

		batchConns, err := w.mergeInputConnections(ctx, wfm, idx, requesterUID, connections, inputConns)
		if err != nil {
			return handleErr(fmt.Errorf("reading connections from trigger data: %w", err))
		}

		if err := wfm.Set(ctx, idx, constant.SegConnection, batchConns); err != nil {
			return handleErr(fmt.Errorf("setting connections in memory: %w", err))
		}

		// Init component template.
		for compID, comp := range param.Recipe.Component {
			wfm.InitComponent(ctx, idx, compID)

			inputVal, err := data.NewValue(comp.Input)
			if err != nil {
				return handleErr(fmt.Errorf("initializing pipeline input memory: %w", err))
			}
			if err := wfm.SetComponentData(ctx, idx, compID, memory.ComponentDataInputTemplate, inputVal); err != nil {
				return handleErr(fmt.Errorf("initializing pipeline input memory: %w", err))
			}

			setupVal, err := data.NewValue(comp.Setup)
			if err != nil {
				return handleErr(fmt.Errorf("initializing pipeline setup memory: %w", err))
			}
			if err := wfm.SetComponentData(ctx, idx, compID, memory.ComponentDataSetupTemplate, setupVal); err != nil {
				return handleErr(fmt.Errorf("initializing pipeline setup memory: %w", err))
			}
		}
		output := data.Map{}

		// Init pipeline output template.
		for key, o := range param.Recipe.Output {
			output[key] = data.NewString(o.Value)
		}
		err = wfm.SetPipelineData(ctx, idx, memory.PipelineOutputTemplate, output)
		if err != nil {
			return handleErr(fmt.Errorf("initializing pipeline memory: %w", err))
		}
	}

	logger.Info("InitComponentsActivity completed")
	return nil
}

func (w *worker) SendStartedEventActivity(ctx context.Context, workflowID string) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("SendStartedEventActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("loading pipeline memory: %w", err)
	}

	handleErr := w.preTriggerErr(ctx, workflowID, wfm)
	for batchIdx := range wfm.GetBatchSize() {
		err = w.memoryStore.SendWorkflowStatusEvent(
			ctx,
			workflowID,
			pubsub.Event{
				Name: string(memory.PipelineStatusUpdated),
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
			return handleErr(fmt.Errorf("sending event: %w", err))
		}
	}

	logger.Info("SendStartedEventActivity completed")
	return nil
}

// SendCompletedEventActivity sends a pipeline update event with the pipeline
// completion.
func (w *worker) SendCompletedEventActivity(ctx context.Context, workflowID string) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("SendCompletedEventActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, workflowID)
	if err != nil {
		return temporal.NewApplicationErrorWithCause("loading pipeline memory", sendCompletedEventActivityErrorType, err)
	}

	for batchIdx := range wfm.GetBatchSize() {
		output, err := wfm.GetPipelineData(ctx, batchIdx, memory.PipelineOutput)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline memory", sendCompletedEventActivityErrorType, err)
		}
		// TODO: optimize the struct conversion
		outputStruct, err := output.ToStructValue()
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline memory", sendCompletedEventActivityErrorType, err)
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

		err = w.memoryStore.SendWorkflowStatusEvent(
			ctx,
			workflowID,
			pubsub.Event{
				Name: string(memory.PipelineStatusUpdated),
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
			return temporal.NewApplicationErrorWithCause("sending event", sendCompletedEventActivityErrorType, err)
		}
	}

	logger.Info("SendCompletedEventActivity completed")
	return nil
}

// CommitWorkflowMemoryActivity stores the workflow memory data in an external
// datastore.
func (w *worker) CommitWorkflowMemoryActivity(ctx context.Context, workflowID string, sysVars recipe.SystemVariables) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("CommitWorkflowMemoryActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, workflowID)
	if err != nil {
		return temporal.NewApplicationErrorWithCause("loading pipeline memory", commitWorkflowMemoryActivityErrorType, err)
	}

	if err := w.memoryStore.CommitWorkflowData(ctx, sysVars.PipelineUserUID, wfm); err != nil {
		return temporal.NewApplicationErrorWithCause("committing workflow memory", commitWorkflowMemoryActivityErrorType, err)
	}

	logger.Info("CommitWorkflowMemoryActivity completed")
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

var nsTypeToOwnerType = map[resource.NamespaceType]mgmtpb.OwnerType{
	resource.Organization: mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION,
	resource.User:         mgmtpb.OwnerType_OWNER_TYPE_USER,
}

func (w *worker) pipelineTriggerDataPoint(workflowID string, sysVars recipe.SystemVariables, triggerMode mgmtpb.Mode) utils.PipelineUsageMetricData {
	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           sysVars.PipelineOwner.NsUID.String(),
		OwnerType:          nsTypeToOwnerType[sysVars.PipelineOwner.NsType],
		UserUID:            sysVars.PipelineUserUID.String(),
		UserType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
		RequesterUID:       sysVars.PipelineRequesterUID.String(),
		RequesterType:      mgmtpb.OwnerType_OWNER_TYPE_USER,
		TriggerMode:        triggerMode,
		PipelineID:         sysVars.PipelineID,
		PipelineUID:        sysVars.PipelineUID.String(),
		PipelineReleaseID:  sysVars.PipelineReleaseID,
		PipelineReleaseUID: sysVars.PipelineReleaseUID.String(),
		PipelineTriggerUID: workflowID,
	}

	// This is a simplistic check that relies on the only supported
	// namespace switch (user->organization). If other types of impersonation
	// are supported, the requester type should be provided in the system
	// variables.
	if dataPoint.UserUID != dataPoint.RequesterUID {
		dataPoint.RequesterType = mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION
	}

	return dataPoint
}

// componentActivityError transforms an error with (potentially) an end-user
// message into a Temporal application error. Temporal clients can extract the
// message and propagate it to the end user.
func componentActivityError(ctx context.Context, wfm *memory.WorkflowMemory, err error, errType, componentID string) error {
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
	preTriggerErrorType                     = "PreTriggerError"
	commitWorkflowMemoryActivityErrorType   = "CommitWorkflowMemoryActivity"
	componentActivityErrorType              = "ComponentActivityError"
	outputActivityErrorType                 = "OutputActivityError"
	postIteratorActivityErrorType           = "PostIteratorActivityError"
	preIteratorActivityErrorType            = "PreIteratorActivityError"
	processBatchConditionsActivityErrorType = "ProcessBatchConditionsActivityError"
	sendCompletedEventActivityErrorType     = "SendCompletedEventActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}

type scheduleEventMessage struct {
	UID         string `json:"uid"`
	TriggeredAt string `json:"triggered-at"`
}

// ClosePipelineActivity sends a PipelineClosed event if the trigger is
// streamed.
func (w *worker) ClosePipelineActivity(ctx context.Context, workflowID string) error {
	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, workflowID)
	if err != nil {
		return err
	}

	if !wfm.IsStreaming() {
		return nil
	}

	if err := w.memoryStore.SendWorkflowStatusEvent(ctx, workflowID, pubsub.Event{
		Name: string(memory.PipelineClosed),
	}); err != nil {
		return fmt.Errorf("sending PipelineClosed event: %w", err)
	}

	return nil
}

// LoadWorkflowMemoryActivityParam ...
type LoadWorkflowMemoryActivityParam struct {
	WorkflowID string
	UserUID    uuid.UUID
	Streaming  bool
}

// LoadWorkflowMemoryActivity fetches the workflow memory from an external
// datastore and loads it into the memory store.
func (w *worker) LoadWorkflowMemoryActivity(ctx context.Context, param LoadWorkflowMemoryActivityParam) error {
	wfm, err := w.memoryStore.FetchWorkflowMemory(ctx, param.UserUID, param.WorkflowID)
	if err != nil {
		return fmt.Errorf("fetching workflow memory: %w", err)
	}

	if param.Streaming {
		wfm.EnableStreaming()
	}

	return nil
}

// PurgeWorkflowMemoryActivity purges the workflow data from memory. We need an
// activity for this (instead of running it in the workflow because it is the
// activities (through worker sessions) who rely on the shared memory.
func (w *worker) PurgeWorkflowMemoryActivity(_ context.Context, workflowID string) error {
	w.memoryStore.PurgeWorkflowMemory(workflowID)
	return nil
}

// CleanupWorkflowMemoryActivity removes the workflow data from memory and from
// the external datastore. This is used for workflow data that won't be needed
// anymore (e.g. for child workflows executed within an iterator).
func (w *worker) CleanupWorkflowMemoryActivity(ctx context.Context, userUID uuid.UUID, workflowID string) error {
	return w.memoryStore.CleanupWorkflowMemory(ctx, userUID, workflowID)
}
