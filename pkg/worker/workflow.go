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
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/errmsg"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// Note: These codes will be refactored soon

type TriggerPipelineWorkflowRequest struct {
	PipelineInputBlobRedisKeys []string
	PipelineID                 string
	PipelineUID                uuid.UUID
	PipelineReleaseID          string
	PipelineReleaseUID         uuid.UUID
	PipelineRecipe             *datamodel.Recipe
	OwnerPermalink             string
	UserUID                    uuid.UUID
	ReturnTraces               bool
	Mode                       mgmtPB.Mode
	MetadataRedisKey           string
}

type TriggerPipelineWorkflowResponse struct {
	OutputBlobRedisKey string
}

// ExecuteDAGActivityRequest represents the parameters for TriggerActivity
type ExecuteDAGActivityRequest struct {
	OrderedComps        [][]*datamodel.Component
	DAG                 *dag
	BatchSize           int
	PipelineRecipe      *datamodel.Recipe
	MemoryBlobRedisKeys []string
	OwnerPermalink      string
	UserUID             uuid.UUID
	MetadataRedisKey    string
}
type ExecuteDAGActivityResponse struct {
	MemoryBlobRedisKeys []string
}

type ExecuteTriggerStartActivityRequest struct {
	PipelineInputBlobRedisKeys []string
	PipelineRecipe             *datamodel.Recipe
	PipelineUID                uuid.UUID
	OwnerPermalink             string
	UserUID                    uuid.UUID
}
type ExecuteTriggerStartActivityResponse struct {
	MemoryBlobRedisKeys []string
	OrderedComps        [][]*datamodel.Component
	DAG                 *dag
	BatchSize           int
	EndComponent        *datamodel.Component
	ComputeTime         map[string]float32
}

