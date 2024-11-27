package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) updateRow(ctx context.Context, job *base.Job) error {
	input := &taskUpdateRowInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	updatedRows, err := e.updateRowsHelper(ctx, input.SharedLink, input.SheetName, []Row{input.Row})
	if err != nil {
		return err
	}

	output := &taskUpdateRowOutput{
		Row: updatedRows[0],
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
