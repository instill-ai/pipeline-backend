package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) insertRow(ctx context.Context, job *base.Job) error {
	input := &taskInsertRowInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	insertedRows, err := e.insertRowsHelper(ctx, input.SharedLink, input.SheetName, []map[string]format.Value{input.RowValue})
	if err != nil {
		return err
	}

	output := &taskInsertRowOutput{
		Row: insertedRows[0],
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
