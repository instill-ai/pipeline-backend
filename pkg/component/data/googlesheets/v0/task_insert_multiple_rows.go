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

	// Get the sheet data
	resp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		sheetName,
	).Context(ctx).ValueRenderOption("UNFORMATTED_VALUE").Do()
	if err != nil {
		return nil, err
	}

	// Get the header row
	if len(resp.Values) == 0 {
		return nil, nil // Empty sheet, no headers
	}
	headers := resp.Values[0] // First row contains headers

	// Calculate the starting row number for insertion
	totalExistingRows := len(resp.Values)
	startRowIndex := totalExistingRows // 0-based index for the first new row

	var requests []*sheets.Request

	// First, insert empty rows to make space
	if len(rows) > 0 {
		insertRequest := &sheets.Request{
			InsertDimension: &sheets.InsertDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: int64(startRowIndex),
					EndIndex:   int64(startRowIndex + len(rows)),
				},
			},
		}
		requests = append(requests, insertRequest)
	}

	// Create update requests for each row with cell data and formatting
	for rowIdx, row := range rows {
		currentRowIndex := startRowIndex + rowIdx

		for colIdx, header := range headers {
			headerStr, ok := header.(string)
			if !ok {
				continue
			}

			var cell *sheets.CellData
			if val, exists := row[headerStr]; exists {
				if val == nil {
					// Update with empty value if cell is nil
					emptyStr := ""
					cell = &sheets.CellData{
						UserEnteredValue: &sheets.ExtendedValue{
							StringValue: &emptyStr,
						},
					}
				} else {
					switch v := val.(type) {
					case format.Number:
						valueStr := v.Float64()
						cell = &sheets.CellData{
							UserEnteredValue: &sheets.ExtendedValue{
								NumberValue: &valueStr,
							},
						}
					case format.Boolean:
						valueStr := v.Boolean()
						cell = &sheets.CellData{
							UserEnteredValue: &sheets.ExtendedValue{
								BoolValue: &valueStr,
							},
						}
					default:
						valueStr := val.String()
						cell = &sheets.CellData{
							UserEnteredValue: &sheets.ExtendedValue{
								StringValue: &valueStr,
							},
						}
					}
				}

				// Add the cell update request
				updateRequest := &sheets.Request{
					UpdateCells: &sheets.UpdateCellsRequest{
						Range: &sheets.GridRange{
							SheetId:          sheetID,
							StartRowIndex:    int64(currentRowIndex),
							EndRowIndex:      int64(currentRowIndex + 1),
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
				requests = append(requests, updateRequest)
			}
		}

		// Copy formatting from the previous row if it exists
		if totalExistingRows > 1 { // More than just headers
			prevRowIndex := totalExistingRows - 1 // Last data row before insertion

			copyFormatRequest := &sheets.Request{
				CopyPaste: &sheets.CopyPasteRequest{
					Source: &sheets.GridRange{
						SheetId:          sheetID,
						StartRowIndex:    int64(prevRowIndex),
						EndRowIndex:      int64(prevRowIndex + 1),
						StartColumnIndex: 0,
						EndColumnIndex:   int64(len(headers)),
					},
					Destination: &sheets.GridRange{
						SheetId:          sheetID,
						StartRowIndex:    int64(currentRowIndex),
						EndRowIndex:      int64(currentRowIndex + 1),
						StartColumnIndex: 0,
						EndColumnIndex:   int64(len(headers)),
					},
					PasteType: "PASTE_FORMAT",
				},
			}
			requests = append(requests, copyFormatRequest)
		}
	}

	// Execute all requests in batch
	if len(requests) > 0 {
		batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: requests,
		}

		_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
		if err != nil {
			return nil, err
		}
	}

	// Fetch the inserted rows from Google Sheets
	insertedRows := make([]Row, len(rows))
	for i := range rows {
		rowNumber := startRowIndex + i + 1 // Convert to 1-based for display
		// Get the specific row
		rowRange := fmt.Sprintf("%s!%d:%d", sheetName, rowNumber, rowNumber)
		rowResp, err := e.sheetService.Spreadsheets.Values.Get(spreadsheetID, rowRange).Context(ctx).ValueRenderOption("UNFORMATTED_VALUE").Do()
		if err != nil {
			return nil, err
		}

		rowMap := make(map[string]format.Value)
		if len(rowResp.Values) > 0 {
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
					default:
						rowMap[headerStr] = data.NewString(fmt.Sprintf("%v", v))
					}
				}
			}
		}

		insertedRows[i] = Row{
			RowValue:  rowMap,
			RowNumber: rowNumber,
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
