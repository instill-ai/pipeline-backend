package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) deleteSpreadsheet(ctx context.Context, job *base.Job) error {
	input := &taskDeleteSpreadsheetInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	spreadsheetID, err := e.extractSpreadsheetID(input.SharedLink)
	if err != nil {
		return err
	}

	// Delete the spreadsheet using Drive API
	err = e.driveService.Files.Delete(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return err
	}

	output := &taskDeleteSpreadsheetOutput{
		Success: true,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
