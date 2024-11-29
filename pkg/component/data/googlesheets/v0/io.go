package googlesheets

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// taskCreateSpreadsheetInput represents input for creating a new spreadsheet
type taskCreateSpreadsheetInput struct {
	Title  string  `instill:"title"`
	Sheets []sheet `instill:"sheets"`
}

type sheet struct {
	Name    string   `instill:"name"`
	Headers []string `instill:"headers"`
}

// taskCreateSpreadsheetOutput represents output after creating a spreadsheet
type taskCreateSpreadsheetOutput struct {
	SharedLink string `instill:"shared-link"`
}

// taskDeleteSpreadsheetInput represents input for deleting a spreadsheet
type taskDeleteSpreadsheetInput struct {
	SharedLink string `instill:"shared-link"`
}

// taskDeleteSpreadsheetOutput represents output after deleting a spreadsheet
type taskDeleteSpreadsheetOutput struct {
	Success bool `instill:"success"`
}

// taskAddSheetInput represents input for adding a new sheet
type taskAddSheetInput struct {
	SharedLink string   `instill:"shared-link"`
	SheetName  string   `instill:"sheet-name"`
	Headers    []string `instill:"headers"`
}

// taskAddSheetOutput represents output after adding a sheet
type taskAddSheetOutput struct {
	Success bool `instill:"success"`
}

// taskDeleteSheetInput represents input for deleting a sheet
type taskDeleteSheetInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
}

// taskDeleteSheetOutput represents output after deleting a sheet
type taskDeleteSheetOutput struct {
	Success bool `instill:"success"`
}

// taskCreateSpreadsheetColumnInput represents input for creating a column
type taskCreateSpreadsheetColumnInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	ColumnName string `instill:"column-name"`
}

// taskCreateSpreadsheetColumnOutput represents output after creating a column
type taskCreateSpreadsheetColumnOutput struct {
	Success bool `instill:"success"`
}

// taskDeleteSpreadsheetColumnInput represents input for deleting a column
type taskDeleteSpreadsheetColumnInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	ColumnName string `instill:"column-name"`
}

// taskDeleteSpreadsheetColumnOutput represents output after deleting a column
type taskDeleteSpreadsheetColumnOutput struct {
	Success bool `instill:"success"`
}

// taskListRowsInput represents input for listing rows
type taskListRowsInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	StartRow   *int   `instill:"start-row,default=2"`
	EndRow     *int   `instill:"end-row"`
}

// taskListRowsOutput represents output after listing rows
type taskListRowsOutput struct {
	Rows []Row `instill:"rows"`
}

// taskLookupRowsInput represents input for looking up multiple rows
type taskLookupRowsInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	ColumnName string `instill:"column-name"`
	Value      string `instill:"value"`
}

// taskLookupRowsOutput represents output after looking up multiple rows
type taskLookupRowsOutput struct {
	Rows []Row `instill:"rows"`
}

// taskGetRowInput represents input for getting a row
type taskGetRowInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	RowNumber  int    `instill:"row-number"`
}

// taskGetRowOutput represents output after getting a row
type taskGetRowOutput struct {
	Row Row `instill:"row"`
}

// taskGetMultipleRowsInput represents input for getting multiple rows
type taskGetMultipleRowsInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	RowNumbers []int  `instill:"row-numbers"`
}

// taskGetMultipleRowsOutput represents output after getting multiple rows
type taskGetMultipleRowsOutput struct {
	Rows []Row `instill:"rows"`
}

// Row represents a row with row number and data
type Row struct {
	RowNumber int                     `instill:"row-number"`
	RowValue  map[string]format.Value `instill:"row-value"`
}

// taskInsertRowInput represents input for inserting a row
type taskInsertRowInput struct {
	SharedLink string                  `instill:"shared-link"`
	SheetName  string                  `instill:"sheet-name"`
	RowValue   map[string]format.Value `instill:"row-value"`
}

// taskInsertRowOutput represents output after inserting a row
type taskInsertRowOutput struct {
	Row Row `instill:"row"`
}

// taskInsertMultipleRowsInput represents input for inserting multiple rows
type taskInsertMultipleRowsInput struct {
	SharedLink string                    `instill:"shared-link"`
	SheetName  string                    `instill:"sheet-name"`
	RowValues  []map[string]format.Value `instill:"row-values"`
}

// taskInsertMultipleRowsOutput represents output after inserting multiple rows
type taskInsertMultipleRowsOutput struct {
	Rows []Row `instill:"rows"`
}

// taskUpdateRowInput represents input for updating a row
type taskUpdateRowInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	Row        Row    `instill:"row"`
}

// taskUpdateRowOutput represents output after updating a row
type taskUpdateRowOutput struct {
	Row Row `instill:"row"`
}

// taskUpdateMultipleRowsInput represents input for updating multiple rows
type taskUpdateMultipleRowsInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	Rows       []Row  `instill:"rows"`
}

// taskUpdateMultipleRowsOutput represents output after updating multiple rows
type taskUpdateMultipleRowsOutput struct {
	Rows []Row `instill:"rows"`
}

// taskDeleteRowInput represents input for deleting a row
type taskDeleteRowInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	RowNumber  int    `instill:"row-number"`
}

// taskDeleteRowOutput represents output after deleting a row
type taskDeleteRowOutput struct {
	Success bool `instill:"success"`
}

// taskDeleteMultipleRowsInput represents input for deleting multiple rows
type taskDeleteMultipleRowsInput struct {
	SharedLink string `instill:"shared-link"`
	SheetName  string `instill:"sheet-name"`
	RowNumbers []int  `instill:"row-numbers"`
}

// taskDeleteMultipleRowsOutput represents output after deleting multiple rows
type taskDeleteMultipleRowsOutput struct {
	Success bool `instill:"success"`
}