type ExecuteTriggerEndActivityRequest struct {
	MemoryBlobRedisKeys []string
	OrderedComps        [][]*datamodel.Component
	BatchSize           int
	EndComponent        *datamodel.Component
	ComputeTime         map[string]float32
	ReturnTraces        bool
	WorkflowExecutionID string //workflow.GetInfo(ctx).WorkflowExecution.ID
	UserUID             uuid.UUID
}
type ExecuteTriggerEndActivityResponse struct {
	BlobRedisKey string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteConnectorActivityRequest struct {
	ID                 string
	InputBlobRedisKeys []string
	DefinitionName     string
	ConnectorName      string
	PipelineMetadata   PipelineMetadataStruct
	Task               string
	UserUID            uuid.UUID
	MetadataRedisKey   string
}

// ExecuteConnectorActivityRequest represents the parameters for TriggerActivity
type ExecuteOperatorActivityRequest struct {
	ID                 string
	InputBlobRedisKeys []string
	DefinitionName     string
	PipelineMetadata   PipelineMetadataStruct
	Task               string
	UserUID            uuid.UUID
	MetadataRedisKey   string
}

type ExecuteActivityResponse struct {
	OutputBlobRedisKeys []string
}

type PipelineMetadataStruct struct {
	UserUID string
}

type WorkflowMeta struct {
	HeaderAuthorization string `json:"header_authorization"`
}

var tracer = otel.Tracer("pipeline-backend.temporal.tracer")

func (w *worker) GetMemoryBlob(ctx context.Context, redisKeys []string) ([]ItemMemory, error) {
	payloads := []ItemMemory{}
	for idx := range redisKeys {
		blob, err := w.redisClient.Get(ctx, redisKeys[idx]).Bytes()
		if err != nil {
			return nil, err
		}
		payload := ItemMemory{}
		err = json.Unmarshal(blob, &payload)
		if err != nil {
			return nil, err
		}

		payloads = append(payloads, payload)

	}
	return payloads, nil
}

func (w *worker) SetMemoryBlob(ctx context.Context, inputs []ItemMemory) ([]string, error) {
	id, _ := uuid.NewV4()
	blobRedisKeys := []string{}
	for idx, input := range inputs {
		inputJSON, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}

		blobRedisKey := fmt.Sprintf("pipeline_memory_blob:%s:%d", id.String(), idx)
		w.redisClient.Set(
			ctx,
			blobRedisKey,
			inputJSON,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		blobRedisKeys = append(blobRedisKeys, blobRedisKey)
	}
	return blobRedisKeys, nil
}

func (w *worker) GetBlob(ctx context.Context, redisKeys []string) ([]*structpb.Struct, error) {
	payloads := []*structpb.Struct{}
	for idx := range redisKeys {
		blob, err := w.redisClient.Get(ctx, redisKeys[idx]).Bytes()
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

func (w *worker) SetBlob(ctx context.Context, inputs []*structpb.Struct) ([]string, error) {
	id, _ := uuid.NewV4()
	blobRedisKeys := []string{}
	for idx, input := range inputs {
		inputJSON, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		blobRedisKey := fmt.Sprintf("pipeline_blob:%s:%d", id.String(), idx)
		w.redisClient.Set(
			ctx,
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

	var startResult ExecuteTriggerStartActivityResponse
	if err := workflow.ExecuteActivity(ctx, w.TriggerStartActivity, &ExecuteTriggerStartActivityRequest{
		PipelineInputBlobRedisKeys: param.PipelineInputBlobRedisKeys,
		PipelineRecipe:             param.PipelineRecipe,
		PipelineUID:                param.PipelineUID,
		OwnerPermalink:             param.OwnerPermalink,
		UserUID:                    param.UserUID,
	}).Get(ctx, &startResult); err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}

	var dagResult ExecuteDAGActivityResponse
	if err := workflow.ExecuteActivity(ctx, w.DAGActivity, &ExecuteDAGActivityRequest{
		OrderedComps:        startResult.OrderedComps,
		MemoryBlobRedisKeys: startResult.MemoryBlobRedisKeys,
		DAG:                 startResult.DAG,
		BatchSize:           startResult.BatchSize,
		PipelineRecipe:      param.PipelineRecipe,
		OwnerPermalink:      param.OwnerPermalink,
		UserUID:             param.UserUID,
		MetadataRedisKey:    param.MetadataRedisKey,
	}).Get(ctx, &dagResult); err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}

	var endResult ExecuteTriggerEndActivityResponse
	if err := workflow.ExecuteActivity(ctx, w.TriggerEndActivity, &ExecuteTriggerEndActivityRequest{
		MemoryBlobRedisKeys: dagResult.MemoryBlobRedisKeys,
		OrderedComps:        startResult.OrderedComps,
		BatchSize:           startResult.BatchSize,
		EndComponent:        startResult.EndComponent,
		ComputeTime:         startResult.ComputeTime,
		ReturnTraces:        param.ReturnTraces,
		WorkflowExecutionID: workflow.GetInfo(ctx).WorkflowExecution.ID,
		UserUID:             param.UserUID,
	}).Get(ctx, &endResult); err != nil {
		w.writeErrorDataPoint(sCtx, err, span, startTime, &dataPoint)
		return nil, err
	}

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED

	if err := w.writeNewDataPoint(sCtx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}
	logger.Info("TriggerPipelineWorkflow completed")
	return &TriggerPipelineWorkflowResponse{
		OutputBlobRedisKey: endResult.BlobRedisKey,
	}, nil
}

func (w *worker) TriggerStartActivity(ctx context.Context, param *ExecuteTriggerStartActivityRequest) (*ExecuteTriggerStartActivityResponse, error) {

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})

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
		return nil, err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return nil, err
	}
	var startCompID string
	for _, c := range orderedComp[0] {
		if c.IsStartComponent() {
			startCompID = c.ID
		}
	}

	var inputs [][]byte
	pipelineInputs, err := w.GetBlob(ctx, param.PipelineInputBlobRedisKeys)
	if err != nil {
		return nil, err
	}
	batchSize := len(pipelineInputs)
	for idx := range pipelineInputs {
		inputStruct := structpb.NewStructValue(pipelineInputs[idx])

		input, err := protojson.Marshal(inputStruct)
		if err != nil {
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
			return nil, err
		}
		memory[idx][startCompID] = inputStruct
		computeTime[startCompID] = 0
	}

	// skip start component
	orderedComp = orderedComp[1:]

	memoryBlobRedisKeys, err := w.SetMemoryBlob(ctx, memory)
	if err != nil {
		return nil, err
	}
	return &ExecuteTriggerStartActivityResponse{
		MemoryBlobRedisKeys: memoryBlobRedisKeys,
		OrderedComps:        orderedComp,
		DAG:                 dag,
		BatchSize:           batchSize,
		EndComponent:        endComp,
		ComputeTime:         computeTime,
	}, nil
}

