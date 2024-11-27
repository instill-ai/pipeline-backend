package googlesheets

import (
	"context"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) createSpreadsheetColumn(ctx context.Context, job *base.Job) error {
	input := &taskCreateSpreadsheetColumnInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	spreadsheetID, err := e.extractSpreadsheetID(input.SharedLink)
	if err != nil {
		return err
	}

	// Get the current sheet data to find the last column
	resp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		input.SheetName+"!1:1",
	).Context(ctx).Do()
	if err != nil {
		return err
	}

	var columnIndex int
	if len(resp.Values) > 0 {
		columnIndex = len(resp.Values[0]) + 1
	} else {
		columnIndex = 1
	}

	// Convert column index to A1 notation
	columnLetter := ""
	for columnIndex > 0 {
		columnIndex--
		columnLetter = string(rune('A'+columnIndex%26)) + columnLetter
		columnIndex = columnIndex / 26
	}

	// Update the header with the new column name
	valueRange := &sheets.ValueRange{
		Values: [][]any{{input.ColumnName}},
	}

	_, err = e.sheetService.Spreadsheets.Values.Update(
		spreadsheetID,
		input.SheetName+"!"+columnLetter+"1",
		valueRange,
	).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		return err
	}

	// TODO(huitang): reflect the real status
	output := &taskCreateSpreadsheetColumnOutput{
		Success: true,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
