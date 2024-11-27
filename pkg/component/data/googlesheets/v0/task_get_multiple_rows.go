package googlesheets

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) getRowsHelper(ctx context.Context, sharedLink string, sheetName string, rowNumbers []int) ([]map[string]format.Value, error) {
	spreadsheetID, err := e.extractSpreadsheetID(sharedLink)
	if err != nil {
		return nil, err
	}

	// Get headers from first row
	headerResp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		sheetName+"!1:1",
	).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	if len(headerResp.Values) == 0 {
		return nil, nil // Empty sheet
	}

	headers := headerResp.Values[0]

	result := make([]map[string]format.Value, len(rowNumbers))
	for i, rowNum := range rowNumbers {
		if rowNum <= 0 {
			continue
		}

		// Convert row number to A1 notation (e.g. "A5:Z5" for row 5)
		rowRange := fmt.Sprintf("%s!%d:%d", sheetName, rowNum, rowNum)

		rowResp, err := e.sheetService.Spreadsheets.Values.Get(
			spreadsheetID,
			rowRange,
		).Context(ctx).Do()
		if err != nil {
			continue // Skip invalid rows
		}

		if len(rowResp.Values) == 0 {
			continue // Skip empty rows
		}

		row := rowResp.Values[0]
		rowMap := make(map[string]format.Value)

		for j, header := range headers {
			headerStr := header.(string)
			if j < len(row) && row[j] != nil {
				switch r := row[j].(type) {
				case string:
					rowMap[headerStr] = data.NewString(r)
				case float64:
					rowMap[headerStr] = data.NewNumberFromFloat(r)
				case bool:
					rowMap[headerStr] = data.NewBoolean(r)
				}
			}
		}
		result[i] = rowMap
	}

	return result, nil
}

func (e *execution) getMultipleRows(ctx context.Context, job *base.Job) error {
	input := &taskGetMultipleRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	rows, err := e.getRowsHelper(ctx, input.SharedLink, input.SheetName, input.RowNumbers)
	if err != nil {
		return err
	}

	output := &taskGetMultipleRowsOutput{
		Rows: rows,
	}

	return job.Output.WriteData(ctx, output)
}
