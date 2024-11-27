package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) deleteRow(ctx context.Context, job *base.Job) error {
	input := &taskDeleteRowInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	err := e.deleteRowsHelper(ctx, input.SharedLink, input.SheetName, []int{input.RowNumber})
	if err != nil {
		return err
	}

	// TODO(huitang): reflect the real status
	output := &taskDeleteRowOutput{
		Success: true,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
