// This file will be refactored soon
package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"time"

	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/guregu/null.v4"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	"github.com/instill-ai/x/errmsg"

	componentstore "github.com/instill-ai/component/store"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	SystemVariables recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	Mode            mgmtpb.Mode
	TriggerFromAPI  bool
	IsStreaming     bool
}

type SchedulePipelineWorkflowParam struct {
	Namespace          resource.Namespace
	PipelineID         string
	PipelineUID        uuid.UUID
	PipelineReleaseID  string
	PipelineReleaseUID uuid.UUID
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
}

type PreIteratorActivityParam struct {
	WorkflowID      string
	ID              string
	UpstreamIDs     []string
	Input           string
	SystemVariables recipe.SystemVariables
}

type PreIteratorActivityResult struct {
	ChildWorkflowIDs []string
	ElementSize      []int
}

type PostIteratorActivityParam struct {
	WorkflowID      string
	ID              string
	OutputElements  map[string]string
	SystemVariables recipe.SystemVariables
}

type PreTriggerActivityParam struct {
	WorkflowID      string
	SystemVariables recipe.SystemVariables
	IsStreaming     bool
}
type PostTriggerActivityParam struct {
	WorkflowID      string
	SystemVariables recipe.SystemVariables
	IsStreaming     bool
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
	sCtx, span := tracer.Start(context.Background(), eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(sCtx)
	logger.Info("TriggerPipelineWorkflow started")
	var err error

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	pipelineRun := &datamodel.PipelineRun{
		PipelineTriggerUID: uuid.FromStringOrNil(param.SystemVariables.PipelineTriggerID), // should not be empty
		PipelineUID:        param.SystemVariables.PipelineUID,
		PipelineVersion:    param.SystemVariables.PipelineReleaseID,
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:             param.SystemVariables.PipelineRunSource,
		TriggeredBy:        param.SystemVariables.PipelineRequesterUID.String(),
		StartedTime:        startTime,
	}
	if pipelineRun.PipelineVersion == "" {
		pipelineRun.PipelineVersion = "latest"
	}

	_ = workflow.ExecuteActivity(ctx, w.UpsertPipelineRunActivity, &UpsertPipelineRunActivityParam{
		PipelineRun: pipelineRun,
	}).Get(ctx, nil)

	// todo: mark pipeline run as failed if not succeeded
	succeeded := false
	defer func() {
		if succeeded {
			return
		}

		var errMsg string
		if err != nil {
			errMsg = err.Error()
		} else {
			errMsg = "trigger pipeline run failed due to unknown error"
		}

		_ = workflow.ExecuteActivity(ctx, w.UpdatePipelineRunActivity, &UpdatePipelineRunActivityParam{
			PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
			PipelineRun: &datamodel.PipelineRun{
				Error: null.StringFrom(errMsg),
			},
		}).Get(ctx, nil)

	}()

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	defer func() {
		if param.TriggerFromAPI {
			_ = w.memoryStore.PurgeWorkflowMemory(sCtx, workflowID)
		}

	}()
	defer func() {
		if param.IsStreaming {
			_ = w.memoryStore.SendWorkflowStatusEvent(sCtx, workflowID, memory.Event{Event: string(memory.PipelineClosed)})
		}
	}()

	var ownerType mgmtpb.OwnerType
	switch param.SystemVariables.PipelineOwnerType {
	case resource.Organization:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION
	case resource.User:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtpb.OwnerType_OWNER_TYPE_UNSPECIFIED
	}
	sessionOptions := &workflow.SessionOptions{
		CreationTimeout:  time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		ExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
	}
	ctx, _ = workflow.CreateSession(ctx, sessionOptions)
	defer workflow.CompleteSession(ctx)

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

	ctx = workflow.WithActivityOptions(ctx, ao)

	if param.TriggerFromAPI {
		if err := workflow.ExecuteActivity(ctx, w.PreTriggerActivity, &PreTriggerActivityParam{
			WorkflowID:      workflowID,
			SystemVariables: param.SystemVariables,
			IsStreaming:     param.IsStreaming,
		}).Get(ctx, nil); err != nil {
			return err
		}
	}

	wfm, err := w.memoryStore.GetWorkflowMemory(sCtx, workflowID)
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

	_ = workflow.ExecuteActivity(ctx, w.UploadRecipeToMinioActivity, &UploadRecipeToMinioActivityParam{
		PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
		UploadToMinioActivityParam: UploadToMinioActivityParam{
			ObjectName:  fmt.Sprintf("pipeline-runs/recipe/%s.json", param.SystemVariables.PipelineTriggerID),
			Data:        b,
			ContentType: constant.ContentTypeJSON,
		},
	}).Get(ctx, nil)

	_ = workflow.ExecuteActivity(ctx, w.UploadInputsToMinioActivity, &UploadInputsToMinioActivityParam{
		PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
	}).Get(ctx, nil)

	if param.IsStreaming {
		wfm.EnableStreaming()
	}

	dag, err := recipe.GenerateDAG(wfm.GetRecipe().Component)
	if err != nil {
		return err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return err
	}

	componentRunFutures := []workflow.Future{}
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

				_ = workflow.ExecuteActivity(ctx, w.UpsertComponentRunActivity, &UpsertComponentRunActivityParam{
					ComponentRun: componentRun,
				}).Get(ctx, nil)

				// adding the data row in advance in case that UploadComponentInputsActivity starts before ComponentActivity
				err = w.repository.UpsertComponentRun(sCtx, componentRun)
				if err != nil {
					logger.Error("failed to log component run start", zap.Error(err))
				}

				args := &ComponentActivityParam{
					WorkflowID:      workflowID,
					ID:              compID,
					UpstreamIDs:     upstreamIDs,
					Type:            comp.Type,
					Task:            comp.Task,
					Condition:       comp.Condition,
					SystemVariables: param.SystemVariables,
					Streaming:       param.IsStreaming,
				}

				componentRunFutures = append(componentRunFutures, workflow.ExecuteActivity(ctx, w.UploadComponentInputsActivity, args))

				futures = append(futures, workflow.ExecuteActivity(ctx, w.ComponentActivity, args))
				futureArgs = append(futureArgs, args)

			case datamodel.Iterator:
				// TODO tillknuesting: support intermediate result streaming for Iterator

				preIteratorResult := &PreIteratorActivityResult{}
				if err = workflow.ExecuteActivity(ctx, w.PreIteratorActivity, &PreIteratorActivityParam{
					WorkflowID:      workflowID,
					ID:              compID,
					UpstreamIDs:     upstreamIDs,
					Input:           comp.Input.(string),
					SystemVariables: param.SystemVariables,
				}).Get(ctx, &preIteratorResult); err != nil {
					if err != nil {
						// TODO: huitang
						// Currently, if any data in the batch has an error, we
						// treat the entire batch as errored. In the future, we
						// should allow partial errors within a batch.
						var applicationErr *temporal.ApplicationError
						if errors.As(err, &applicationErr) && applicationErr.Message() != "" {

							for batchIdx := range wfm.GetBatchSize() {
								if wfmErr := wfm.SetComponentStatus(sCtx, batchIdx, compID, memory.ComponentStatusErrored, true); wfmErr != nil {
									return wfmErr
								}
								if wfmErr := wfm.SetComponentErrorMessage(sCtx, batchIdx, compID, applicationErr.Message()); wfmErr != nil {
									return wfmErr
								}
							}
						}

						w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
						continue
					}
				}

				itFutures := []workflow.Future{}
				for iter := range wfm.GetBatchSize() {
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
							// TODO: support streaming inside iterator.
							// IsStreaming:     param.IsStreaming,
						}))
				}
				for iter := 0; iter < wfm.GetBatchSize(); iter++ {
					err = itFutures[iter].Get(ctx, nil)
					if err != nil {
						logger.Error(fmt.Sprintf("unable to execute iterator workflow: %s", err.Error()))
						return err
					}
				}

				if err = workflow.ExecuteActivity(ctx, w.PostIteratorActivity, &PostIteratorActivityParam{
					WorkflowID:      workflowID,
					ID:              compID,
					OutputElements:  comp.OutputElements,
					SystemVariables: param.SystemVariables,
				}).Get(ctx, nil); err != nil {
					if err != nil {
						// TODO: huitang
						// Currently, if any data in the batch has an error, we
						// treat the entire batch as errored. In the future, we
						// should allow partial errors within a batch.
						var applicationErr *temporal.ApplicationError
						if errors.As(err, &applicationErr) && applicationErr.Message() != "" {

							for batchIdx := range wfm.GetBatchSize() {
								if wfmErr := wfm.SetComponentStatus(sCtx, batchIdx, compID, memory.ComponentStatusErrored, true); wfmErr != nil {
									return wfmErr
								}
								if wfmErr := wfm.SetComponentErrorMessage(sCtx, batchIdx, compID, applicationErr.Message()); wfmErr != nil {
									return wfmErr
								}
							}
						}

						w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
						continue
					}
				}
			}

		}

		for idx := range futures {
			err = futures[idx].Get(ctx, nil)
			if err != nil {
				// TODO: huitang
				// Currently, if any data in the batch has an error, we
				// treat the entire batch as errored. In the future, we
				// should allow partial errors within a batch.
				var applicationErr *temporal.ApplicationError
				if errors.As(err, &applicationErr) && applicationErr.Message() != "" {

					for batchIdx := range wfm.GetBatchSize() {
						if wfmErr := wfm.SetComponentStatus(sCtx, batchIdx, futureArgs[idx].ID, memory.ComponentStatusErrored, true); wfmErr != nil {
							return wfmErr
						}
						if wfmErr := wfm.SetComponentErrorMessage(sCtx, batchIdx, futureArgs[idx].ID, applicationErr.Message()); wfmErr != nil {
							return wfmErr
						}
					}
				}

				w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
				continue
			}

			// ComponentActivity is responsible for returning a temporal
			// application error with the relevant information. Wrapping
			// the error here prevents the client from accessing the error
			// message from the activity.
			// return err
			componentRunFutures = append(componentRunFutures, workflow.ExecuteActivity(ctx, w.UploadComponentOutputsActivity, futureArgs[idx]))

		}

	}

	duration := time.Since(startTime)
	dataPoint.ComputeTimeDuration = duration.Seconds()
	dataPoint.Status = mgmtpb.Status_STATUS_COMPLETED

	if param.TriggerFromAPI {
		if err := workflow.ExecuteActivity(ctx, w.OutputActivity, &ComponentActivityParam{
			WorkflowID: workflowID,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if err := workflow.ExecuteActivity(ctx, w.UploadOutputsToMinioActivity, &UploadOutputsToMinioActivityParam{
			PipelineTriggerID: workflowID,
		}).Get(ctx, nil); err != nil {
			return err
		}

		if err := workflow.ExecuteActivity(ctx, w.PostTriggerActivity, &PostTriggerActivityParam{
			WorkflowID:      workflowID,
			SystemVariables: param.SystemVariables,
			IsStreaming:     param.IsStreaming,
		}).Get(ctx, nil); err != nil {
			return err
		}

		// TODO: we should check whether to collect failed component or not
		if err := workflow.ExecuteActivity(ctx, w.IncreasePipelineTriggerCountActivity, param.SystemVariables).Get(ctx, nil); err != nil {
			return fmt.Errorf("updating pipeline trigger count: %w", err)
		}

		if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
			logger.Warn(err.Error())
		}
	}

	for _, f := range componentRunFutures {
		_ = f.Get(ctx, nil)
	}

	_ = workflow.ExecuteActivity(ctx, w.UpdatePipelineRunActivity, &UpdatePipelineRunActivityParam{
		PipelineTriggerID: param.SystemVariables.PipelineTriggerID,
		PipelineRun: &datamodel.PipelineRun{
			CompletedTime: null.TimeFrom(time.Now()),
			Status:        datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_COMPLETED),
			TotalDuration: null.IntFrom(duration.Milliseconds()),
		},
	}).Get(ctx, nil)

	succeeded = true

	logger.Info("TriggerPipelineWorkflow completed in", zap.Duration("duration", duration))

	return nil
}

