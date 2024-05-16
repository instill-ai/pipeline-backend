// This file will be refactored soon
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	BatchSize        int
	MemoryStorageKey *recipe.TriggerMemoryKey
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	Mode             mgmtPB.Mode
}

// ConnectorActivityParam represents the parameters for TriggerActivity
type ConnectorActivityParam struct {
	MemoryStorageKey *recipe.TriggerMemoryKey
	TargetStorageKey string
	ID               string
	UpstreamIDs      []string
	Condition        *string
	Input            *structpb.Struct
	Connection       *structpb.Struct
	DefinitionUID    uuid.UUID
	Task             string
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
}

// OperatorActivityParam represents the parameters for TriggerActivity
type OperatorActivityParam struct {
	MemoryStorageKey *recipe.TriggerMemoryKey
	TargetStorageKey string
	ID               string
	UpstreamIDs      []string
	Condition        *string
	Input            *structpb.Struct
	DefinitionUID    uuid.UUID
	Task             string
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
}

type PreIteratorActivityParam struct {
	MemoryStorageKey *recipe.TriggerMemoryKey
	TargetStorageKey string
	ID               string
	UpstreamIDs      []string
	Input            string
	SystemVariables  recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
}

type PreIteratorActivityResult struct {
	MemoryStorageKeys []*recipe.TriggerMemoryKey
	ElementSize       []int
}

type PostIteratorActivityParam struct {
	MemoryStorageKeys []*recipe.TriggerMemoryKey
	TargetStorageKey  string
	ID                string
	OutputElements    map[string]string
	SystemVariables   recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
}

