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
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type TriggerPipelineWorkflowParam struct {
	MemoryRedisKey     string
	PipelineID         string
	PipelineUID        uuid.UUID
	PipelineReleaseID  string
	PipelineReleaseUID uuid.UUID
	OwnerPermalink     string
	UserUID            uuid.UUID
	Mode               mgmtPB.Mode
	InputKey           string
}

// ConnectorActivityParam represents the parameters for TriggerActivity
type ConnectorActivityParam struct {
	MemoryRedisKey string
	ID             string
	AncestorIDs    []string
	Condition      *string
	Input          *structpb.Struct
	Connection     *structpb.Struct
	DefinitionUID  uuid.UUID
	Task           string
	InputKey       string
}

// OperatorActivityParam represents the parameters for TriggerActivity
type OperatorActivityParam struct {
	MemoryRedisKey string
	ID             string
	AncestorIDs    []string
	Condition      *string
	Input          *structpb.Struct
	DefinitionUID  uuid.UUID
	Task           string
	InputKey       string
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

	namespace := strings.Split(param.OwnerPermalink, "/")[0]
	var ownerType mgmtPB.OwnerType
	switch namespace {
	case "organizations":
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION
	case "users":
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           strings.Split(param.OwnerPermalink, "/")[1],
		OwnerType:          ownerType,
		UserUID:            param.UserUID.String(),
		UserType:           mgmtPB.OwnerType_OWNER_TYPE_USER, // TODO: currently only support /users type, will change after beta
		TriggerMode:        param.Mode,
		PipelineID:         param.PipelineID,
		PipelineUID:        param.PipelineUID.String(),
		PipelineReleaseID:  param.PipelineReleaseID,
		PipelineReleaseUID: param.PipelineReleaseUID.String(),
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
					ID:             comp.ID,
					AncestorIDs:    dag.GetAncestorIDs(comp.ID),
					MemoryRedisKey: param.MemoryRedisKey,
					DefinitionUID:  uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.DefinitionName, "/")[1]),
					Task:           comp.ConnectorComponent.Task,
					Input:          comp.ConnectorComponent.Input,
					Connection:     comp.ConnectorComponent.Connection,
					Condition:      comp.ConnectorComponent.Condition,
					InputKey:       param.InputKey,
				}))

			case comp.IsOperatorComponent():
				futures = append(futures, workflow.ExecuteActivity(ctx, w.OperatorActivity, &OperatorActivityParam{
					ID:             comp.ID,
					AncestorIDs:    dag.GetAncestorIDs(comp.ID),
					MemoryRedisKey: param.MemoryRedisKey,
					DefinitionUID:  uuid.FromStringOrNil(strings.Split(comp.OperatorComponent.DefinitionName, "/")[1]),
					Task:           comp.OperatorComponent.Task,
					Input:          comp.OperatorComponent.Input,
					Condition:      comp.OperatorComponent.Condition,
					InputKey:       param.InputKey,
				}))

			case comp.IsIteratorComponent():
				childWorkflowOptions := workflow.ChildWorkflowOptions{
					TaskQueue:                TaskQueue,
					WorkflowID:               fmt.Sprintf("%s:iterators:%s", workflow.GetInfo(ctx).WorkflowExecution.ID, comp.ID),
					WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
					RetryPolicy: &temporal.RetryPolicy{
						MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
					},
				}
				ctx = workflow.WithChildOptions(ctx, childWorkflowOptions)

				m, err := recipe.LoadMemory(sCtx, w.redisClient, param.MemoryRedisKey)
				if err != nil {
					logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
					return err
				}

				// TODO: use batch and put the preprocessing in activity
				iterComp := recipe.ComponentsMemory{}
				for batchIdx := range m.Inputs {

					input, err := recipe.RenderInput(comp.IteratorComponent.Input, batchIdx, m.Components, m.Inputs, m.Secrets, param.InputKey)
					if err != nil {
						return err
					}

					subM := &recipe.TriggerMemory{
						Components: map[string]recipe.ComponentsMemory{},
						Inputs:     []recipe.InputsMemory{},
						Secrets:    m.Secrets,
						Vars:       m.Vars,
						InputKey:   "request",
					}

					elems := make([]*recipe.ComponentItemMemory, len(input.([]any)))
					for elemIdx := range input.([]any) {
						elems[elemIdx] = &recipe.ComponentItemMemory{
							Element: input.([]any)[elemIdx],
						}

					}

					subM.Components[comp.ID] = elems

					for k := range m.Components {
						subM.Components[k] = recipe.ComponentsMemory{}
						for range elems {
							subM.Components[k] = append(subM.Components[k], m.Components[k][batchIdx])
						}
					}

					for range elems {
						subM.Inputs = append(subM.Inputs, m.Inputs[batchIdx])
					}

					iteratorRedisKey := fmt.Sprintf("pipeline_trigger:%s", childWorkflowOptions.WorkflowID)
					err = recipe.WriteMemoryAndRecipe(
						sCtx,
						w.redisClient,
						iteratorRedisKey,
						&datamodel.Recipe{
							Components: comp.IteratorComponent.Components,
						},
						subM,
						param.OwnerPermalink,
					)
					if err != nil {
						logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
						return err
					}

					err = workflow.ExecuteChildWorkflow(
						ctx,
						"TriggerPipelineWorkflow",
						&TriggerPipelineWorkflowParam{
							MemoryRedisKey:     iteratorRedisKey,
							PipelineID:         param.PipelineID,
							PipelineUID:        param.PipelineUID,
							PipelineReleaseID:  param.PipelineReleaseID,
							PipelineReleaseUID: param.PipelineReleaseUID,
							OwnerPermalink:     param.OwnerPermalink,
							UserUID:            param.UserUID,
							Mode:               mgmtPB.Mode_MODE_SYNC,
						}).Get(ctx, nil)
					if err != nil {
						logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
						return err
					}

					iteratorResult, err := recipe.LoadMemory(sCtx, w.redisClient, iteratorRedisKey)
					if err != nil {
						logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
						return err
					}

					output := recipe.ComponentIO{}
					for k, v := range comp.IteratorComponent.OutputElements {
						elemVals := []any{}

						for elemIdx := range input.([]any) {
							elemVal, err := recipe.RenderInput(v, elemIdx, iteratorResult.Components, iteratorResult.Inputs, iteratorResult.Secrets, param.InputKey)
							if err != nil {
								return err
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
				err = recipe.WriteComponentMemory(sCtx, w.redisClient, param.MemoryRedisKey, comp.ID, iterComp)
				if err != nil {
					logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
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

	compInputs, idxMap, err := w.processInput(memory, param.ID, param.AncestorIDs, param.Condition, param.Input, param.InputKey, param.DefinitionUID)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	con, err := w.processConnection(memory, param.Connection)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	// TODO
	// vars: header_authorization, instill_user_uid, instill_model_backend, instill_mgmt_backend
	execution, err := w.connector.CreateExecution(param.DefinitionUID, param.Task, con, logger)
	if err != nil {
		return w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	compOutputs, err := execution.ExecuteWithValidation(compInputs)
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

	compInputs, idxMap, err := w.processInput(memory, param.ID, param.AncestorIDs, param.Condition, param.Input, param.InputKey, param.DefinitionUID)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	// TODO
	// vars: header_authorization, instill_user_uid, instill_model_backend, instill_mgmt_backend
	execution, err := w.operator.CreateExecution(param.DefinitionUID, param.Task, nil, logger)
	if err != nil {
		return w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	compOutputs, err := execution.ExecuteWithValidation(compInputs)
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

func (w *worker) processInput(memory *recipe.TriggerMemory, id string, ancestorIDs []string, condition *string, input *structpb.Struct, inputKey string, definitionUID uuid.UUID) ([]*structpb.Struct, map[int]int, error) {
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
			var compInputTemplate *structpb.Struct

			compInputTemplate = input
			// TODO: remove this hardcode injection
			// blockchain-numbers
			if definitionUID.String() == "70d8664a-d512-4517-a5e8-5d4da81756a7" {

				metadata, err := structpb.NewValue(map[string]any{
					"pipeline": map[string]any{
						"uid": "${vars._PIPELINE_UID}",
						"r":   "${vars._PIPELINE_RECIPE}",
					},
					"owner": map[string]any{
						"uid": "${vars._OWNER_UID}",
					},
				})
				if err != nil {
					return nil, nil, err
				}
				if compInputTemplate == nil {
					compInputTemplate = &structpb.Struct{}
				}
				compInputTemplate.Fields["metadata"] = metadata
			}

			compInputTemplateJSON, err := protojson.Marshal(compInputTemplate)
			if err != nil {
				return nil, nil, err
			}
			var compInputTemplateStruct any
			err = json.Unmarshal(compInputTemplateJSON, &compInputTemplateStruct)
			if err != nil {
				return nil, nil, err
			}

			compInputStruct, err := recipe.RenderInput(compInputTemplateStruct, idx, memory.Components, memory.Inputs, memory.Secrets, inputKey)
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

	conStruct, err := recipe.RenderInput(conTemplateStruct, 0, nil, nil, memory.Secrets, "")
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
	ConnectorActivityError = "ConnectorActivityError"
	OperatorActivityError  = "OperatorActivityError"
)

// EndUserErrorDetails provides a structured way to add an end-user error
// message to a temporal.ApplicationError.
type EndUserErrorDetails struct {
	Message string
}
