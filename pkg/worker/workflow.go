// This file will be refactored soon
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	BatchSize        int
	MemoryStorageKey *recipe.BatchMemoryKey
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	Mode             mgmtpb.Mode
	IsIterator       bool
	IsStreaming      bool
}

// ComponentActivityParam represents the parameters for TriggerActivity
type ComponentActivityParam struct {
	WorkflowID       string
	MemoryStorageKey *recipe.BatchMemoryKey
	ID               string
	UpstreamIDs      []string
	Condition        string
	Input            map[string]any
	Setup            map[string]any
	Type             string
	Task             string
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
}

type PreIteratorActivityParam struct {
	WorkflowID       string
	MemoryStorageKey *recipe.BatchMemoryKey
	ID               string
	UpstreamIDs      []string
	Input            string
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
}

type PreIteratorActivityResult struct {
	ChildWorkflowIDs  []string
	MemoryStorageKeys []*recipe.BatchMemoryKey
	ElementSize       []int
}

type PostIteratorActivityParam struct {
	WorkflowID        string
	MemoryStorageKeys []*recipe.BatchMemoryKey
	ID                string
	OutputElements    map[string]string
	SystemVariables   recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
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

	// Inline function to initialize the channel only if streaming is active
	initChan := func() chan WorkFlowSignal {
		if param.IsStreaming {
			return make(chan WorkFlowSignal, 100) // 100 is probably more than enough
		}
		return nil
	}

	// sChan is used to signal from the Workflow to the QueryHandler if an Activity is completed.
	// The QueryHandler will be called by the client to get the status of the Workflow in order
	// to act accordingly e.g. signal partial completion of the Workflow. The buffer size is set to 100 but
	// can be adjusted based on the expected number of components in the Workflow.
	var sChan = initChan()
	const statusStep = "step"
	const statusCompleted = "completed"

	if param.IsStreaming {
		logger.Debug("streaming is active")
		// sChan is used to signal from the Workflow to the QueryHandler if an Activity is completed.
		// The QueryHandler will be called by the client to get the status of the Workflow in order
		// act accordingly e.g. signal partial completion of the Workflow. The buffer size is set to 100 but
		// can be adjusted based on the expected number of components in the Workflow.
		var sChan2 = make(chan WorkFlowSignal, 100)
		sChan = sChan2

		// Register query handler for workflow status
		err := workflow.SetQueryHandler(ctx, "workflowStatusQuery", func() (WorkFlowSignal, error) {
			select {
			case msg := <-sChan:
				if len(msg.Status) == 0 {
					return WorkFlowSignal{}, nil
				}
				return msg, nil
			case <-time.After(time.Second * 3):
				return WorkFlowSignal{Status: "timeout"}, nil
			}
		})
		if err != nil {
			return err
		}

		sChan <- WorkFlowSignal{Status: "started"}
		defer close(sChan)
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

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxActivityRetry,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	r, err := recipe.LoadRecipe(sCtx, w.redisClient, param.MemoryStorageKey.Recipe)
	if err != nil {
		return err
	}

	dag, err := recipe.GenerateDAG(r.Component)
	if err != nil {
		return err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return err
	}

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	for i := range param.BatchSize {
		if param.MemoryStorageKey.Components[i] == nil {
			param.MemoryStorageKey.Components[i] = map[string]string{}
		}
	}

	// The components in the same group can be executed in parallel
	for group := range orderedComp {
		futures := []workflow.Future{}
		for compID, comp := range orderedComp[group] {
			upstreamIDs := dag.GetUpstreamCompIDs(compID)

			switch comp.Type {
			default:
				futures = append(futures, workflow.ExecuteActivity(ctx, w.ComponentActivity, &ComponentActivityParam{
					WorkflowID:       workflowID,
					ID:               compID,
					UpstreamIDs:      upstreamIDs,
					Type:             comp.Type,
					Task:             comp.Task,
					Input:            comp.Input.(map[string]any),
					Setup:            comp.Setup,
					Condition:        comp.Condition,
					MemoryStorageKey: param.MemoryStorageKey,
					SystemVariables:  param.SystemVariables,
				}))

			case datamodel.Iterator:
				//TODO tillknuesting: support intermediate result streaming for Iterator

				preIteratorResult := &PreIteratorActivityResult{}
				if err = workflow.ExecuteActivity(ctx, w.PreIteratorActivity, &PreIteratorActivityParam{
					WorkflowID:       workflowID,
					ID:               compID,
					UpstreamIDs:      upstreamIDs,
					Input:            comp.Input.(string),
					SystemVariables:  param.SystemVariables,
					MemoryStorageKey: param.MemoryStorageKey,
				}).Get(ctx, &preIteratorResult); err != nil {
					return err
				}

				itFutures := []workflow.Future{}
				for iter := range param.BatchSize {
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
							IsIterator:       true,
							BatchSize:        preIteratorResult.ElementSize[iter],
							MemoryStorageKey: preIteratorResult.MemoryStorageKeys[iter],
							SystemVariables:  param.SystemVariables,
							Mode:             mgmtpb.Mode_MODE_SYNC,
						}))

				}
				for iter := 0; iter < param.BatchSize; iter++ {
					err = itFutures[iter].Get(ctx, nil)
					if err != nil {
						logger.Error(fmt.Sprintf("unable to execute iterator workflow: %s", err.Error()))
						return err
					}
				}

				if err = workflow.ExecuteActivity(ctx, w.PostIteratorActivity, &PostIteratorActivityParam{
					WorkflowID:        workflowID,
					ID:                compID,
					MemoryStorageKeys: preIteratorResult.MemoryStorageKeys,
					OutputElements:    comp.OutputElements,
					SystemVariables:   param.SystemVariables,
				}).Get(ctx, nil); err != nil {
					return err
				}
			}

		}

		for idx := range futures {
			var result ComponentActivityParam
			err = futures[idx].Get(ctx, &result)
			if err != nil {
				w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)

				// ComponentActivity is responsible of returning a temporal
				// application error with the relevant information. Wrapping
				// the error here prevents the client from accessing the error
				// message from the activity.
				return err
			}
			if param.IsStreaming {
				sChan <- WorkFlowSignal{Status: statusStep, ID: result.ID}
			}
		}

		for batchIdx := range param.BatchSize {
			for compID := range orderedComp[group] {
				param.MemoryStorageKey.Components[batchIdx][compID] = fmt.Sprintf("%s:%d:%s:%s", workflowID, batchIdx, recipe.SegComponent, compID)
			}
		}
		if param.IsStreaming {
			// if we don't sleep, there will be race condition between Redis write and read
			if err := workflow.Sleep(ctx, time.Millisecond*10); err != nil {
				logger.Error(fmt.Sprintf(" workflow unable to sleep: %s", err.Error()))
			}
		}
	}

	if param.IsStreaming {
		sChan <- WorkFlowSignal{Status: statusCompleted}
	}

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtpb.Status_STATUS_COMPLETED

	if !param.IsIterator {
		// TODO: we should check whether to collect failed component or not
		if err := workflow.ExecuteActivity(ctx, w.IncreasePipelineTriggerCountActivity, param.SystemVariables).Get(ctx, nil); err != nil {
			return fmt.Errorf("updating pipeline trigger count: %w", err)
		}

		if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
			logger.Warn(err.Error())
		}
	}

	logger.Info("TriggerPipelineWorkflow completed in", zap.Duration("duration", time.Since(startTime)))

	return nil
}