func (w *worker) TriggerEndActivity(ctx context.Context, param *ExecuteTriggerEndActivityRequest) (*ExecuteTriggerEndActivityResponse, error) {

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})

	memory, err := w.GetMemoryBlob(ctx, param.MemoryBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	pipelineOutputs := []*structpb.Struct{}
	if param.EndComponent == nil {
		for idx := 0; idx < param.BatchSize; idx++ {
			pipelineOutputs = append(pipelineOutputs, &structpb.Struct{})
		}
	} else {
		for idx := 0; idx < param.BatchSize; idx++ {
			pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}
			for k, v := range param.EndComponent.EndComponent.Fields {
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
		traces, err = GenerateTraces(param.OrderedComps, memory, param.ComputeTime, param.BatchSize)

		if err != nil {
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
		return nil, err
	}
	blobRedisKey := fmt.Sprintf("async_pipeline_response:%s", param.WorkflowExecutionID)
	w.redisClient.Set(
		ctx,
		blobRedisKey,
		outputJSON,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)
	return &ExecuteTriggerEndActivityResponse{
		BlobRedisKey: blobRedisKey,
	}, nil
}
func (w *worker) DAGActivity(ctx context.Context, param *ExecuteDAGActivityRequest) (*ExecuteDAGActivityResponse, error) {

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})

	memory, err := w.GetMemoryBlob(ctx, param.MemoryBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	for group := range param.OrderedComps {

		eg := errgroup.Group{}
		for _, comp := range param.OrderedComps[group] {
			idxMap := map[int]int{}

			var compInputs []*structpb.Struct

			for idx := 0; idx < param.BatchSize; idx++ {
				if memory[idx][comp.ID] == nil {
					memory[idx][comp.ID] = map[string]any{
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
					if checkComponentSkipped(memory[idx][ancestorID]) {
						memory[idx][comp.ID]["status"].(map[string]any)["skipped"] = true
						break
					}
				}

				if !checkComponentSkipped(memory[idx][comp.ID]) {
					if comp.GetCondition() != nil && *comp.GetCondition() != "" {
						condStr := *comp.GetCondition()
						var varMapping map[string]string
						condStr, _, varMapping = SanitizeCondition(condStr)

						expr, err := parser.ParseExpr(condStr)
						if err != nil {
							return nil, err
						}

						condMemory := map[string]any{}
						for k, v := range memory[idx] {
							condMemory[varMapping[k]] = v
						}
						cond, err := EvalCondition(expr, condMemory)
						if err != nil {
							return nil, err
						}
						if cond == false {
							memory[idx][comp.ID]["status"].(map[string]any)["skipped"] = true
						} else {
							memory[idx][comp.ID]["status"].(map[string]any)["started"] = true
						}
					} else {
						memory[idx][comp.ID]["status"].(map[string]any)["started"] = true
					}
				}
				if checkComponentStarted(memory[idx][comp.ID]) {

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
									return nil, err
								}
								recipePb := &structpb.Struct{}
								err = protojson.Unmarshal(recipeByte, recipePb)
								if err != nil {
									return nil, err
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
									return nil, err
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
							return nil, err
						}
						var compInputTemplateStruct any
						err = json.Unmarshal(compInputTemplateJSON, &compInputTemplateStruct)
						if err != nil {
							return nil, err
						}

						compInputStruct, err := RenderInput(compInputTemplateStruct, memory[idx])
						if err != nil {
							return nil, err
						}
						compInputJSON, err := json.Marshal(compInputStruct)
						if err != nil {
							return nil, err
						}

						compInput := &structpb.Struct{}
						err = protojson.Unmarshal([]byte(compInputJSON), compInput)
						if err != nil {
							return nil, err
						}

						memory[idx][comp.ID]["input"] = compInputStruct
						idxMap[len(compInputs)] = idx
						compInputs = append(compInputs, compInput)

					}

					switch {
					case comp.IsConnectorComponent() && comp.ConnectorComponent.ConnectorName != "":
						inputBlobRedisKeys, err := w.SetBlob(ctx, compInputs)
						if err != nil {
							return nil, err
						}
						for idx := range inputBlobRedisKeys {
							defer w.redisClient.Del(ctx, inputBlobRedisKeys[idx])
						}
						id := comp.ID
						definitionName := comp.ConnectorComponent.DefinitionName
						connectorName := comp.ConnectorComponent.ConnectorName
						task := comp.ConnectorComponent.Task

						eg.Go(func() error {
							result, err := w.ConnectorActivity(ctx, &ExecuteConnectorActivityRequest{
								ID:                 id,
								InputBlobRedisKeys: inputBlobRedisKeys,
								DefinitionName:     definitionName,
								ConnectorName:      connectorName,
								PipelineMetadata: PipelineMetadataStruct{
									UserUID: param.UserUID.String(),
								},
								Task:             task,
								UserUID:          param.UserUID,
								MetadataRedisKey: param.MetadataRedisKey,
							})
							if err != nil {
								return err
							}

							outputs, err := w.GetBlob(ctx, result.OutputBlobRedisKeys)
							for idx := range result.OutputBlobRedisKeys {
								defer w.redisClient.Del(ctx, result.OutputBlobRedisKeys[idx])
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

								memory[idxMap[compBatchIdx]][id]["output"] = outputStruct
								memory[idxMap[compBatchIdx]][id]["status"].(map[string]any)["completed"] = true
							}
							return nil
						})

					case comp.IsOperatorComponent():
						inputBlobRedisKeys, err := w.SetBlob(ctx, compInputs)
						if err != nil {
							return nil, err
						}
						for idx := range inputBlobRedisKeys {
							defer w.redisClient.Del(ctx, inputBlobRedisKeys[idx])
						}
						id := comp.ID
						definitionName := comp.OperatorComponent.DefinitionName
						task := comp.OperatorComponent.Task
						eg.Go(func() error {
							result, err := w.OperatorActivity(ctx, &ExecuteOperatorActivityRequest{
								ID:                 id,
								InputBlobRedisKeys: inputBlobRedisKeys,
								DefinitionName:     definitionName,
								PipelineMetadata: PipelineMetadataStruct{
									UserUID: param.UserUID.String(),
								},
								Task:             task,
								UserUID:          param.UserUID,
								MetadataRedisKey: param.MetadataRedisKey,
							})
							if err != nil {
								return err
							}

							outputs, err := w.GetBlob(ctx, result.OutputBlobRedisKeys)
							for idx := range result.OutputBlobRedisKeys {
								defer w.redisClient.Del(ctx, result.OutputBlobRedisKeys[idx])
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
								memory[idxMap[compBatchIdx]][id]["output"] = outputStruct
								memory[idxMap[compBatchIdx]][id]["status"].(map[string]any)["completed"] = true
							}
							return nil
						})

					case comp.IsIteratorComponent():

						input, err := RenderInput(comp.IteratorComponent.Input, memory[idx])
						if err != nil {
							return nil, err
						}
						batchSize := len(input.([]any))

						subMemory := make([]ItemMemory, batchSize)
						for elemIdx, elem := range input.([]any) {
							b, _ := json.Marshal(memory[idx])
							_ = json.Unmarshal(b, &subMemory[elemIdx])

							subMemory[elemIdx][comp.ID]["element"] = elem
						}

						comps := []*datamodel.Component{{ID: comp.ID}}
						comps = append(comps, comp.IteratorComponent.Components...)

						dag, err := GenerateDAG(comps)
						if err != nil {
							return nil, err
						}

						orderedComp, err := dag.TopologicalSort()
						if err != nil {
							return nil, err
						}

						subMemoryBlobRedisKeys, err := w.SetMemoryBlob(ctx, subMemory)
						if err != nil {
							return nil, err
						}

						result, err := w.DAGActivity(ctx, &ExecuteDAGActivityRequest{
							OrderedComps:        orderedComp,
							MemoryBlobRedisKeys: subMemoryBlobRedisKeys,
							DAG:                 dag,
							BatchSize:           batchSize,
							PipelineRecipe:      param.PipelineRecipe,
							OwnerPermalink:      param.OwnerPermalink,
							UserUID:             param.UserUID,
						})
						if err != nil {
							return nil, err
						}

						subMemory, err = w.GetMemoryBlob(ctx, result.MemoryBlobRedisKeys)
						if err != nil {
							return nil, err
						}

						memory[idx][comp.ID]["output"] = map[string]any{}
						for k, v := range comp.IteratorComponent.OutputElements {
							elemVals := []any{}

							for elemIdx := range input.([]any) {
								elemVal, err := RenderInput(v, subMemory[elemIdx])
								if err != nil {
									return nil, err
								}
								elemVals = append(elemVals, elemVal)

							}
							memory[idx][comp.ID]["output"].(map[string]any)[k] = elemVals
						}
					}

				}
			}
		}

		if err := eg.Wait(); err != nil {
			return nil, err
		}

	}
	// TODO
	memoryBlobRedisKeys, err := w.SetMemoryBlob(ctx, memory)
	if err != nil {
		return nil, err
	}

	return &ExecuteDAGActivityResponse{
		MemoryBlobRedisKeys: memoryBlobRedisKeys,
	}, nil
}