type UsageCollectActivityParam struct {
	SystemVariables recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	NumComponents   int
}
type UsageCheckActivityParam struct {
	SystemVariables recipe.SystemVariables // TODO: we should store vars directly in trigger memory.
	NumComponents   int
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

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

	var ownerType mgmtPB.OwnerType
	switch param.SystemVariables.PipelineOwnerType {
	case resource.Organization:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION
	case resource.User:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           param.SystemVariables.PipelineOwnerUID.String(),
		OwnerType:          ownerType,
		UserUID:            param.SystemVariables.PipelineUserUID.String(),
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER, // TODO: currently only support /users type, will change after beta
		TriggerMode:        param.Mode,
		PipelineID:         param.SystemVariables.PipelineID,
		PipelineUID:        param.SystemVariables.PipelineUID.String(),
		PipelineReleaseID:  param.SystemVariables.PipelineReleaseID,
		PipelineReleaseUID: param.SystemVariables.PipelineReleaseUID.String(),
		PipelineTriggerUID: workflow.GetInfo(ctx).WorkflowExecution.ID,
		TriggerTime:        startTime.Format(time.RFC3339Nano),
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

	dag, err := recipe.GenerateDAG(r.Components)
	if err != nil {
		return err
	}
	numComponents := len(r.Components)
	for _, comp := range r.Components {
		if comp.IsIteratorComponent() {
			numComponents += len(comp.IteratorComponent.Components)
		}
	}

	if err := workflow.ExecuteActivity(ctx, w.UsageCheckActivity, &UsageCheckActivityParam{
		SystemVariables: param.SystemVariables,
		NumComponents:   numComponents,
	}).Get(ctx, nil); err != nil {
		details := EndUserErrorDetails{
			Message: fmt.Sprintf("Pipeline %s failed to execute. %s", param.SystemVariables.PipelineUID, errmsg.MessageOrErr(err)),
		}
		return temporal.NewApplicationErrorWithCause("pipeline failed to trigger", PipelineWorkflowError, err, details)
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return err
	}

	workflowID := workflow.GetInfo(ctx).WorkflowExecution.ID

	if param.MemoryStorageKey.Components == nil {
		param.MemoryStorageKey.Components = map[string][]string{}
	}

	// The components in the same group can be executed in parallel
	for group := range orderedComp {

		futures := []workflow.Future{}
		for _, comp := range orderedComp[group] {

			upstreamIDs := dag.GetUpstreamCompIDs(comp.ID)
			targetStorageKey := fmt.Sprintf("%s:%s:%s", workflowID, recipe.SegComponent, comp.ID)

			switch {
			case comp.IsConnectorComponent():
				futures = append(futures, workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ConnectorActivityParam{
					ID:               comp.ID,
					UpstreamIDs:      upstreamIDs,
					DefinitionUID:    uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.DefinitionName, "/")[1]),
					Task:             comp.ConnectorComponent.Task,
					Input:            comp.ConnectorComponent.Input,
					Connection:       comp.ConnectorComponent.Connection,
					Condition:        comp.ConnectorComponent.Condition,
					MemoryStorageKey: param.MemoryStorageKey,
					TargetStorageKey: targetStorageKey,
					SystemVariables:  param.SystemVariables,
				}))

			case comp.IsOperatorComponent():
				futures = append(futures, workflow.ExecuteActivity(ctx, w.OperatorActivity, &OperatorActivityParam{
					ID:               comp.ID,
					UpstreamIDs:      upstreamIDs,
					DefinitionUID:    uuid.FromStringOrNil(strings.Split(comp.OperatorComponent.DefinitionName, "/")[1]),
					Task:             comp.OperatorComponent.Task,
					Input:            comp.OperatorComponent.Input,
					Condition:        comp.OperatorComponent.Condition,
					MemoryStorageKey: param.MemoryStorageKey,
					TargetStorageKey: targetStorageKey,
					SystemVariables:  param.SystemVariables,
				}))

			case comp.IsIteratorComponent():

				preIteratorResult := &PreIteratorActivityResult{}
				if err = workflow.ExecuteActivity(ctx, w.PreIteratorActivity, &PreIteratorActivityParam{
					ID:               comp.ID,
					UpstreamIDs:      upstreamIDs,
					Input:            comp.IteratorComponent.Input,
					SystemVariables:  param.SystemVariables,
					MemoryStorageKey: param.MemoryStorageKey,
					TargetStorageKey: targetStorageKey,
				}).Get(ctx, &preIteratorResult); err != nil {
					return err
				}

				itFutures := []workflow.Future{}
				for iter := 0; iter < param.BatchSize; iter++ {
					childWorkflowID := fmt.Sprintf("%s:%s:%s:%s:%d", workflowID, recipe.SegComponent, comp.ID, recipe.SegIteration, iter)
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
						"TriggerPipelineWorkflow",
						&TriggerPipelineWorkflowParam{
							BatchSize:        preIteratorResult.ElementSize[iter],
							MemoryStorageKey: preIteratorResult.MemoryStorageKeys[iter],
							SystemVariables:  param.SystemVariables,
							Mode:             mgmtPB.Mode_MODE_SYNC,
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
					ID:                comp.ID,
					MemoryStorageKeys: preIteratorResult.MemoryStorageKeys,
					TargetStorageKey:  targetStorageKey,
					OutputElements:    comp.IteratorComponent.OutputElements,
					SystemVariables:   param.SystemVariables,
				}).Get(ctx, nil); err != nil {
					return err
				}

			}

		}

		for idx := range futures {
			err = futures[idx].Get(ctx, nil)
			if err != nil {
				w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
				return err
			}
		}
		for _, comp := range orderedComp[group] {
			param.MemoryStorageKey.Components[comp.ID] = make([]string, param.BatchSize)
			for batchIdx := 0; batchIdx < param.BatchSize; batchIdx++ {
				param.MemoryStorageKey.Components[comp.ID][batchIdx] = fmt.Sprintf("%s:%s:%s:%d", workflowID, recipe.SegComponent, comp.ID, batchIdx)
			}
		}

	}

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED

	// TODO: we should check whether to collect failed component or not
	if err := workflow.ExecuteActivity(ctx, w.UsageCollectActivity, &UsageCollectActivityParam{
		SystemVariables: param.SystemVariables,
		NumComponents:   numComponents,
	}).Get(ctx, nil); err != nil {
		return err
	}
	if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}
	logger.Info("TriggerPipelineWorkflow completed")
	return nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ConnectorActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ConnectorActivity started")

	memory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	compInputs, idxMap, err := w.processInput(memory, param.ID, param.UpstreamIDs, param.Condition, param.Input)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	con, err := w.processConnection(memory, param.Connection)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	vars, err := recipe.GenerateSystemVariables(ctx, param.SystemVariables)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	execution, err := w.connector.CreateExecution(param.DefinitionUID, vars, con, param.Task)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	compOutputs, err := execution.Execute(ctx, compInputs)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	compMem, err := w.processOutput(memory, param.ID, compOutputs, idxMap)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.TargetStorageKey, compMem)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	logger.Info("ConnectorActivity completed")
	return nil
}