func (w *worker) ComponentActivity(ctx context.Context, param *ComponentActivityParam) (*ComponentActivityParam, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ComponentActivity started")

	batchMemory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	compInputs, idxMap, err := w.processInput(batchMemory, param.ID, param.UpstreamIDs, param.Condition, param.Input)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	cons, err := w.processSetup(batchMemory, param.Setup)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}
	sysVars, err := recipe.GenerateSystemVariables(ctx, param.SystemVariables)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	comp, err := w.component.GetDefinitionByID(param.Type, nil, nil)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	// Note: we assume that setup in the batch are all the same
	execution, err := w.component.CreateExecution(uuid.FromStringOrNil(comp.Uid), sysVars, cons[0], param.Task)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	compOutputs, err := execution.Execute(ctx, compInputs)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	compMem, err := w.processOutput(batchMemory, param.ID, compOutputs, idxMap)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.WorkflowID, param.ID, compMem)
	if err != nil {
		return nil, componentActivityError(err, componentActivityErrorType, param.ID)
	}

	logger.Info("ComponentActivity completed")

	// the data is logged in temporal hence we should only return data that is needed
	p := &ComponentActivityParam{
		WorkflowID: param.WorkflowID,
		ID:         param.ID, // is used by the caller to identify the component
	}
	return p, nil
}

