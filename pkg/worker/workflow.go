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
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type TriggerPipelineWorkflowRequest struct {
	PipelineInputBlobRedisKeys []string
	PipelineID                 string
	PipelineUID                uuid.UUID
	PipelineReleaseID          string
	PipelineReleaseUID         uuid.UUID
	PipelineRecipe             *datamodel.Recipe
	OwnerPermalink             string
	UserPermalink              string
	ReturnTraces               bool
	Mode                       mgmtPB.Mode
}

type TriggerPipelineWorkflowResponse struct {
	OutputBlobRedisKey string
}

// ExecutePipelineActivityRequest represents the parameters for TriggerActivity
type ExecutePipelineActivityRequest struct {
	OrderedComps   [][]*datamodel.Component
	Memory         []ItemMemory
	DAG            *dag
	BatchSize      int
	PipelineRecipe *datamodel.Recipe
}

type ExecutePipelineActivityResponse struct {
	OutputBlobRedisKeys []string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteConnectorActivityRequest struct {
	ID                 string
	InputBlobRedisKeys []string
	DefinitionName     string
	ConnectorName      string
	Task               string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteOperatorActivityRequest struct {
	ID                 string
	InputBlobRedisKeys []string
	DefinitionName     string
	Task               string
}

type ExecuteActivityResponse struct {
	OutputBlobRedisKeys []string
}

type parallelTask struct {
	comp      *datamodel.Component
	future    workflow.Future
	idxMap    map[int]int
	startTime time.Time
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

func (w *worker) GetBlob(redisKeys []string) ([]*structpb.Struct, error) {
	payloads := []*structpb.Struct{}
	for idx := range redisKeys {
		blob, err := w.redisClient.Get(context.Background(), redisKeys[idx]).Bytes()
		if err != nil {
			return nil, err
		}
		payload := &structpb.Struct{}
		err = protojson.Unmarshal(blob, payload)
		if err != nil {
			return nil, err
		}

		payloads = append(payloads, payload)

	}
	return payloads, nil
}

func (w *worker) SetBlob(inputs []*structpb.Struct) ([]string, error) {
	id, _ := uuid.NewV4()
	blobRedisKeys := []string{}
	for idx, input := range inputs {
		inputJSON, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		blobRedisKey := fmt.Sprintf("async_connector_blob:%s:%d", id.String(), idx)
		w.redisClient.Set(
			context.Background(),
			blobRedisKey,
			inputJSON,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		blobRedisKeys = append(blobRedisKeys, blobRedisKey)
	}
	return blobRedisKeys, nil
}

type ItemMemory map[string]map[string]any

func checkComponentCompleted(i map[string]any) bool {
	if status, ok := i["status"]; ok {
		if completed, ok2 := status.(map[string]any)["completed"]; ok2 {
			return completed.(bool)
		}
	}
	return false
}
func checkComponentStarted(i map[string]any) bool {
	if status, ok := i["status"]; ok {
		if started, ok2 := status.(map[string]any)["started"]; ok2 {
			return started.(bool)
		}
	}
	return false
}
func checkComponentSkipped(i map[string]any) bool {
	if status, ok := i["status"]; ok {
		if skipped, ok2 := status.(map[string]any)["skipped"]; ok2 {
			return skipped.(bool)
		}
	}
	return false
}
func checkComponentError(i map[string]any) bool {
	if status, ok := i["status"]; ok {
		if error, ok2 := status.(map[string]any)["error"]; ok2 {
			return error.(bool)
		}
	}
	return false
}

// TriggerPipelineWorkflow is a pipeline trigger workflow definition.
func (w *worker) TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowRequest) (*TriggerPipelineWorkflowResponse, error) {

	startTime := time.Now()
	eventName := "TriggerPipelineWorkflow"

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
		UserUID:            strings.Split(param.UserPermalink, "/")[1],
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
	var endComp *datamodel.Component
	compsToDAG := []*datamodel.Component{}
	for idx := range param.PipelineRecipe.Components {
		if param.PipelineRecipe.Components[idx].IsEndComponent() {
			endComp = param.PipelineRecipe.Components[idx]
		} else {
			compsToDAG = append(compsToDAG, param.PipelineRecipe.Components[idx])
		}
	}
	dag, err := GenerateDAG(compsToDAG)
	if err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}
	var startCompID string
	for _, c := range orderedComp[0] {
		if c.IsStartComponent() {
			startCompID = c.ID
		}
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	var inputs [][]byte
	pipelineInputs, err := w.GetBlob(param.PipelineInputBlobRedisKeys)
	if err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}
	batchSize := len(pipelineInputs)
	for idx := range pipelineInputs {
		inputStruct := structpb.NewStructValue(pipelineInputs[idx])

		input, err := protojson.Marshal(inputStruct)
		if err != nil {
			w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
			return nil, err
		}
		inputs = append(inputs, input)
	}

	memory := make([]ItemMemory, batchSize)
	computeTime := map[string]float32{}

	for idx := range inputs {
		memory[idx] = ItemMemory{}
		for _, comp := range param.PipelineRecipe.Components {
			computeTime[comp.ID] = 0
		}

	}

	// Setup global values
	for idx := range inputs {

		// TODO: we should prevent user name a component call `global`
		memory[idx]["global"], err = GenerateGlobalValue(param.PipelineUID, param.PipelineRecipe, param.OwnerPermalink)
		if err != nil {
			return nil, err
		}

	}
	// Setup start component values
	for idx := range inputs {
		var inputStruct map[string]any
		err := json.Unmarshal(inputs[idx], &inputStruct)
		if err != nil {
			w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
			return nil, err
		}
		memory[idx][startCompID] = inputStruct
		computeTime[startCompID] = 0
	}

	// skip start component
	orderedComp = orderedComp[1:]

	err = w.PipelineActivity(ctx, ao, &ExecutePipelineActivityRequest{
		OrderedComps:   orderedComp,
		Memory:         memory,
		DAG:            dag,
		BatchSize:      batchSize,
		PipelineRecipe: param.PipelineRecipe,
	})
	if err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}

	pipelineOutputs := []*structpb.Struct{}
	if endComp == nil {
		for idx := 0; idx < batchSize; idx++ {
			pipelineOutputs = append(pipelineOutputs, &structpb.Struct{})
		}
	} else {
		for idx := 0; idx < batchSize; idx++ {
			pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}
			for k, v := range endComp.EndComponent.Fields {
				// TODO: The end component should allow partial upstream skipping.
				// This is a temporary implementation.
				o, _ := RenderInput(v.Value, memory[idx])
				structVal, err := structpb.NewValue(o)
				if err != nil {
					return nil, err
				}
				pipelineOutput.Fields[k] = structVal

			}
			pipelineOutputs = append(pipelineOutputs, pipelineOutput)

		}
	}

	var traces map[string]*pipelinePB.Trace
	if param.ReturnTraces {
		traces, err = GenerateTraces(orderedComp, memory, computeTime, batchSize)

		if err != nil {
			w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
			return nil, err
		}
	}

	pipelineResp := &pipelinePB.TriggerUserPipelineResponse{
		Outputs: pipelineOutputs,
		Metadata: &pipelinePB.TriggerMetadata{
			Traces: traces,
		},
	}
	outputJSON, err := protojson.Marshal(pipelineResp)
	if err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}
	blobRedisKey := fmt.Sprintf("async_pipeline_response:%s", workflow.GetInfo(ctx).WorkflowExecution.ID)
	w.redisClient.Set(
		context.Background(),
		blobRedisKey,
		outputJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED

	if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}
	logger.Info("TriggerPipelineWorkflow completed")
	return &TriggerPipelineWorkflowResponse{
		OutputBlobRedisKey: blobRedisKey,
	}, nil
}

