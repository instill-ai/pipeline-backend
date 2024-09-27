package base

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/types/known/structpb"
)

var _ IExecution = &ExecutionWrapper{}

// ExecutionWrapper performs validation and usage collection around the
// execution of a component.
type ExecutionWrapper struct {
	IExecution
}

type inputReader struct {
	schema string
	input  *structpb.Struct
}

func NewInputReader(input *structpb.Struct, schema string) *inputReader {
	return &inputReader{
		input:  input,
		schema: schema,
	}
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

	// Note: We need to check usage of all inputs simultaneously, so all inputs
	// must be read before execution.
	for batchIdx, job := range jobs {
		inputs[batchIdx], err = job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
	}

	if err = h.Check(ctx, inputs); err != nil {
		return err
	}

	wrappedJobs := make([]*Job, len(jobs))
	for batchIdx, job := range jobs {
		wrappedJobs[batchIdx] = &Job{
			Input:  NewInputReader(inputs[batchIdx], e.GetTaskInputSchema()),
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
		outputs[batchIdx] = job.Output.(*outputWriter).GetOutput()
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

func recoverJobError(ctx context.Context, job *Job) {
	if r := recover(); r != nil {
		fmt.Printf("panic: %+v", r)
		job.Error.Error(ctx, fmt.Errorf("panic: %+v", r))
		return
	}
}