func (w *worker) UpsertPipelineRunActivity(ctx context.Context, param *UpsertPipelineRunActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("UpsertPipelineRunActivity started")

	err := w.repository.UpsertPipelineRun(ctx, param.PipelineRun)
	if err != nil {
		logger.Error("failed to log pipeline run", zap.Error(err))
	}

	logger.Info("UpsertPipelineRunActivity completed")
	return nil
}

func (w *worker) UpdatePipelineRunActivity(ctx context.Context, param *UpdatePipelineRunActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
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
	logger.Info("UpsertComponentRunActivity started")
	// adding the data row in advance in case that UploadComponentInputsActivity starts before ComponentActivity
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
		return componentActivityError(err, componentActivityErrorType, param.ID)
	}
	conditionMap, err := w.processCondition(ctx, wfm, param.ID, param.UpstreamIDs, param.Condition)
	if err != nil {
		return componentActivityError(err, componentActivityErrorType, param.ID)
	}
	setups, err := NewSetupReader(wfm, param.ID, conditionMap).Read(ctx)
	if err != nil {
		return componentActivityError(err, componentActivityErrorType, param.ID)
	}
	sysVars, err := recipe.GenerateSystemVariables(ctx, param.SystemVariables)
	if err != nil {
		return componentActivityError(err, componentActivityErrorType, param.ID)
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
		return componentActivityError(err, componentActivityErrorType, param.ID)
	}

	err = execution.Execute(
		ctx,
		NewInputReader(wfm, param.ID, conditionMap),
		NewOutputWriter(wfm, param.ID, conditionMap, param.Streaming),
	)
	if err != nil {
		return componentActivityError(err, componentActivityErrorType, param.ID)
	}

	for _, idx := range conditionMap {
		if err = wfm.SetComponentStatus(ctx, idx, param.ID, memory.ComponentStatusCompleted, true); err != nil {
			return componentActivityError(err, componentActivityErrorType, param.ID)
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
		err = wfm.SetPipelineData(ctx, idx, memory.PipelineOutput, output)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline output", outputActivityErrorType, err)
		}
	}

	logger.Info("OutputActivity completed")
	return nil
}

