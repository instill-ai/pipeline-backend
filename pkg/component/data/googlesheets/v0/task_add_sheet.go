package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"google.golang.org/api/sheets/v4"
)

func (e *execution) addSheet(ctx context.Context, job *base.Job) error {
	input := &taskAddSheetInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	spreadsheetID, err := e.extractSpreadsheetID(input.SharedLink)
	if err != nil {
		return err
	}

	// Create the add sheet request
	addSheetRequest := &sheets.Request{
		AddSheet: &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: input.SheetName,
			},
		},
	}

	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{addSheetRequest},
	}

	// Execute the batch update
	_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return err
	}

	// If headers are provided, update the first row
	if len(input.Headers) > 0 {
		valueRange := &sheets.ValueRange{
			Values: [][]any{
				e.convertStringsToInterface(input.Headers),
			},
		}

		// Update the header row
		_, err = e.sheetService.Spreadsheets.Values.Update(
			spreadsheetID,
			input.SheetName+"!A1",
			valueRange,
		).ValueInputOption("RAW").Context(ctx).Do()
		if err != nil {
			return err
		}
	}

	// TODO(huitang): reflect the real status
	output := &taskAddSheetOutput{
		Success: true,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
