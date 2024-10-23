package base

import (
	"context"
	"fmt"
	"sync"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ IExecution = &ExecutionWrapper{}

// ExecutionWrapper performs validation and usage collection around the
// execution of a component.
type ExecutionWrapper struct {
	IExecution
}

type inputReader struct {
	InputReader
	schema string
	input  *structpb.Struct
}

func NewInputReader(ir InputReader, input *structpb.Struct, schema string) *inputReader {
	return &inputReader{
		InputReader: ir,
		input:       input,
		schema:      schema,
	}
}

func (ir *inputReader) ReadData(ctx context.Context, input any) (err error) {
	return ir.InputReader.ReadData(ctx, input)
}

func (ir *inputReader) Read(ctx context.Context) (input *structpb.Struct, err error) {
	input = ir.input
	if err = Validate(input, ir.schema, "input"); err != nil {
		return nil, err
	}
	return input, nil
}

type outputWriter struct {
	OutputWriter
	schema string
	output *structpb.Struct
}

func NewOutputWriter(ow OutputWriter, schema string) *outputWriter {
	return &outputWriter{
		OutputWriter: ow,
		schema:       schema,
	}
}
func (ow *outputWriter) WriteData(ctx context.Context, output any) (err error) {
	outputMap, err := data.Marshal(output)
	if err != nil {
		return err
	}
	outputStructPB, err := outputMap.ToStructValue()
	if err != nil {
		return err
	}
	ow.output = outputStructPB.GetStructValue()

	return ow.OutputWriter.WriteData(ctx, output)
}
func (ow *outputWriter) Write(ctx context.Context, output *structpb.Struct) (err error) {

	if err := Validate(output, ow.schema, "output"); err != nil {
		return err
	}
	ow.output = output

	return ow.OutputWriter.Write(ctx, output)

}

func (ow *outputWriter) GetOutput() *structpb.Struct {
	return ow.output
}

// Execute wraps the execution method with validation and usage collection.
func (e *ExecutionWrapper) Execute(ctx context.Context, jobs []*Job) (err error) {

	newUH := e.GetComponent().UsageHandlerCreator()
	h, err := newUH(e)
	if err != nil {
		return err
	}

	inputs := make([]*structpb.Struct, len(jobs))
	outputs := make([]*structpb.Struct, len(jobs))

	validInputs := make([]*structpb.Struct, 0, len(jobs))
	validJobs := make([]*Job, 0, len(jobs))
	validJobIdx := make([]int, 0, len(jobs))

	// Note: We need to check usage of all inputs simultaneously, so all inputs
	// must be read before execution.
	for batchIdx, job := range jobs {
		inputs[batchIdx], err = job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		validInputs = append(validInputs, inputs[batchIdx])
		validJobs = append(validJobs, job)
		validJobIdx = append(validJobIdx, batchIdx)
	}

	if err = h.Check(ctx, inputs); err != nil {
		return err
	}

	wrappedJobs := make([]*Job, len(validJobs))
	for batchIdx, job := range validJobs {
		wrappedJobs[batchIdx] = &Job{
			Input:  NewInputReader(job.Input, validInputs[batchIdx], e.GetTaskInputSchema()),
			Output: NewOutputWriter(job.Output, e.GetTaskOutputSchema()),
			Error:  job.Error,
		}
	}

	if err := e.IExecution.Execute(ctx, wrappedJobs); err != nil {
		return err
	}

	// Since there might be multiple writes, we collect the usage at the end of
	// the execution.​
	for batchIdx, job := range wrappedJobs {
		outputs[validJobIdx[batchIdx]] = job.Output.(*outputWriter).GetOutput()
	}
	if err := h.Collect(ctx, inputs, outputs); err != nil {
		return err
	}

	return nil

}

func SequentialExecutor(ctx context.Context, jobs []*Job, execute func(*structpb.Struct) (*structpb.Struct, error)) error {
	// The execution takes an array of inputs and returns an array of outputs,
	// processed sequentially.
	// Note: The `SequentialExecutor` does not support component streaming.
	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		output, err := execute(input)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
	}
	return nil
}

func ConcurrentExecutor(ctx context.Context, jobs []*Job, execute func(*structpb.Struct, *Job, context.Context) (*structpb.Struct, error)) error {
	var wg sync.WaitGroup
	wg.Add(len(jobs))
	for _, job := range jobs {
		go func() {
			defer wg.Done()
			defer recoverJobError(ctx, job)
			input, err := job.Input.Read(ctx)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
			output, err := execute(input, job, ctx)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
			err = job.Output.Write(ctx, output)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}
		}()
	}
	wg.Wait()
	return nil
}

func ConcurrentDataExecutor(ctx context.Context, jobs []*Job, execute func(context.Context, *Job) error) error {
	var wg sync.WaitGroup
	wg.Add(len(jobs))
	for _, job := range jobs {
		go func() {
			defer wg.Done()
			defer recoverJobError(ctx, job)
			err := execute(ctx, job)
			if err != nil {
				job.Error.Error(ctx, err)
				return
			}

		}()
	}
	wg.Wait()
	return nil
}

func recoverJobError(ctx context.Context, job *Job) {
	if r := recover(); r != nil {
		fmt.Printf("panic: %+v", r)
		job.Error.Error(ctx, fmt.Errorf("panic: %+v", r))
		return
	}
}