// TODO: complete iterator
// PreIteratorActivity generate the trigger memory for each iteration.
func (w *worker) PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreIteratorActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
	}

	result := &PreIteratorActivityResult{
		ElementSize: make([]int, wfm.GetBatchSize()),
	}

	batchSize := wfm.GetBatchSize()
	childWorkflowIDs := make([]string, batchSize)

	for iter := range wfm.GetBatchSize() {
		if err = wfm.SetComponentStatus(ctx, iter, param.ID, memory.ComponentStatusStarted, true); err != nil {
			return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
		}
		childWorkflowID := fmt.Sprintf("%s:%d:%s:%s:%s", param.WorkflowID, iter, constant.SegComponent, param.ID, constant.SegIteration)
		childWorkflowIDs[iter] = childWorkflowID

		input, err := recipe.Render(ctx, data.NewString(param.Input), iter, wfm, false)
		if err != nil {
			return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
		}

		elems := input.(*data.Array).Values
		elementSize := len(elems)
		result.ElementSize[iter] = elementSize

		iteratorRecipe := &datamodel.Recipe{
			Component: wfm.GetRecipe().Component[param.ID].Component,
		}

		childWFM, err := w.memoryStore.NewWorkflowMemory(ctx, childWorkflowIDs[iter], iteratorRecipe, elementSize)
		if err != nil {
			return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
		}

		for e := range elementSize {

			iteratorElem := data.NewMap(
				map[string]data.Value{
					"element": elems[e],
				},
			)
			err = childWFM.Set(ctx, e, param.ID, iteratorElem)
			if err != nil {
				return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
			}
			variable, err := wfm.Get(ctx, iter, constant.SegVariable)
			if err != nil {
				return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
			}
			secret, err := wfm.Get(ctx, iter, constant.SegSecret)
			if err != nil {
				return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
			}
			err = childWFM.SetPipelineData(ctx, e, memory.PipelineVariable, variable)
			if err != nil {
				return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
			}
			err = childWFM.SetPipelineData(ctx, e, memory.PipelineSecret, secret)
			if err != nil {
				return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
			}

			for _, id := range param.UpstreamIDs {
				component, err := wfm.Get(ctx, iter, id)
				if err != nil {
					return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
				}
				err = childWFM.Set(ctx, e, id, component)
				if err != nil {
					return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
				}
			}
			for compID, comp := range iteratorRecipe.Component {
				inputVal, err := data.NewValue(comp.Input)
				if err != nil {
					return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
				}
				setupVal, err := data.NewValue(comp.Setup)
				if err != nil {
					return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
				}
				childWFM.InitComponent(ctx, e, compID)
				if err := childWFM.SetComponentData(ctx, e, compID, memory.ComponentDataInput, inputVal); err != nil {
					return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
				}
				if err := childWFM.SetComponentData(ctx, e, compID, memory.ComponentDataSetup, setupVal); err != nil {
					return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
				}
			}
		}

	}
	result.ChildWorkflowIDs = childWorkflowIDs
	logger.Info("PreIteratorActivity completed")
	return result, nil
}