func (w *worker) OperatorActivity(ctx context.Context, param *OperatorActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("OperatorActivity started")

	memory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	compInputs, idxMap, err := w.processInput(memory, param.ID, param.UpstreamIDs, param.Condition, param.Input)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	vars, err := recipe.GenerateSystemVariables(ctx, param.SystemVariables)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	execution, err := w.operator.CreateExecution(param.DefinitionUID, vars, param.Task)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	compOutputs, err := execution.Execute(ctx, compInputs)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	compMem, err := w.processOutput(memory, param.ID, compOutputs, idxMap)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.TargetStorageKey, compMem)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	logger.Info("OperatorActivity completed")
	return nil
}

// PreIteratorActivity generate the trigger memory for each iteration.
func (w *worker) PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreIteratorActivity started")

	m, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryStorageKey)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
	}
	r, err := recipe.LoadRecipe(ctx, w.redisClient, param.MemoryStorageKey.Recipe)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
	}

	var iteratorRecipe *datamodel.Recipe
	for _, comp := range r.Components {
		if comp.ID == param.ID {
			iteratorRecipe = &datamodel.Recipe{
				Components: comp.IteratorComponent.Components,
			}
			break
		}
	}

	result := &PreIteratorActivityResult{
		MemoryStorageKeys: make([]*recipe.TriggerMemoryKey, len(m.Inputs)),
		ElementSize:       make([]int, len(m.Inputs)),
	}
	recipeKey := fmt.Sprintf("%s:%s", param.TargetStorageKey, recipe.SegRecipe)
	err = recipe.WriteRecipe(ctx, w.redisClient, recipeKey, iteratorRecipe)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
	}
	for iter := range m.Inputs {

		input, err := recipe.RenderInput(param.Input, iter, m.Components, m.Inputs, m.Secrets)
		if err != nil {
			return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
		}

		elems := make([]*recipe.ComponentItemMemory, len(input.([]any)))
		for elemIdx := range input.([]any) {
			elems[elemIdx] = &recipe.ComponentItemMemory{
				Element: input.([]any)[elemIdx],
			}

		}
		elementSize := len(elems)
		result.ElementSize[iter] = elementSize

		err = recipe.WriteComponentMemory(ctx, w.redisClient, fmt.Sprintf("%s:%s:%d:%s:%s", param.TargetStorageKey, recipe.SegIteration, iter, recipe.SegComponent, param.ID), elems)
		if err != nil {
			return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
		}

		inputStorageKeys := make([]string, elementSize)
		for e := 0; e < elementSize; e++ {
			inputStorageKeys[e] = param.MemoryStorageKey.Inputs[iter]
		}
		compStorageKeys := map[string][]string{}
		for _, ID := range param.UpstreamIDs {
			compStorageKeys[ID] = make([]string, elementSize)
			for e := 0; e < elementSize; e++ {
				compStorageKeys[ID][e] = param.MemoryStorageKey.Components[ID][iter]
			}
		}
		compStorageKeys[param.ID] = make([]string, elementSize)
		for e := 0; e < elementSize; e++ {
			compStorageKeys[param.ID][e] = fmt.Sprintf("%s:%s:%d:%s:%s:%d",
				param.TargetStorageKey, recipe.SegIteration, iter, recipe.SegComponent, param.ID, e)
		}

		k := &recipe.TriggerMemoryKey{
			Components: compStorageKeys,
			Inputs:     inputStorageKeys,
			Secrets:    param.MemoryStorageKey.Secrets,
			Vars:       param.MemoryStorageKey.Vars,
			Recipe:     recipeKey,
		}
		result.MemoryStorageKeys[iter] = k

	}

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
		return w.toApplicationError(err, param.ID, PreIteratorActivityError)
	}

	iterComp := recipe.ComponentsMemory{}
	for iter := range param.MemoryStorageKeys {

		k := param.MemoryStorageKeys[iter]
		for _, comp := range r.Components {
			k.Components[comp.ID] = make([]string, len(k.Inputs))
			for e := 0; e < len(k.Inputs); e++ {
				k.Components[comp.ID][e] = fmt.Sprintf("%s:%s:%d:%s:%s:%d", param.TargetStorageKey, recipe.SegIteration, iter, recipe.SegComponent, comp.ID, e)
			}
		}

		m, err := recipe.LoadMemory(ctx, w.redisClient, k)
		if err != nil {
			return w.toApplicationError(err, param.ID, PostIteratorActivityError)
		}

		output := recipe.ComponentIO{}
		for k, v := range param.OutputElements {
			elemVals := []any{}

			for elemIdx := range m.Inputs {
				elemVal, err := recipe.RenderInput(v, elemIdx, m.Components, m.Inputs, m.Secrets)
				if err != nil {
					return w.toApplicationError(err, param.ID, PostIteratorActivityError)
				}
				elemVals = append(elemVals, elemVal)

			}
			output[k] = elemVals
		}

		iterComp = append(iterComp, &recipe.ComponentItemMemory{
			Output: &output,
			Status: &recipe.ComponentStatus{ // TODO: use real status
				Started:   true,
				Completed: true,
			},
		})
	}
	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.TargetStorageKey, iterComp)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return w.toApplicationError(err, param.ID, PostIteratorActivityError)
	}

	logger.Info("PostIteratorActivity completed")
	return nil
}

