package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) append(ctx context.Context, job *base.Job) error {

	in := &appendInput{}
	err := job.Input.ReadData(ctx, in)
	if err != nil {
		return err
	}

	arr := in.Array
	element := in.Element
	arr = append(arr, element)

	out := &appendOutput{Array: arr}
	return job.Output.WriteData(ctx, out)
}
