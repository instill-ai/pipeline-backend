package worker

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"google.golang.org/protobuf/types/known/structpb"
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
		setupVal, err := recipe.Render(ctx, setupTemplate, i.conditionMap[idx], i.wfm)
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
	compID       string
	wfm          memory.WorkflowMemory
	conditionMap map[int]int
}

func NewInputReader(wfm memory.WorkflowMemory, compID string, conditionMap map[int]int) *inputReader {
	return &inputReader{
		compID:       compID,
		wfm:          wfm,
		conditionMap: conditionMap,
	}
}

func (i *inputReader) Read(ctx context.Context) (inputs []*structpb.Struct, err error) {

	for idx := range len(i.conditionMap) {
		inputTemplate, err := i.wfm.GetComponentData(ctx, i.conditionMap[idx], i.compID, memory.ComponentDataInput)
		if err != nil {
			return nil, err
		}

		inputVal, err := recipe.Render(ctx, inputTemplate, i.conditionMap[idx], i.wfm)
		if err != nil {
			return nil, err
		}

		if err = i.wfm.SetComponentData(ctx, i.conditionMap[idx], i.compID, memory.ComponentDataInput, inputVal); err != nil {
			return nil, err
		}

		input, err := inputVal.ToStructValue()
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input.GetStructValue())

	}
	return inputs, nil
}

type outputWriter struct {
	compID       string
	wfm          memory.WorkflowMemory
	conditionMap map[int]int
}

func NewOutputWriter(wfm memory.WorkflowMemory, compID string, conditionMap map[int]int) *outputWriter {
	return &outputWriter{
		compID:       compID,
		wfm:          wfm,
		conditionMap: conditionMap,
	}
}

func (o *outputWriter) Write(ctx context.Context, outputs []*structpb.Struct) (err error) {

	for idx, output := range outputs {
		val, err := data.NewValueFromStruct(structpb.NewStructValue(output))
		if err != nil {
			return err
		}
		if err := o.wfm.SetComponentData(ctx, o.conditionMap[idx], o.compID, memory.ComponentDataOutput, val); err != nil {
			return err
		}
		outputTemplate, err := o.wfm.Get(ctx, idx, string(memory.PipelineOutputTemplate))
		if err != nil {
			return err
		}
		output, err := recipe.Render(ctx, outputTemplate, idx, o.wfm)
		if err != nil {
			return err
		}
		err = o.wfm.SetPipelineData(ctx, idx, memory.PipelineOutput, output)
		if err != nil {
			return err
		}
	}

	return nil
}