// PostIteratorActivity merges the trigger memory from each iteration.
func (w *worker) PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PostIteratorActivity started")

	wfm, err := w.memoryStore.GetWorkflowMemory(ctx, param.WorkflowID)
	if err != nil {
		return componentActivityError(err, postIteratorActivityErrorType, param.ID)
	}

	for iter := range wfm.GetBatchSize() {
		childWorkflowID := fmt.Sprintf("%s:%d:%s:%s:%s", param.WorkflowID, iter, constant.SegComponent, param.ID, constant.SegIteration)
		childWFM, err := w.memoryStore.GetWorkflowMemory(ctx, childWorkflowID)
		if err != nil {
			return componentActivityError(err, postIteratorActivityErrorType, param.ID)
		}
		defer func() {
			_ = w.memoryStore.PurgeWorkflowMemory(ctx, childWorkflowID)
		}()

		output := data.NewMap(nil)
		for k, v := range param.OutputElements {
			elemVals := data.NewArray(nil)

			for elemIdx := range childWFM.GetBatchSize() {
				elemVal, err := recipe.Render(ctx, data.NewString(v), elemIdx, childWFM, false)
				if err != nil {
					return componentActivityError(err, postIteratorActivityErrorType, param.ID)
				}
				elemVals.Values = append(elemVals.Values, elemVal)

			}
			output.Fields[k] = elemVals
		}
		if err = wfm.SetComponentData(ctx, iter, param.ID, memory.ComponentDataOutput, output); err != nil {
			return componentActivityError(err, postIteratorActivityErrorType, param.ID)
		}

		if err = wfm.SetComponentStatus(ctx, iter, param.ID, memory.ComponentStatusCompleted, true); err != nil {
			return componentActivityError(err, postIteratorActivityErrorType, param.ID)
		}
	}

	logger.Info("PostIteratorActivity completed")
	return nil
}

