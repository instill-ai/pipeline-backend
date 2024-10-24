package worker

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/x/errmsg"
)

type setupReader struct {
	compID       string
	wfm          memory.WorkflowMemory
	conditionMap map[int]int
}

func NewSetupReader(wfm memory.WorkflowMemory, compID string, conditionMap map[int]int) *setupReader {
	return &setupReader{
		compID:       compID,
		wfm:          wfm,
		conditionMap: conditionMap,
	}
}

func (i *setupReader) Read(ctx context.Context) (setups []*structpb.Struct, err error) {
	for idx := range len(i.conditionMap) {
		setupTemplate, err := i.wfm.GetComponentData(ctx, i.conditionMap[idx], i.compID, memory.ComponentDataSetup)
		if err != nil {
			return nil, err
		}
		setupVal, err := recipe.Render(ctx, setupTemplate, i.conditionMap[idx], i.wfm, false)
		if err != nil {
			return nil, err
		}
		setup, err := setupVal.ToStructValue()
		if err != nil {
			return nil, err
		}
		setups = append(setups, setup.GetStructValue())
	}

	return setups, nil
}

type inputReader struct {
	compID      string
	wfm         memory.WorkflowMemory
	originalIdx int
}

func NewInputReader(wfm memory.WorkflowMemory, compID string, originalIdx int) *inputReader {
	return &inputReader{
		compID:      compID,
		wfm:         wfm,
		originalIdx: originalIdx,
	}
}

func (i *inputReader) read(ctx context.Context) (inputVal format.Value, err error) {

	inputTemplate, err := i.wfm.GetComponentData(ctx, i.originalIdx, i.compID, memory.ComponentDataInput)
	if err != nil {
		return nil, err
	}

	inputVal, err = recipe.Render(ctx, inputTemplate, i.originalIdx, i.wfm, false)
	if err != nil {
		return nil, err
	}

	if err = i.wfm.SetComponentData(ctx, i.originalIdx, i.compID, memory.ComponentDataInput, inputVal); err != nil {
		return nil, err
	}
	return inputVal, nil
}

func (i *inputReader) Read(ctx context.Context) (inputStruct *structpb.Struct, err error) {
	inputVal, err := i.read(ctx)
	if err != nil {
		return nil, err
	}

	input, err := inputVal.ToStructValue()
	if err != nil {
		return nil, err
	}

	return input.GetStructValue(), nil
}

func (i *inputReader) ReadData(ctx context.Context, input any) (err error) {
	inputVal, err := i.read(ctx)
	if err != nil {
		return err
	}
	return data.Unmarshal(inputVal, input)
}

type outputWriter struct {
	compID      string
	wfm         memory.WorkflowMemory
	originalIdx int
	streaming   bool
}

func NewOutputWriter(wfm memory.WorkflowMemory, compID string, originalIdx int, streaming bool) *outputWriter {
	return &outputWriter{
		compID:      compID,
		wfm:         wfm,
		originalIdx: originalIdx,
		streaming:   streaming,
	}
}

func (o *outputWriter) WriteData(ctx context.Context, output any) (err error) {

	val, err := data.Marshal(output)
	if err != nil {
		return err
	}

	return o.write(ctx, val)
}

func (o *outputWriter) Write(ctx context.Context, output *structpb.Struct) (err error) {

	val, err := data.NewValueFromStruct(structpb.NewStructValue(output))
	if err != nil {
		return err
	}
	return o.write(ctx, val)
}

func (o *outputWriter) write(ctx context.Context, val format.Value) (err error) {

	if err := o.wfm.SetComponentData(ctx, o.originalIdx, o.compID, memory.ComponentDataOutput, val); err != nil {
		return err
	}

	if o.streaming {
		outputTemplate, err := o.wfm.Get(ctx, o.originalIdx, string(memory.PipelineOutputTemplate))
		if err != nil {
			return err
		}

		output, err := recipe.Render(ctx, outputTemplate, o.originalIdx, o.wfm, true)
		if err != nil {
			return err
		}
		err = o.wfm.SetPipelineData(ctx, o.originalIdx, memory.PipelineOutput, output)
		if err != nil {
			return err
		}
	}

	return nil
}

type errorHandler struct {
	compID      string
	wfm         memory.WorkflowMemory
	originalIdx int
}

func NewErrorHandler(wfm memory.WorkflowMemory, compID string, originalIdx int) *errorHandler {
	return &errorHandler{
		compID:      compID,
		wfm:         wfm,
		originalIdx: originalIdx,
	}
}

func (e *errorHandler) Error(ctx context.Context, err error) {
	_ = e.wfm.SetComponentStatus(ctx, e.originalIdx, e.compID, memory.ComponentStatusErrored, true)
	_ = e.wfm.SetComponentErrorMessage(ctx, e.originalIdx, e.compID, errmsg.MessageOrErr(err))
}
