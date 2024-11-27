package googlesheets

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

func (e *execution) extractSpreadsheetID(sharedLink string) (string, error) {
	re := regexp.MustCompile(`https://docs.google.com/spreadsheets/d/([a-zA-Z0-9-_]+)`)
	matches := re.FindStringSubmatch(sharedLink)
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid shared link")
	}

	return matches[1], nil
}

func (e *execution) convertStringsToInterface(strings []string) []any {
	interfaces := make([]any, len(strings))
	for i, s := range strings {
		interfaces[i] = s
	}
	return interfaces
}

func (e *execution) convertSheetNameToSheetID(ctx context.Context, spreadsheetID string, sheetName string) (int64, error) {
	spreadsheet, err := e.sheetService.Spreadsheets.Get(spreadsheetID).Context(ctx).Do()
	if err != nil {
		return 0, err
	}

	for _, sheet := range spreadsheet.Sheets {
		if strings.EqualFold(sheet.Properties.Title, sheetName) {
			return sheet.Properties.SheetId, nil
		}
	}

	return 0, fmt.Errorf("sheet not found")
}