// TODO: complete iterator
// PreIteratorActivity generate the trigger memory for each iteration.
func (w *worker) PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreIteratorActivity started")

	m, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
	}
	r, err := recipe.LoadRecipe(ctx, w.redisClient, param.MemoryStorageKey.Recipe)
	if err != nil {
		return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
	}

	iteratorRecipe := &datamodel.Recipe{
		Component: r.Component[param.ID].Component,
	}

	result := &PreIteratorActivityResult{
		MemoryStorageKeys: make([]*recipe.BatchMemoryKey, len(m)),
		ElementSize:       make([]int, len(m)),
	}
	recipeKey := fmt.Sprintf("%s:%s", param.WorkflowID, recipe.SegRecipe)
	err = recipe.WriteRecipe(ctx, w.redisClient, recipeKey, iteratorRecipe)
	if err != nil {
		return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
	}

	batchSize := len(m)
	childWorkflowIDs := make([]string, batchSize)
	for iter := range batchSize {
		childWorkflowIDs[iter] = fmt.Sprintf("%s:%d:%s:%s:%s", param.WorkflowID, iter, recipe.SegComponent, param.ID, recipe.SegIteration)
	}

	for iter := range m {

		input, err := recipe.RenderInput(param.Input, iter, m[iter])
		if err != nil {
			return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
		}

		elems := make([]*recipe.ComponentMemory, len(input.([]any)))
		for elemIdx := range input.([]any) {
			elems[elemIdx] = &recipe.ComponentMemory{
				Element: input.([]any)[elemIdx],
			}

		}
		elementSize := len(elems)
		result.ElementSize[iter] = elementSize

		err = recipe.WriteComponentMemory(ctx, w.redisClient, childWorkflowIDs[iter], param.ID, elems)
		if err != nil {
			return nil, componentActivityError(err, preIteratorActivityErrorType, param.ID)
		}

		varKeys := make([]string, elementSize)
		for e := range elementSize {
			varKeys[e] = param.MemoryStorageKey.Variables[iter]
		}
		secretKeys := make([]string, elementSize)
		for e := range elementSize {
			secretKeys[e] = param.MemoryStorageKey.Secrets[iter]
		}
		compKeys := make([]map[string]string, elementSize)
		for e := range elementSize {
			compKeys[e] = make(map[string]string)
			for _, id := range param.UpstreamIDs {
				compKeys[e][id] = param.MemoryStorageKey.Components[iter][id]
			}
		}

		for e := range elementSize {

			compKeys[e][param.ID] = fmt.Sprintf("%s:%d:%s:%s", childWorkflowIDs[iter], e, recipe.SegComponent, param.ID)
		}

		k := &recipe.BatchMemoryKey{
			Components:     compKeys,
			Variables:      varKeys,
			Secrets:        secretKeys,
			Recipe:         recipeKey,
			OwnerPermalink: param.MemoryStorageKey.OwnerPermalink,
		}
		result.MemoryStorageKeys[iter] = k

	}
	result.ChildWorkflowIDs = childWorkflowIDs
	logger.Info("PreIteratorActivity completed")
	return result, nil
}

// PostIteratorActivity merges the trigger memory from each iteration.
func (w *worker) PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PostIteratorActivity started")

	// recipes for all iteration are the same
	r, err := recipe.LoadRecipe(ctx, w.redisClient, param.MemoryStorageKeys[0].Recipe)
	if err != nil {
		return componentActivityError(err, postIteratorActivityErrorType, param.ID)
	}

	iterComp := []*recipe.ComponentMemory{}
	for iter := range param.MemoryStorageKeys {

		k := param.MemoryStorageKeys[iter]
		for e := range len(k.Variables) {
			for compID := range r.Component {
				k.Components[e][compID] = fmt.Sprintf("%s:%d:%s:%s:%s:%d:%s:%s", param.WorkflowID, iter, recipe.SegComponent, param.ID, recipe.SegIteration, e, recipe.SegComponent, compID)
			}
		}

		m, err := recipe.LoadMemory(ctx, w.redisClient, k)
		if err != nil {
			return componentActivityError(err, postIteratorActivityErrorType, param.ID)
		}

		output := recipe.ComponentIO{}
		for k, v := range param.OutputElements {
			elemVals := []any{}

			for elemIdx := range len(m) {
				elemVal, err := recipe.RenderInput(v, elemIdx, m[elemIdx])
				if err != nil {
					return componentActivityError(err, postIteratorActivityErrorType, param.ID)
				}
				elemVals = append(elemVals, elemVal)

			}
			output[k] = elemVals
		}

		iterComp = append(iterComp, &recipe.ComponentMemory{
			Output: &output,
			Status: &recipe.ComponentStatus{ // TODO: use real status
				Started:   true,
				Completed: true,
			},
		})
	}
	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.WorkflowID, param.ID, iterComp)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return componentActivityError(err, postIteratorActivityErrorType, param.ID)
	}

	logger.Info("PostIteratorActivity completed")
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