func (w *worker) ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteActivityResponse, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("ConnectorActivity started")

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})

	compInputs, err := w.GetBlob(ctx, param.InputBlobRedisKeys)
	if err != nil {
		return nil, err
	}

	con, err := w.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(param.DefinitionName, "/")[1]), nil, nil)
	if err != nil {
		return nil, err
	}

	dbConnector, err := w.repository.GetConnectorByUID(ctx, uuid.FromStringOrNil(strings.Split(param.ConnectorName, "/")[1]), false)
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
			m, err := w.redisClient.Get(ctx, param.MetadataRedisKey).Bytes()
			if err != nil {
				return nil
			}
			workflowMeta := WorkflowMeta{}
			err = json.Unmarshal(m, &workflowMeta)
			if err != nil {
				return nil
			}

			str.Fields["header_authorization"] = structpb.NewStringValue(workflowMeta.HeaderAuthorization)
			str.Fields["instill_user_uid"] = structpb.NewStringValue(param.PipelineMetadata.UserUID)
			str.Fields["instill_model_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort))
			str.Fields["instill_mgmt_backend"] = structpb.NewStringValue(fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort))
			return &str
		}
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

	outputBlobRedisKeys, err := w.SetBlob(ctx, compOutputs)
	if err != nil {
		return nil, err
	}

	logger.Info("ConnectorActivity completed")
	return &ExecuteActivityResponse{OutputBlobRedisKeys: outputBlobRedisKeys}, nil
}

func (w *worker) OperatorActivity(ctx context.Context, param *ExecuteOperatorActivityRequest) (*ExecuteActivityResponse, error) {

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("OperatorActivity started")

	ctx = metadata.NewIncomingContext(ctx, metadata.MD{constant.HeaderAuthTypeKey: []string{"user"}, constant.HeaderUserUIDKey: []string{param.UserUID.String()}})

	compInputs, err := w.GetBlob(ctx, param.InputBlobRedisKeys)
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

	outputBlobRedisKeys, err := w.SetBlob(ctx, compOutputs)
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
