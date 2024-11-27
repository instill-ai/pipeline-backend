package googlesheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) updateRowsHelper(ctx context.Context, sharedLink string, sheetName string, rowNumbers []int, rows []map[string]format.Value) ([]map[string]format.Value, error) {
	spreadsheetID, err := e.extractSpreadsheetID(sharedLink)
	if err != nil {
		return nil, err
	}

	sheetID, err := e.convertSheetNameToSheetID(ctx, spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}

	// Get the sheet data
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

	var requests []*sheets.Request

	// Create update requests for each row
	for i, rowNumber := range rowNumbers {

		row := rows[i]

		for colIdx, header := range headers {
			headerStr, ok := header.(string)
			if !ok {
				continue
			}

			if val, exists := row[headerStr]; exists {
				// Only add cell data if key exists in input row
				valueStr := val.String()
				cell := &sheets.CellData{
					UserEnteredValue: &sheets.ExtendedValue{
						StringValue: &valueStr,
					},
				}

				request := &sheets.Request{
					UpdateCells: &sheets.UpdateCellsRequest{
						Range: &sheets.GridRange{
							SheetId:          sheetID,
							StartRowIndex:    int64(rowNumber - 1), // Convert to 0-based index
							EndRowIndex:      int64(rowNumber),
							StartColumnIndex: int64(colIdx),
							EndColumnIndex:   int64(colIdx + 1),
						},
						Rows: []*sheets.RowData{
							{
								Values: []*sheets.CellData{cell},
							},
						},
						Fields: "userEnteredValue",
					},
				}
				requests = append(requests, request)
			}
		}

	}

	// Execute batch update
	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	// Fetch the updated rows from Google Sheets
	updatedRows := make([]map[string]format.Value, len(rowNumbers))
	for i, rowNumber := range rowNumbers {
		// Get the specific row
		rowRange := fmt.Sprintf("%s!%d:%d", sheetName, rowNumber, rowNumber)
		rowResp, err := e.sheetService.Spreadsheets.Values.Get(spreadsheetID, rowRange).Context(ctx).Do()
		if err != nil {
			return nil, err
		}

		if len(rowResp.Values) == 0 {
			continue
		}

		// Convert row data to map
		rowMap := make(map[string]format.Value)
		rowValues := rowResp.Values[0]
		for j, header := range headers {
			headerStr, ok := header.(string)
			if !ok || j >= len(rowValues) {
				continue
			}

			if rowValues[j] != nil {
				switch v := rowValues[j].(type) {
				case string:
					rowMap[headerStr] = data.NewString(v)
				case float64:
					rowMap[headerStr] = data.NewNumberFromFloat(v)
				case bool:
					rowMap[headerStr] = data.NewBoolean(v)
				}
			}
		}
		updatedRows[i] = rowMap
	}

	return updatedRows, nil
}

func (e *execution) updateMultipleRows(ctx context.Context, job *base.Job) error {
	input := &taskUpdateMultipleRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	updatedRows, err := e.updateRowsHelper(ctx, input.SharedLink, input.SheetName, input.RowNumbers, input.Rows)
	if err != nil {
		return err
	}

	output := &taskUpdateMultipleRowsOutput{
		Rows: updatedRows,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}
