package googlesheets

import (
	"context"
	"fmt"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) listRowsHelper(ctx context.Context, sharedLink string, sheetName string, startRow *int, endRow *int) ([]Row, error) {
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
	// Get spreadsheet metadata to find total rows
	spreadsheet, err := e.sheetService.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	var targetSheet *sheets.Sheet
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			targetSheet = sheet
			break
		}
	}

	if targetSheet == nil {
		return nil, fmt.Errorf("sheet %s not found", sheetName)
	}

	maxRows := targetSheet.Properties.GridProperties.RowCount

	// If endRow is not specified or exceeds maxRows, set it to maxRows
	if endRow == nil || *endRow > int(maxRows) {
		maxRowsInt := int(maxRows)
		endRow = &maxRowsInt
	}
	if startRow == nil {
		startRow = func() *int { startRow := 2; return &startRow }()
	}

	// Get data rows
	dataRange := fmt.Sprintf("%s!%d:%d", sheetName, *startRow, *endRow)
	dataResp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		dataRange,
	).Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0)
	for i, row := range dataResp.Values {
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
		rows = append(rows, Row{
			RowValue:  rowMap,
			RowNumber: i + *startRow,
		})
	}
	return rows, nil
}

func (e *execution) listRows(ctx context.Context, job *base.Job) error {
	input := &taskListRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	rows, err := e.listRowsHelper(ctx, input.SharedLink, input.SheetName, input.StartRow, input.EndRow)
	if err != nil {
		return err
	}

	output := &taskListRowsOutput{
		Rows: rows,
	}

	return job.Output.WriteData(ctx, output)
}
