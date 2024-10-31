package collection

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) split(ctx context.Context, job *base.Job) error {
	in := &splitInput{}
	err := job.Input.ReadData(ctx, in)
	if err != nil {
		return err
	}
	arr := in.Array
	size := in.GroupSize
	groups := make([][]format.Value, 0)

	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		groups = append(groups, arr[i:end])
	}

	return job.Output.WriteData(ctx, &splitOutput{Arrays: groups})
}
