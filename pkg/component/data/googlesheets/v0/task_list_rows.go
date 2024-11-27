package googlesheets

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func (e *execution) listRows(ctx context.Context, job *base.Job) error {
	input := &taskListRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	spreadsheetID, err := e.extractSpreadsheetID(input.SharedLink)
	if err != nil {
		return err
	}

	// Get headers from first row
	headerResp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		input.SheetName+"!1:1",
	).Context(ctx).Do()
	if err != nil {
		return err
	}

	if len(headerResp.Values) == 0 {
		return nil // Empty sheet
	}

	headers := headerResp.Values[0]

	// Get data rows
	dataRange := fmt.Sprintf("%s!%d:%d", input.SheetName, input.StartRow, input.EndRow)
	dataResp, err := e.sheetService.Spreadsheets.Values.Get(
		spreadsheetID,
		dataRange,
	).Context(ctx).Do()
	if err != nil {
		return err
	}

	rows := make([]map[string]format.Value, 0)
	for _, row := range dataResp.Values {
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
		rows = append(rows, rowMap)
	}

	output := &taskListRowsOutput{
		Rows: rows,
	}

	return job.Output.WriteData(ctx, output)
}
