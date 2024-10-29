package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) assign(ctx context.Context, job *base.Job) error {
	in := &assignInput{}
	if err := job.Input.ReadData(ctx, in); err != nil {
		return err
	}

	out := &assignOutput{Data: in.Data}
	return job.Output.WriteData(ctx, out)
}