func (w *worker) PipelineActivity(ctx workflow.Context, ao workflow.ActivityOptions, param *ExecutePipelineActivityRequest) error {
	var err error
	for group := range param.OrderedComps {

		var parallelTasks = make([]*parallelTask, 0, len(param.OrderedComps[group]))

		for _, comp := range param.OrderedComps[group] {
			task := &parallelTask{
				comp:   comp,
				idxMap: map[int]int{},
			}

			var compInputs []*structpb.Struct

			for idx := 0; idx < param.BatchSize; idx++ {
				if param.Memory[idx][comp.ID] == nil {
					param.Memory[idx][comp.ID] = map[string]any{
						"input":  map[string]any{},
						"output": map[string]any{},
						"status": map[string]any{
							"started":   false,
							"completed": false,
							"skipped":   false,
						},
					}
				}

				for _, ancestorID := range param.DAG.GetAncestorIDs(comp.ID) {
					if checkComponentSkipped(param.Memory[idx][ancestorID]) {
						param.Memory[idx][comp.ID]["status"].(map[string]any)["skipped"] = true
						break
					}
				}

				if !checkComponentSkipped(param.Memory[idx][comp.ID]) {
					if comp.GetCondition() != nil && *comp.GetCondition() != "" {
						condStr := *comp.GetCondition()
						var varMapping map[string]string
						condStr, _, varMapping = SanitizeCondition(condStr)

						expr, err := parser.ParseExpr(condStr)
						if err != nil {
							return err
						}

						condMemory := map[string]any{}
						for k, v := range param.Memory[idx] {
							condMemory[varMapping[k]] = v
						}
						cond, err := EvalCondition(expr, condMemory)
						if err != nil {
							return err
						}
						if cond == false {
							param.Memory[idx][comp.ID]["status"].(map[string]any)["skipped"] = true
						} else {
							param.Memory[idx][comp.ID]["status"].(map[string]any)["started"] = true
						}
					} else {
						param.Memory[idx][comp.ID]["status"].(map[string]any)["started"] = true
					}
				}
				if checkComponentStarted(param.Memory[idx][comp.ID]) {

					// Render input

					if comp.IsConnectorComponent() || comp.IsOperatorComponent() {
						var compInputTemplateJSON []byte
						var compInputTemplate *structpb.Struct
						if comp.IsConnectorComponent() {
							compInputTemplate = comp.ConnectorComponent.Input
							// TODO: remove this hardcode injection
							// blockchain-numbers
							if comp.ConnectorComponent.DefinitionName == "connector-definitions/70d8664a-d512-4517-a5e8-5d4da81756a7" {

								recipeByte, err := json.Marshal(param.PipelineRecipe)
								if err != nil {
									return err
								}
								recipePb := &structpb.Struct{}
								err = protojson.Unmarshal(recipeByte, recipePb)
								if err != nil {
									return err
								}
								metadata, err := structpb.NewValue(map[string]any{
									"pipeline": map[string]any{
										"uid":    "${global.pipeline.uid}",
										"recipe": "${global.pipeline.recipe}",
									},
									"owner": map[string]any{
										"uid": "${global.owner.uid}",
									},
								})
								if err != nil {
									return err
								}
								if compInputTemplate == nil {
									compInputTemplate = &structpb.Struct{}
								}
								compInputTemplate.Fields["metadata"] = metadata
							}
						} else {
							compInputTemplate = comp.OperatorComponent.Input
						}

						compInputTemplateJSON, err = protojson.Marshal(compInputTemplate)
						if err != nil {
							return err
						}
						var compInputTemplateStruct any
						err = json.Unmarshal(compInputTemplateJSON, &compInputTemplateStruct)
						if err != nil {
							return err
						}

						compInputStruct, err := RenderInput(compInputTemplateStruct, param.Memory[idx])
						if err != nil {
							return err
						}
						compInputJSON, err := json.Marshal(compInputStruct)
						if err != nil {
							return err
						}

						compInput := &structpb.Struct{}
						err = protojson.Unmarshal([]byte(compInputJSON), compInput)
						if err != nil {
							return err
						}

						param.Memory[idx][comp.ID]["input"] = compInputStruct
						task.idxMap[len(compInputs)] = idx
						compInputs = append(compInputs, compInput)

					}

					switch {
					case comp.IsConnectorComponent() && comp.ConnectorComponent.ConnectorName != "":
						inputBlobRedisKeys, err := w.SetBlob(compInputs)
						if err != nil {
							return err
						}
						for idx := range inputBlobRedisKeys {
							defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
						}

						ctx = workflow.WithActivityOptions(ctx, ao)

						task.startTime = time.Now()
						task.future = workflow.ExecuteActivity(ctx, w.ConnectorActivity, &ExecuteConnectorActivityRequest{
							ID:                 comp.ID,
							InputBlobRedisKeys: inputBlobRedisKeys,
							DefinitionName:     comp.ConnectorComponent.DefinitionName,
							ConnectorName:      comp.ConnectorComponent.ConnectorName,
							Task:               comp.ConnectorComponent.Task,
						})
						parallelTasks = append(parallelTasks, task)

					case comp.IsOperatorComponent():
						inputBlobRedisKeys, err := w.SetBlob(compInputs)
						if err != nil {
							return err
						}
						for idx := range inputBlobRedisKeys {
							defer w.redisClient.Del(context.Background(), inputBlobRedisKeys[idx])
						}

						ctx = workflow.WithActivityOptions(ctx, ao)
						task.startTime = time.Now()
						task.future = workflow.ExecuteActivity(ctx, w.OperatorActivity, &ExecuteOperatorActivityRequest{
							ID:                 comp.ID,
							InputBlobRedisKeys: inputBlobRedisKeys,
							DefinitionName:     comp.OperatorComponent.DefinitionName,
							Task:               comp.OperatorComponent.Task,
						})
						parallelTasks = append(parallelTasks, task)

					case comp.IsIteratorComponent():

						input, err := RenderInput(comp.IteratorComponent.Input, param.Memory[idx])
						if err != nil {
							return err
						}
						batchSize := len(input.([]any))

						subMemory := make([]ItemMemory, batchSize)
						for elemIdx, elem := range input.([]any) {
							b, _ := json.Marshal(param.Memory[idx])
							_ = json.Unmarshal(b, &subMemory[elemIdx])

							subMemory[elemIdx][comp.ID]["element"] = elem
						}

						comps := []*datamodel.Component{{ID: comp.ID}}
						comps = append(comps, comp.IteratorComponent.Components...)

						dag, err := GenerateDAG(comps)
						if err != nil {
							return err
						}

						orderedComp, err := dag.TopologicalSort()

						if err != nil {
							return err
						}

						err = w.PipelineActivity(ctx, ao, &ExecutePipelineActivityRequest{
							OrderedComps:   orderedComp,
							Memory:         subMemory,
							DAG:            dag,
							BatchSize:      batchSize,
							PipelineRecipe: param.PipelineRecipe,
						})
						if err != nil {
							return err
						}

						param.Memory[idx][comp.ID]["output"] = map[string]any{}
						for k, v := range comp.IteratorComponent.OutputElements {
							elemVals := []any{}

							for elemIdx := range input.([]any) {
								elemVal, err := RenderInput(v, subMemory[elemIdx])
								if err != nil {
									return err
								}
								elemVals = append(elemVals, elemVal)

							}
							param.Memory[idx][comp.ID]["output"].(map[string]any)[k] = elemVals
						}
					}

				}
			}
		}

		for idx := range parallelTasks {
			if parallelTasks[idx].comp.IsConnectorComponent() || parallelTasks[idx].comp.IsOperatorComponent() {
				var result ExecuteActivityResponse
				if err := parallelTasks[idx].future.Get(ctx, &result); err != nil {
					return err
				}

				outputs, err := w.GetBlob(result.OutputBlobRedisKeys)
				for idx := range result.OutputBlobRedisKeys {
					defer w.redisClient.Del(context.Background(), result.OutputBlobRedisKeys[idx])
				}
				if err != nil {
					return err
				}
				for compBatchIdx := range outputs {

					outputJSON, err := protojson.Marshal(outputs[compBatchIdx])
					if err != nil {
						return err
					}
					var outputStruct map[string]any
					err = json.Unmarshal(outputJSON, &outputStruct)
					if err != nil {
						return err
					}
					param.Memory[parallelTasks[idx].idxMap[compBatchIdx]][parallelTasks[idx].comp.ID]["output"] = outputStruct
					param.Memory[parallelTasks[idx].idxMap[compBatchIdx]][parallelTasks[idx].comp.ID]["status"].(map[string]any)["completed"] = true
				}
			}
		}
	}
	return nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteActivityResponse, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ConnectorActivity started")

	compInputs, err := w.GetBlob(param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	con, err := w.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(param.DefinitionName, "/")[1]), nil, nil)
	if err != nil {
		return nil, err
	}

	dbConnector, err := w.repository.GetConnectorByUIDAdmin(ctx, uuid.FromStringOrNil(strings.Split(param.ConnectorName, "/")[1]), false)
	if err != nil {
		return nil, err
	}

	configuration := func() *structpb.Struct {
		if dbConnector.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbConnector.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			// TODO: optimize this
			str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
			str.Fields["instill_mgmt_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort))
			return &str
		}
		str := structpb.Struct{Fields: make(map[string]*structpb.Value)}
		// TODO: optimize this
		str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
		str.Fields["instill_mgmt_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort))
		return nil
	}()

	execution, err := w.connector.CreateExecution(uuid.FromStringOrNil(con.Uid), param.Task, configuration, logger)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, ConnectorActivityError)
	}
	compOutputs, err := execution.ExecuteWithValidation(compInputs)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, ConnectorActivityError)
	}

	outputBlobRedisKeys, err := w.SetBlob(compOutputs)
	if err != nil {
		return nil, err
	}

	logger.Info("ConnectorActivity completed")
	return &ExecuteActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
}

func (w *worker) OperatorActivity(ctx context.Context, param *ExecuteOperatorActivityRequest) (*ExecuteActivityResponse, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("OperatorActivity started")

	compInputs, err := w.GetBlob(param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	op, err := w.operator.GetOperatorDefinitionByUID(uuid.FromStringOrNil(strings.Split(param.DefinitionName, "/")[1]), nil)
	if err != nil {
		return nil, err
	}

	execution, err := w.operator.CreateExecution(uuid.FromStringOrNil(op.Uid), param.Task, nil, logger)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, OperatorActivityError)
	}
	compOutputs, err := execution.ExecuteWithValidation(compInputs)
	if err != nil {
		return nil, w.toApplicationError(err, param.ID, OperatorActivityError)
	}

	outputBlobRedisKeys, err := w.SetBlob(compOutputs)
	if err != nil {
		return nil, err
	}
	return &ExecuteActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
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
