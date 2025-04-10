package worker

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/x/errmsg"
)

type setupReader struct {
	memoryStore       *memory.Store
	workflowID        string
	compID            string
	processedBatchIDs []int
}

func (s *setupReader) Read(ctx context.Context) (setups []*structpb.Struct, err error) {
	wfm, err := s.memoryStore.GetWorkflowMemory(ctx, s.workflowID)
	if err != nil {
		return nil, err
	}

	for _, batchIdx := range s.processedBatchIDs {
		setupTemplate, err := wfm.GetComponentData(ctx, batchIdx, s.compID, memory.ComponentDataSetupTemplate)
		if err != nil {
			return nil, err
		}
		setupVal, err := recipe.Render(ctx, setupTemplate, batchIdx, wfm, false)
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
	memoryStore   *memory.Store
	workflowID    string
	compID        string
	originalIdx   int
	binaryFetcher external.BinaryFetcher
}

func newInputReader(memoryStore *memory.Store, workflowID string, compID string, originalIdx int, binaryFetcher external.BinaryFetcher) *inputReader {
	return &inputReader{
		memoryStore:   memoryStore,
		workflowID:    workflowID,
		compID:        compID,
		originalIdx:   originalIdx,
		binaryFetcher: binaryFetcher,
	}
}

// Deprecated: read() is deprecated and will be removed in a future version. Use
// ReadData() instead.
// structpb is not suitable for handling binary data and will be phased out gradually.
func (i *inputReader) read(ctx context.Context) (inputVal format.Value, err error) {
	wfm, err := i.memoryStore.GetWorkflowMemory(ctx, i.workflowID)
	if err != nil {
		return nil, err
	}

	inputTemplate, err := wfm.GetComponentData(ctx, i.originalIdx, i.compID, memory.ComponentDataInputTemplate)
	if err != nil {
		return nil, err
	}

	inputVal, err = recipe.Render(ctx, inputTemplate, i.originalIdx, wfm, false)
	if err != nil {
		return nil, err
	}

	if err = wfm.SetComponentData(ctx, i.originalIdx, i.compID, memory.ComponentDataInput, inputVal); err != nil {
		return nil, err
	}
	return inputVal, nil
}

// Deprecated: Read() is deprecated and will be removed in a future version. Use
// ReadData() instead. structpb is not suitable for handling binary data and
// will be phased out gradually.
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

	unmarshaler := data.NewUnmarshaler(i.binaryFetcher)
	if err := unmarshaler.Unmarshal(ctx, inputVal, input); err != nil {
		return err
	}

	return nil
}

type outputWriter struct {
	memoryStore *memory.Store
	workflowID  string
	compID      string
	originalIdx int
	streaming   bool
}

func newOutputWriter(memoryStore *memory.Store, workflowID string, compID string, originalIdx int, streaming bool) *outputWriter {
	return &outputWriter{
		memoryStore: memoryStore,
		workflowID:  workflowID,
		compID:      compID,
		originalIdx: originalIdx,
		streaming:   streaming,
	}
}

func (o *outputWriter) WriteData(ctx context.Context, output any) (err error) {
	marshaler := data.NewMarshaler()
	val, err := marshaler.Marshal(output)
	if err != nil {
		return err
	}

	return o.write(ctx, val)
}

// Deprecated: Write() is deprecated and will be removed in a future version.
// Use WriteData() instead. structpb is not suitable for handling binary data
// and will be phased out gradually.
func (o *outputWriter) Write(ctx context.Context, output *structpb.Struct) (err error) {

	val, err := data.NewValueFromStruct(structpb.NewStructValue(output))
	if err != nil {
		return err
	}
	return o.write(ctx, val)
}

// Deprecated: write() is deprecated and will be removed in a future version.
// Use WriteData() instead. structpb is not suitable for handling binary data
// and will be phased out gradually.
func (o *outputWriter) write(ctx context.Context, val format.Value) (err error) {
	wfm, err := o.memoryStore.GetWorkflowMemory(ctx, o.workflowID)
	if err != nil {
		return fmt.Errorf("getting workflow memory: %w", err)
	}

	if err := wfm.SetComponentData(ctx, o.originalIdx, o.compID, memory.ComponentDataOutput, val); err != nil {
		return fmt.Errorf("setting component output data: %w", err)
	}

	if o.streaming {
		outputTemplate, err := wfm.Get(ctx, o.originalIdx, string(memory.PipelineOutputTemplate))
		if err != nil {
			return fmt.Errorf("getting output template: %w", err)
		}

		output, err := recipe.Render(ctx, outputTemplate, o.originalIdx, wfm, true)
		if err != nil {
			return fmt.Errorf("rendering output template: %w", err)
		}
		err = wfm.SetPipelineData(ctx, o.originalIdx, memory.PipelineOutput, output)
		if err != nil {
			return fmt.Errorf("setting pipeline output data: %w", err)
		}
	}

	return nil
}

type errorHandler struct {
	memoryStore *memory.Store
	workflowID  string
	compID      string
	originalIdx int

	parentWorkflowID  *string
	parentCompID      *string
	parentOriginalIdx *int
}

func newErrorHandler(memoryStore *memory.Store, workflowID string, compID string, originalIdx int, parentWorkflowID *string, parentCompID *string, parentOriginalIdx *int) *errorHandler {
	return &errorHandler{
		memoryStore:       memoryStore,
		workflowID:        workflowID,
		compID:            compID,
		originalIdx:       originalIdx,
		parentWorkflowID:  parentWorkflowID,
		parentCompID:      parentCompID,
		parentOriginalIdx: parentOriginalIdx,
	}
}

func (e *errorHandler) Error(ctx context.Context, err error) {

	wfm, wfmErr := e.memoryStore.GetWorkflowMemory(ctx, e.workflowID)
	if wfmErr != nil {
		return
	}

	_ = wfm.SetComponentStatus(ctx, e.originalIdx, e.compID, memory.ComponentStatusErrored, true)
	_ = wfm.SetComponentErrorMessage(ctx, e.originalIdx, e.compID, errmsg.MessageOrErr(err))

	if e.parentWorkflowID != nil {
		iterWfm, iterWfmErr := e.memoryStore.GetWorkflowMemory(ctx, *e.parentWorkflowID)
		if iterWfmErr != nil {
			return
		}
		_ = iterWfm.SetComponentStatus(ctx, *e.parentOriginalIdx, *e.parentCompID, memory.ComponentStatusErrored, true)
		_ = iterWfm.SetComponentErrorMessage(ctx, *e.parentOriginalIdx, *e.parentCompID, errmsg.MessageOrErr(err))
	}
}