// PreTriggerActivity clone the trigger memory from Redis to MemoryStore.
func (w *worker) PreTriggerActivity(ctx context.Context, param *PreTriggerActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreTriggerActivity started")

	wfm, err := w.memoryStore.LoadWorkflowMemoryFromRedis(ctx, param.WorkflowID)
	if err != nil {
		return temporal.NewApplicationErrorWithCause("loading pipeline memory", preTriggerActivityErrorType, err)
	}
	var recipe *datamodel.Recipe
	if param.SystemVariables.PipelineReleaseUID.IsNil() {
		pipeline, err := w.repository.GetPipelineByUIDAdmin(ctx, param.SystemVariables.PipelineUID, false, false)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline recipe", preTriggerActivityErrorType, err)
		}
		recipe = pipeline.Recipe
	} else {
		release, err := w.repository.GetPipelineReleaseByUIDAdmin(ctx, param.SystemVariables.PipelineReleaseUID, false)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline recipe", preTriggerActivityErrorType, err)
		}
		recipe = release.Recipe
	}

	wfm.SetRecipe(recipe)

	pt := ""
	var nsSecrets []*datamodel.Secret
	ownerPermalink := fmt.Sprintf("%s/%s", param.SystemVariables.PipelineOwnerType, param.SystemVariables.PipelineOwnerUID)
	for {
		var secrets []*datamodel.Secret
		secrets, _, pt, err = w.repository.ListNamespaceSecrets(ctx, ownerPermalink, 100, pt, filtering.Filter{})
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline secret memory", preTriggerActivityErrorType, err)
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

	for idx := range wfm.GetBatchSize() {
		pipelineSecrets, err := wfm.Get(ctx, idx, constant.SegSecret)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("loading pipeline secret memory", preTriggerActivityErrorType, err)
		}

		for _, secret := range nsSecrets {
			if _, ok := pipelineSecrets.(*data.Map).Fields[secret.ID]; !ok {
				pipelineSecrets.(*data.Map).Fields[secret.ID] = data.NewString(*secret.Value)
			}
		}

		// Init component template
		for compID, comp := range recipe.Component {

			wfm.InitComponent(ctx, idx, compID)

			inputVal, err := data.NewValue(comp.Input)
			if err != nil {
				return temporal.NewApplicationErrorWithCause("initializing pipeline input memory", preTriggerActivityErrorType, err)
			}
			if err := wfm.SetComponentData(ctx, idx, compID, memory.ComponentDataInput, inputVal); err != nil {
				return temporal.NewApplicationErrorWithCause("initializing pipeline input memory", preTriggerActivityErrorType, err)
			}

			setupVal, err := data.NewValue(comp.Setup)
			if err != nil {
				return temporal.NewApplicationErrorWithCause("initializing pipeline setup memory", preTriggerActivityErrorType, err)
			}
			if err := wfm.SetComponentData(ctx, idx, compID, memory.ComponentDataSetup, setupVal); err != nil {
				return temporal.NewApplicationErrorWithCause("initializing pipeline setup memory", preTriggerActivityErrorType, err)
			}
		}
		output := data.NewMap(nil)

		for key, o := range recipe.Output {
			output.Fields[key] = data.NewString(o.Value)
		}
		err = wfm.SetPipelineData(ctx, idx, memory.PipelineOutputTemplate, output)
		if err != nil {
			return temporal.NewApplicationErrorWithCause("initializing pipeline memory", preTriggerActivityErrorType, err)
		}
	}

	if param.IsStreaming {
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
				return temporal.NewApplicationErrorWithCause("sending event", preTriggerActivityErrorType, err)
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

	err := w.memoryStore.WriteWorkflowMemoryToRedis(ctx, param.WorkflowID)
	if err != nil {
		return temporal.NewApplicationErrorWithCause("loading pipeline memory", postTriggerActivityErrorType, err)
	}
	wfm, err := w.memoryStore.LoadWorkflowMemoryFromRedis(ctx, param.WorkflowID)
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

		if param.IsStreaming {
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
				if err = wfm.SetComponentStatus(ctx, idx, upstreamID, memory.ComponentStatusSkipped, true); err != nil {
					return nil, err
				}
				break
			}
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
func (w *worker) writeErrorDataPoint(ctx context.Context, err error, span trace.Span, startTime time.Time, dataPoint *utils.PipelineUsageMetricData) {
	span.SetStatus(1, err.Error())
	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtpb.Status_STATUS_ERRORED
	_ = w.writeNewDataPoint(ctx, *dataPoint)
}

// componentActivityError transforms an error with (potentially) an end-user
// message into a Temporal application error. Temporal clients can extract the
// message and propagate it to the end user.
func componentActivityError(err error, errType, componentID string) error {
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
		},
		Mode: mgmtpb.Mode_MODE_ASYNC,
	}

	wfUID, _ := uuid.NewV4()
	childWorkflowOptions := workflow.ChildWorkflowOptions{
		TaskQueue:                TaskQueue,
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
