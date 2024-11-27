package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) getRow(ctx context.Context, job *base.Job) error {
	input := &taskGetRowInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	rows, err := e.getRowsHelper(ctx, input.SharedLink, input.SheetName, []int{input.RowNumber})
	if err != nil {
		return err
	}

	output := &taskGetRowOutput{
		Row: rows[0],
	}

	return job.Output.WriteData(ctx, output)
}
