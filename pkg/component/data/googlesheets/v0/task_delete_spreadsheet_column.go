package googlesheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) deleteSpreadsheetColumn(ctx context.Context, job *base.Job) error {
	input := &taskDeleteSpreadsheetColumnInput{}
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

	// Get the current sheet data to find the column index
	resp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		input.SheetName+"!1:1",
	).Context(ctx).Do()
	if err != nil {
		return err
	}

	// Find the column index
	var columnIndex = -1
	if len(resp.Values) > 0 {
		for i, header := range resp.Values[0] {
			if header.(string) == input.ColumnName {
				columnIndex = i
				break
			}
		}
	}

	if columnIndex == -1 {
		return fmt.Errorf("column not found")
	}

	// Create delete dimension request
	request := &sheets.Request{
		DeleteDimension: &sheets.DeleteDimensionRequest{
			Range: &sheets.DimensionRange{
				SheetId:    sheetID,
				Dimension:  "COLUMNS",
				StartIndex: int64(columnIndex),
				EndIndex:   int64(columnIndex + 1), // Delete 1 column
			},
		},
	}

	// Execute batch update
	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{request},
	}

	_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return err
	}

	output := &taskDeleteSpreadsheetColumnOutput{
		Success: true,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
