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
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	MemoryRedisKey  string
	SystemVariables recipe.SystemVariables
	Mode            mgmtPB.Mode
}

// ConnectorActivityParam represents the parameters for TriggerActivity
type ConnectorActivityParam struct {
	MemoryRedisKey  string
	ID              string
	AncestorIDs     []string
	Condition       *string
	Input           *structpb.Struct
	Connection      *structpb.Struct
	DefinitionUID   uuid.UUID
	Task            string
	SystemVariables recipe.SystemVariables
}

// OperatorActivityParam represents the parameters for TriggerActivity
type OperatorActivityParam struct {
	MemoryRedisKey  string
	ID              string
	AncestorIDs     []string
	Condition       *string
	Input           *structpb.Struct
	DefinitionUID   uuid.UUID
	Task            string
	SystemVariables recipe.SystemVariables
}

type PreIteratorActivityParam struct {
	MemoryRedisKey  string
	ID              string
	Input           string
	WorkflowID      string
	SystemVariables recipe.SystemVariables
}

type PostIteratorActivityParam struct {
	MemoryRedisKey   string
	ID               string
	OutputElements   map[string]string
	ChildWorkflowIDs []string
	SystemVariables  recipe.SystemVariables
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

	r, err := recipe.LoadRecipe(sCtx, w.redisClient, param.MemoryRedisKey)
	if err != nil {
		return err
	}

	dag, err := recipe.GenerateDAG(r.Components)
	if err != nil {
		return err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return err
	}

	// The components in the same group can be executed in parallel
	for group := range orderedComp {

		futures := []workflow.Future{}
		for _, comp := range orderedComp[group] {

			switch {
			case comp.IsConnectorComponent():
				futures = append(futures, workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ConnectorActivityParam{
					ID:              comp.ID,
					AncestorIDs:     dag.GetAncestorIDs(comp.ID),
					MemoryRedisKey:  param.MemoryRedisKey,
					DefinitionUID:   uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.DefinitionName, "/")[1]),
					Task:            comp.ConnectorComponent.Task,
					Input:           comp.ConnectorComponent.Input,
					Connection:      comp.ConnectorComponent.Connection,
					Condition:       comp.ConnectorComponent.Condition,
					SystemVariables: param.SystemVariables,
				}))

			case comp.IsOperatorComponent():
				futures = append(futures, workflow.ExecuteActivity(ctx, w.OperatorActivity, &OperatorActivityParam{
					ID:              comp.ID,
					AncestorIDs:     dag.GetAncestorIDs(comp.ID),
					MemoryRedisKey:  param.MemoryRedisKey,
					DefinitionUID:   uuid.FromStringOrNil(strings.Split(comp.OperatorComponent.DefinitionName, "/")[1]),
					Task:            comp.OperatorComponent.Task,
					Input:           comp.OperatorComponent.Input,
					Condition:       comp.OperatorComponent.Condition,
					SystemVariables: param.SystemVariables,
				}))

			case comp.IsIteratorComponent():

				var childWorkflowIDs []string
				if err = workflow.ExecuteActivity(ctx, w.PreIteratorActivity, &PreIteratorActivityParam{
					ID:              comp.ID,
					MemoryRedisKey:  param.MemoryRedisKey,
					Input:           comp.IteratorComponent.Input,
					WorkflowID:      workflow.GetInfo(ctx).WorkflowExecution.ID,
					SystemVariables: param.SystemVariables,
				}).Get(ctx, &childWorkflowIDs); err != nil {
					return err
				}

				itFutures := []workflow.Future{}
				for _, childWorkflowID := range childWorkflowIDs {

					childWorkflowOptions := workflow.ChildWorkflowOptions{
						TaskQueue:                TaskQueue,
						WorkflowID:               fmt.Sprintf("%s:iterators:%s", workflow.GetInfo(ctx).WorkflowExecution.ID, comp.ID),
						WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
						RetryPolicy: &temporal.RetryPolicy{
							MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
						},
					}
					itCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

					iteratorRedisKey := fmt.Sprintf("pipeline_trigger:%s", childWorkflowID)
					itFutures = append(itFutures, workflow.ExecuteChildWorkflow(
						itCtx,
						"TriggerPipelineWorkflow",
						&TriggerPipelineWorkflowParam{
							MemoryRedisKey:  iteratorRedisKey,
							SystemVariables: param.SystemVariables,
							Mode:            mgmtPB.Mode_MODE_SYNC,
						}))

				}
				for idx := range childWorkflowIDs {
					err = itFutures[idx].Get(ctx, nil)
					if err != nil {
						logger.Error(fmt.Sprintf("unable to execute iterator workflow: %s", err.Error()))
						return err
					}
				}

				if err = workflow.ExecuteActivity(ctx, w.PostIteratorActivity, &PostIteratorActivityParam{
					ID:               comp.ID,
					MemoryRedisKey:   param.MemoryRedisKey,
					OutputElements:   comp.IteratorComponent.OutputElements,
					ChildWorkflowIDs: childWorkflowIDs,
					SystemVariables:  param.SystemVariables,
				}).Get(ctx, &childWorkflowIDs); err != nil {
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

	}

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED

	if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}
	logger.Info("TriggerPipelineWorkflow completed")
	return nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ConnectorActivityParam) error {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ConnectorActivity started")

	memory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryRedisKey)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	compInputs, idxMap, err := w.processInput(memory, param.ID, param.AncestorIDs, param.Condition, param.Input, param.DefinitionUID)
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

	compOutputs, err := execution.Execute(compInputs)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	compMem, err := w.processOutput(memory, param.ID, compOutputs, idxMap)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.MemoryRedisKey, param.ID, compMem)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	logger.Info("ConnectorActivity completed")
	return nil
}