func (w *worker) UsageCheckActivity(ctx context.Context, param *UsageCheckActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("UsageCheckActivity started")

	err := w.pipelineUsageHandler.Check(ctx, param.SystemVariables, param.NumComponents)
	if err != nil {
		return w.toApplicationError(err, "usage_check", UsageCollectActivityError)
	}
	logger.Info("UsageCheckActivity completed")
	return nil
}

func (w *worker) UsageCollectActivity(ctx context.Context, param *UsageCollectActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("UsageCollectActivity started")

	err := w.pipelineUsageHandler.Collect(ctx, param.SystemVariables, param.NumComponents)
	if err != nil {
		return w.toApplicationError(err, "usage_collect", UsageCollectActivityError)
	}
	logger.Info("UsageCollectActivity completed")
	return nil
}

func (w *worker) processInput(memory *recipe.TriggerMemory, id string, UpstreamIDs []string, condition *string, input *structpb.Struct) ([]*structpb.Struct, map[int]int, error) {
	batchSize := len(memory.Inputs)
	var compInputs []*structpb.Struct
	idxMap := map[int]int{}

	memory.Components[id] = make([]*recipe.ComponentItemMemory, batchSize)

	for idx := 0; idx < batchSize; idx++ {

		memory.Components[id][idx] = &recipe.ComponentItemMemory{
			Input:  &recipe.ComponentIO{},
			Output: &recipe.ComponentIO{},
			Status: &recipe.ComponentStatus{},
		}

		for _, upstreamID := range UpstreamIDs {
			if memory.Components[upstreamID][idx].Status.Skipped {
				memory.Components[id][idx].Status.Skipped = true
				break
			}
		}
		if !memory.Components[id][idx].Status.Skipped {
			if condition != nil && *condition != "" {
				condStr := *condition
				var varMapping map[string]string
				condStr, _, varMapping = recipe.SanitizeCondition(condStr)

				expr, err := parser.ParseExpr(condStr)
				if err != nil {
					return nil, nil, err
				}

				condMemory := map[string]any{}

				for k, v := range memory.Components {
					condMemory[varMapping[k]] = v[idx]
				}

				cond, err := recipe.EvalCondition(expr, condMemory)
				if err != nil {
					return nil, nil, err
				}
				if cond == false {
					memory.Components[id][idx].Status.Skipped = true
				} else {
					memory.Components[id][idx].Status.Started = true
				}
			} else {
				memory.Components[id][idx].Status.Started = true
			}
		}
		if memory.Components[id][idx].Status.Started {

			var compInputTemplateJSON []byte
			compInputTemplate := input

			compInputTemplateJSON, err := protojson.Marshal(compInputTemplate)
			if err != nil {
				return nil, nil, err
			}
			var compInputTemplateStruct any
			err = json.Unmarshal(compInputTemplateJSON, &compInputTemplateStruct)
			if err != nil {
				return nil, nil, err
			}

			compInputStruct, err := recipe.RenderInput(compInputTemplateStruct, idx, memory.Components, memory.Inputs, memory.Secrets)
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

			*memory.Components[id][idx].Input = compInputStruct.(map[string]any)

			idxMap[len(compInputs)] = idx
			compInputs = append(compInputs, compInput)

		}
	}
	return compInputs, idxMap, nil
}

