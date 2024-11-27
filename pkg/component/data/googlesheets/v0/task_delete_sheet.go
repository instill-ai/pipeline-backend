package googlesheets

import (
	"context"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) deleteSheet(ctx context.Context, job *base.Job) error {
	input := &taskDeleteSheetInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	spreadsheetID, err := e.extractSpreadsheetID(input.SharedLink)
	if err != nil {
		return err
	}

	sheetID, err := e.convertSheetNameToSheetID(ctx, spreadsheetID, input.SheetName)
	if err != nil {
		return err
	}

	// Create delete sheet request
	request := &sheets.Request{
		DeleteSheet: &sheets.DeleteSheetRequest{
			SheetId: sheetID,
		},
	}

	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{request},
	}

	// Execute the batch update
	_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return err
	}

	output := &taskDeleteSheetOutput{}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
