package googlesheets

import (
	"context"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) createSpreadsheet(ctx context.Context, job *base.Job) error {
	input := &taskCreateSpreadsheetInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	// Create a new spreadsheet
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: input.Title,
		},
	}

	// Create the spreadsheet
	createdSpreadsheet, err := e.sheetService.Spreadsheets.Create(spreadsheet).Context(ctx).Do()
	if err != nil {
		return err
	}

	// For each sheet in the input
	for _, sheet := range input.Sheets {
		if sheet.Name != "sheet1" {
			// Create the add sheet request
			addSheetRequest := &sheets.Request{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{
						Title: sheet.Name,
					},
				},
			}

			batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
				Requests: []*sheets.Request{addSheetRequest},
			}

			// Execute the batch update
			_, err = e.sheetService.Spreadsheets.BatchUpdate(createdSpreadsheet.SpreadsheetId, batchUpdateRequest).Context(ctx).Do()
			if err != nil {
				return err
			}
		}

		// If headers are provided, update the first row
		if len(sheet.Headers) > 0 {
			valueRange := &sheets.ValueRange{
				Values: [][]any{
					e.convertStringsToInterface(sheet.Headers),
				},
			}

			// Update the header row
			_, err = e.sheetService.Spreadsheets.Values.Update(
				createdSpreadsheet.SpreadsheetId,
				sheet.Name+"!A1",
				valueRange,
			).ValueInputOption("RAW").Context(ctx).Do()
			if err != nil {
				return err
			}
		}
	}

	output := &taskCreateSpreadsheetOutput{
		SharedLink: createdSpreadsheet.SpreadsheetUrl,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
