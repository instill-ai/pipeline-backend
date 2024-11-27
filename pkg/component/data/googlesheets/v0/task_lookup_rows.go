package googlesheets

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) lookupRowsHelper(ctx context.Context, sharedLink string, sheetName string, columnName string, value string) ([]int, []map[string]format.Value, error) {
	spreadsheetID, err := e.extractSpreadsheetID(sharedLink)
	if err != nil {
		return nil, nil, err
	}

	// Get all values from the sheet
	resp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		sheetName,
	).Context(ctx).Do()
	if err != nil {
		return nil, nil, err
	}

	if len(resp.Values) == 0 {
		return nil, nil, nil // Empty sheet
	}

	// Get headers from first row
	headers := resp.Values[0]

	// Find the target column index
	var columnIndex int = -1
	for i, header := range headers {
		if header.(string) == columnName {
			columnIndex = i
			break
		}
	}

	if columnIndex == -1 {
		return nil, nil, nil // Column not found
	}

	// Look for matching rows
	var rowNumbers []int
	var result []map[string]format.Value
	for i := 1; i < len(resp.Values); i++ {
		row := resp.Values[i]
		// Check if the column value matches exactly
		if len(row) > columnIndex && row[columnIndex] != nil && row[columnIndex].(string) == value {
			// Create map for matching row
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
			result = append(result, rowMap)
			rowNumbers = append(rowNumbers, i)
		}
	}

	return rowNumbers, result, nil
}

func (e *execution) lookupRows(ctx context.Context, job *base.Job) error {
	input := &taskLookupRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	rowNumbers, rows, err := e.lookupRowsHelper(ctx, input.SharedLink, input.SheetName, input.ColumnName, input.Value)
	if err != nil {
		return err
	}

	output := &taskLookupRowsOutput{
		Rows:       rows,
		RowNumbers: rowNumbers,
	}

	return job.Output.WriteData(ctx, output)
}
