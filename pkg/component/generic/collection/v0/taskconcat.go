package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) concat(ctx context.Context, job *base.Job) error {
	in := &concatInput{}
	if err := job.Input.ReadData(ctx, in); err != nil {
		return err
	}

	arrays := in.Arrays
	concat := []format.Value{}

	for _, a := range arrays {
		concat = append(concat, a...)
	}

	out := &concatOutput{Array: concat}
	return job.Output.WriteData(ctx, out)
}
