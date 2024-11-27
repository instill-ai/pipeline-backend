package googlesheets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestCreateSpreadsheetColumn(t *testing.T) {
	c := qt.New(t)

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet1!1:1" && r.Method == "GET" {
			// Return mock get values response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"values": [["Header1", "Header2"]]
			}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet1!C1" && r.Method == "PUT" {
			// Return mock update values response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/empty-sheet/values/sheet1!1:1" && r.Method == "GET" {
			// Return mock empty sheet response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/empty-sheet/values/sheet1!A1" && r.Method == "PUT" {
			// Return mock update values response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	testCases := []struct {
		name           string
		input          taskCreateSpreadsheetColumnInput
		expectedOutput taskCreateSpreadsheetColumnOutput
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "ok - create column in existing sheet",
			input: taskCreateSpreadsheetColumnInput{
				SharedLink: "https://docs.google.com/spreadsheets/d/test-id",
				SheetName:  "sheet1",
				ColumnName: "Header3",
			},
			expectedOutput: taskCreateSpreadsheetColumnOutput{
				Success: true,
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "ok - create first column in empty sheet",
			input: taskCreateSpreadsheetColumnInput{
				SharedLink: "https://docs.google.com/spreadsheets/d/empty-sheet",
				SheetName:  "sheet1",
				ColumnName: "Header1",
			},
			expectedOutput: taskCreateSpreadsheetColumnOutput{
				Success: true,
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "error - invalid spreadsheet ID",
			input: taskCreateSpreadsheetColumnInput{
				SharedLink: "invalid-link",
				SheetName:  "sheet1",
				ColumnName: "Header1",
			},
			expectedOutput: taskCreateSpreadsheetColumnOutput{},
			expectErr:      true,
			expectedErrMsg: "invalid shared link",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			// Create sheets service with test server
			sheetsService, err := sheets.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(ts.URL))
			c.Assert(err, qt.IsNil)

			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			exe, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskCreateSpreadsheetColumn,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(exe, qt.IsNotNil)

			// Set sheets service
			exe.(*execution).sheetService = sheetsService

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *taskCreateSpreadsheetColumnInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput taskCreateSpreadsheetColumnOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				switch output := output.(type) {
				case *taskCreateSpreadsheetColumnOutput:
					capturedOutput = *output
				}
				return nil
			})

			// Set up error handling
			var executionErr error
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				executionErr = err
			})

			if tc.expectErr {
				ow.WriteDataMock.Optional()
			} else {
				eh.ErrorMock.Optional()
			}

			// Execute the test
			err = exe.Execute(context.Background(), []*base.Job{job})

			if tc.expectErr {
				c.Assert(executionErr, qt.Not(qt.IsNil))
				if tc.expectedErrMsg != "" {
					c.Assert(executionErr.Error(), qt.Equals, tc.expectedErrMsg)
				}
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(capturedOutput, qt.DeepEquals, tc.expectedOutput)
			}
		})
	}
}