func (w *worker) processInput(batchMemory []*recipe.Memory, id string, UpstreamIDs []string, condition string, input any) ([]*structpb.Struct, map[int]int, error) {
	var compInputs []*structpb.Struct
	idxMap := map[int]int{}

	for idx := range batchMemory {

		batchMemory[idx].Component[id] = &recipe.ComponentMemory{
			Input:  &recipe.ComponentIO{},
			Output: &recipe.ComponentIO{},
			Status: &recipe.ComponentStatus{},
		}

		for _, upstreamID := range UpstreamIDs {
			if batchMemory[idx].Component[upstreamID].Status.Skipped {
				batchMemory[idx].Component[id].Status.Skipped = true
				break
			}
		}

		if !batchMemory[idx].Component[id].Status.Skipped {
			if condition != "" {

				// TODO: these code should be refactored and shared some common functions with RenderInput
				condStr := condition
				var varMapping map[string]string
				condStr, _, varMapping = recipe.SanitizeCondition(condStr)

				expr, err := parser.ParseExpr(condStr)
				if err != nil {
					return nil, nil, err
				}

				condMemory := map[string]any{}

				for k, v := range batchMemory[idx].Component {
					condMemory[varMapping[k]] = v
				}
				condMemory[varMapping["variable"]] = batchMemory[idx].Variable

				cond, err := recipe.EvalCondition(expr, condMemory)
				if err != nil {
					return nil, nil, err
				}
				if cond == false {
					batchMemory[idx].Component[id].Status.Skipped = true
				} else {
					batchMemory[idx].Component[id].Status.Started = true
				}
			} else {
				batchMemory[idx].Component[id].Status.Started = true
			}
		}

		if batchMemory[idx].Component[id].Status.Started {

			var compInputTemplateJSON []byte
			compInputTemplate := input

			compInputTemplateJSON, err := json.Marshal(compInputTemplate)
			if err != nil {
				return nil, nil, err
			}
			var compInputTemplateStruct any
			err = json.Unmarshal(compInputTemplateJSON, &compInputTemplateStruct)
			if err != nil {
				return nil, nil, err
			}

			compInputStruct, err := recipe.RenderInput(compInputTemplateStruct, idx, batchMemory[idx])
			if err != nil {
				return nil, nil, err
			}
			compInputJSON, err := json.Marshal(compInputStruct)
			if err != nil {
				return nil, nil, err
			}

			compInput := &structpb.Struct{}
			err = protojson.Unmarshal([]byte(compInputJSON), compInput)
			if err != nil {
				return nil, nil, err
			}

			*batchMemory[idx].Component[id].Input = compInputStruct.(map[string]any)

			idxMap[len(compInputs)] = idx
			compInputs = append(compInputs, compInput)

		}
	}
	return compInputs, idxMap, nil
}

func (w *worker) processOutput(batchMemory []*recipe.Memory, id string, compOutputs []*structpb.Struct, idxMap map[int]int) ([]*recipe.ComponentMemory, error) {

	for idx := range compOutputs {

		outputJSON, err := protojson.Marshal(compOutputs[idx])
		if err != nil {
			return nil, err
		}
		var outputStruct map[string]any
		err = json.Unmarshal(outputJSON, &outputStruct)
		if err != nil {
			return nil, err
		}
		*batchMemory[idxMap[idx]].Component[id].Output = outputStruct
		batchMemory[idxMap[idx]].Component[id].Status.Completed = true
	}

	compMem := make([]*recipe.ComponentMemory, len(batchMemory))
	for idx, m := range batchMemory {
		compMem[idx] = m.Component[id]
	}

	return compMem, nil
}

func (w *worker) processSetup(batchMemory []*recipe.Memory, setup map[string]any) ([]*structpb.Struct, error) {

	if setup == nil {
		setup = map[string]any{}
	}

	conTemplateJSON, err := json.Marshal(setup)
	if err != nil {
		return nil, err
	}
	var conTemplateStruct any
	err = json.Unmarshal(conTemplateJSON, &conTemplateStruct)
	if err != nil {
		return nil, err
	}

	cons := []*structpb.Struct{}
	for idx := range batchMemory {
		conStruct, err := recipe.RenderInput(conTemplateStruct, 0, batchMemory[idx])
		if err != nil {
			return nil, err
		}
		conJSON, err := json.Marshal(conStruct)
		if err != nil {
			return nil, err
		}
		con := &structpb.Struct{}
		err = protojson.Unmarshal([]byte(conJSON), con)
		if err != nil {
			return nil, err
		}
		cons = append(cons, con)
	}

	return cons, nil
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
	preIteratorActivityErrorType  = "PreIteratorActivityError"
	postIteratorActivityErrorType = "PostIteratorActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}