func (w *worker) OperatorActivity(ctx context.Context, param *OperatorActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("OperatorActivity started")

	memory, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryRedisKey)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	compInputs, idxMap, err := w.processInput(memory, param.ID, param.AncestorIDs, param.Condition, param.Input, param.DefinitionUID)
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

	compOutputs, err := execution.Execute(compInputs)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	compMem, err := w.processOutput(memory, param.ID, compOutputs, idxMap)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.MemoryRedisKey, param.ID, compMem)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	logger.Info("OperatorActivity completed")
	return nil
}

func (w *worker) PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (childWorkflowIDs []string, err error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PreIteratorActivity started")

	m, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryRedisKey)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
	}
	r, err := recipe.LoadRecipe(ctx, w.redisClient, param.MemoryRedisKey)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
	}

	var iteratorRecipe *datamodel.Recipe
	for _, comp := range r.Components {
		if comp.ID == param.ID {
			iteratorRecipe = &datamodel.Recipe{
				Components: comp.IteratorComponent.Components,
			}
		}
	}

	childWorkflowIDs = make([]string, len(m.Inputs))
	for batchIdx := range m.Inputs {

		childWorkflowID := fmt.Sprintf("%s:iterators:%s", param.WorkflowID, param.ID)
		childWorkflowIDs[batchIdx] = childWorkflowID

		input, err := recipe.RenderInput(param.Input, batchIdx, m.Components, m.Inputs, m.Secrets)
		if err != nil {
			return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
		}

		subM := &recipe.TriggerMemory{
			Components: map[string]recipe.ComponentsMemory{},
			Inputs:     []recipe.InputsMemory{},
			Secrets:    m.Secrets,
			Vars:       m.Vars,
		}

		elems := make([]*recipe.ComponentItemMemory, len(input.([]any)))
		for elemIdx := range input.([]any) {
			elems[elemIdx] = &recipe.ComponentItemMemory{
				Element: input.([]any)[elemIdx],
			}

		}

		subM.Components[param.ID] = elems

		for k := range m.Components {
			subM.Components[k] = recipe.ComponentsMemory{}
			for range elems {
				subM.Components[k] = append(subM.Components[k], m.Components[k][batchIdx])
			}
		}

		for range elems {
			subM.Inputs = append(subM.Inputs, m.Inputs[batchIdx])
		}

		iteratorRedisKey := fmt.Sprintf("pipeline_trigger:%s", childWorkflowID)
		err = recipe.WriteMemoryAndRecipe(
			ctx,
			w.redisClient,
			iteratorRedisKey,
			iteratorRecipe,
			subM,
			fmt.Sprintf("%s/%s", param.SystemVariables.PipelineOwnerType, param.SystemVariables.PipelineOwnerUID),
		)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
			return nil, w.toApplicationError(err, param.ID, PreIteratorActivityError)
		}

	}

	logger.Info("PreIteratorActivity completed")
	return childWorkflowIDs, nil
}

func (w *worker) PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("PostIteratorActivity started")

	m, err := recipe.LoadMemory(ctx, w.redisClient, param.MemoryRedisKey)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return w.toApplicationError(err, param.ID, PostIteratorActivityError)
	}

	iterComp := recipe.ComponentsMemory{}
	for batchIdx := range m.Inputs {

		iteratorRedisKey := fmt.Sprintf("pipeline_trigger:%s", param.ChildWorkflowIDs[batchIdx])

		iteratorResult, err := recipe.LoadMemory(ctx, w.redisClient, iteratorRedisKey)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
			return w.toApplicationError(err, param.ID, PostIteratorActivityError)
		}

		output := recipe.ComponentIO{}
		for k, v := range param.OutputElements {
			elemVals := []any{}

			for elemIdx := range iteratorResult.Inputs {
				elemVal, err := recipe.RenderInput(v, elemIdx, iteratorResult.Components, iteratorResult.Inputs, iteratorResult.Secrets)
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
	err = recipe.WriteComponentMemory(ctx, w.redisClient, param.MemoryRedisKey, param.ID, iterComp)
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return w.toApplicationError(err, param.ID, PostIteratorActivityError)
	}

	logger.Info("PostIteratorActivity completed")
	return nil
}

func (w *worker) processInput(memory *recipe.TriggerMemory, id string, ancestorIDs []string, condition *string, input *structpb.Struct, definitionUID uuid.UUID) ([]*structpb.Struct, map[int]int, error) {
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

		for _, ancestorID := range ancestorIDs {
			if memory.Components[ancestorID][idx].Status.Skipped {
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
	ConnectorActivityError    = "ConnectorActivityError"
	OperatorActivityError     = "OperatorActivityError"
	PreIteratorActivityError  = "PreIteratorActivityError"
	PostIteratorActivityError = "PostIteratorActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}
