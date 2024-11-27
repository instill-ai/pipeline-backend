package googlesheets

import (
	"context"

	"google.golang.org/api/sheets/v4"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) deleteRowsHelper(ctx context.Context, sharedLink string, sheetName string, rowNumbers []int) error {

	spreadsheetID, err := e.extractSpreadsheetID(sharedLink)
	if err != nil {
		return err
	}

	sheetID, err := e.convertSheetNameToSheetID(ctx, spreadsheetID, sheetName)
	if err != nil {
		return err
	}

	// Create delete dimension request for each row index
	var requests []*sheets.Request
	for _, rowNumber := range rowNumbers {
		requests = append(requests, &sheets.Request{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: int64(rowNumber - 1), // Convert to 0-based index
					EndIndex:   int64(rowNumber),     // Delete 1 row
				},
			},
		})
	}

	// Execute batch update
	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	_, err = e.sheetService.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

func (e *execution) deleteMultipleRows(ctx context.Context, job *base.Job) error {
	input := &taskDeleteMultipleRowsInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	err := e.deleteRowsHelper(ctx, input.SharedLink, input.SheetName, input.RowNumbers)
	if err != nil {
		return err
	}

	// TODO(huitang): reflect the real status
	output := &taskDeleteMultipleRowsOutput{
		Success: true,
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil

}