func (w *worker) processOutput(memory *recipe.TriggerMemory, id string, compOutputs []*structpb.Struct, idxMap map[int]int) ([]*recipe.ComponentItemMemory, error) {
	for compBatchIdx := range compOutputs {

		outputJSON, err := protojson.Marshal(compOutputs[compBatchIdx])
		if err != nil {
			return nil, err
		}
		var outputStruct map[string]any
		err = json.Unmarshal(outputJSON, &outputStruct)
		if err != nil {
			return nil, err
		}
		*memory.Components[id][idxMap[compBatchIdx]].Output = outputStruct
		memory.Components[id][idxMap[compBatchIdx]].Status.Completed = true
	}
	return memory.Components[id], nil
}

func (w *worker) processConnection(memory *recipe.TriggerMemory, connection *structpb.Struct) (*structpb.Struct, error) {
	conTemplateJSON, err := protojson.Marshal(connection)
	if err != nil {
		return nil, err
	}
	var conTemplateStruct any
	err = json.Unmarshal(conTemplateJSON, &conTemplateStruct)
	if err != nil {
		return nil, err
	}

	conStruct, err := recipe.RenderInput(conTemplateStruct, 0, nil, nil, memory.Secrets)
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
	return con, nil
}

// writeErrorDataPoint is a helper function that writes the error data point to
// the usage metrics table.
func (w *worker) writeErrorDataPoint(ctx context.Context, err error, span trace.Span, startTime time.Time, dataPoint *utils.PipelineUsageMetricData) {
	span.SetStatus(1, err.Error())
	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
	_ = w.writeNewDataPoint(ctx, *dataPoint)
}

// toApplicationError wraps a temporal task error in a temporal.Application
// error, adding end-user information that can be extracted by the temporal
// client.
func (w *worker) toApplicationError(err error, componentID, errType string) error {
	details := EndUserErrorDetails{
		// If no end-user message is present in the error, MessageOrErr will
		// return the string version of the error. For an end user, this extra
		// information is more actionable than no information at all.
		Message: fmt.Sprintf("Component %s failed to execute. %s", componentID, errmsg.MessageOrErr(err)),
	}
	// return fault.Wrap(err, fmsg.WithDesc("component failed to execute", issue))
	return temporal.NewApplicationErrorWithCause("component failed to execute", errType, err, details)
}

// The following constants help temporal clients to trace the origin of an
// execution error. They can be leveraged to e.g. define retry policy rules.
// This may evolve in the future to values that have more to do with the
// business domain (e.g. VendorError (non billable), InputDataError (billable),
// etc.).
const (
	PipelineWorkflowError     = "PipelineWorkflowError"
	ConnectorActivityError    = "ConnectorActivityError"
	OperatorActivityError     = "OperatorActivityError"
	PreIteratorActivityError  = "PreIteratorActivityError"
	PostIteratorActivityError = "PostIteratorActivityError"
	UsageCheckActivityError   = "UsageCheckActivityError"
	UsageCollectActivityError = "UsageCollectActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}
