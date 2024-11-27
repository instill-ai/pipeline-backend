package googlesheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) insertRowsHelper(ctx context.Context, sharedLink string, sheetName string, rows []map[string]format.Value) ([]Row, error) {

	spreadsheetID, err := e.extractSpreadsheetID(sharedLink)
	if err != nil {
		return nil, err
	}

	sheetID, err := e.convertSheetNameToSheetID(ctx, spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}

	// Get the last row index by querying the sheet data
	resp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		sheetName,
	).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	// Get the header row
	if len(resp.Values) == 0 {
		return nil, nil // Empty sheet, no headers
	}
	headers := resp.Values[0] // First row contains headers

	// Create values array for each row
	var values [][]any
	for _, row := range rows {
		rowValues := make([]any, len(headers))
		for colIdx, header := range headers {
			headerStr, ok := header.(string)
			if !ok {
				continue
			}
			if val, exists := row[headerStr]; exists {
				rowValues[colIdx] = val.String()
			}
		}
		values = append(values, rowValues)
	}

	// Create insert dimension request
	request := &sheets.Request{
		AppendCells: &sheets.AppendCellsRequest{
			SheetId: sheetID,
			Rows:    []*sheets.RowData{},
			Fields:  "*",
		},
	}

	// Add each row to the request
	for _, rowValues := range values {
		cells := make([]*sheets.CellData, len(headers))
		for colIdx, value := range rowValues {
			if value == nil {
				continue
			}
			valueStr := value.(string)
			cells[colIdx] = &sheets.CellData{
				UserEnteredValue: &sheets.ExtendedValue{
					StringValue: &valueStr,
				},
			}
		}
		request.AppendCells.Rows = append(request.AppendCells.Rows, &sheets.RowData{
			Values: cells,
		})
	}

	// Execute batch update
	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{request},
	}

	_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	// Get the last row number before insertion
	lastRowResp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		fmt.Sprintf("%s!A1:A", sheetName),
	).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	startRow := len(lastRowResp.Values) - len(values) + 1
	rowNumbers := make([]int, len(values))
	for i := range values {
		rowNumbers[i] = startRow + i
	}

	// Convert the inserted values back to map format
	insertedRows := make([]Row, len(values))
	for i, rowValues := range values {
		rowMap := make(map[string]format.Value)
		for j, val := range rowValues {
			if j >= len(headers) {
				continue
			}
			headerStr, ok := headers[j].(string)
			if !ok {
				continue
			}
			switch val := val.(type) {
			case string:
				rowMap[headerStr] = data.NewString(val)
			case float64:
				rowMap[headerStr] = data.NewNumberFromFloat(val)
			case bool:
				rowMap[headerStr] = data.NewBoolean(val)
			}
		}
		insertedRows[i] = Row{
			RowValue:  rowMap,
			RowNumber: rowNumbers[i],
		}
	}

	return insertedRows, nil
}

func (e *execution) insertMultipleRows(ctx context.Context, job *base.Job) error {
	input := &taskInsertMultipleRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	insertedRows, err := e.insertRowsHelper(ctx, input.SharedLink, input.SheetName, input.RowValues)
	if err != nil {
		return err
	}

	output := &taskInsertMultipleRowsOutput{
		Rows: insertedRows,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
