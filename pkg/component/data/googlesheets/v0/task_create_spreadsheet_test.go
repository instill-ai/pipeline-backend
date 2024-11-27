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

func TestCreateSpreadsheet(t *testing.T) {
	c := qt.New(t)

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v4/spreadsheets" && r.Method == "POST" {
			// Return mock spreadsheet response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"spreadsheetId": "test-id",
				"spreadsheetUrl": "https://test-spreadsheet-url",
				"sheets": [{"properties": {"title": "sheet1"}}]
			}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/test-id:batchUpdate" && r.Method == "POST" {
			// Return mock batch update response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet1!A1" && r.Method == "PUT" {
			// Return mock update values response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet2!A1" && r.Method == "PUT" {
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
		input          taskCreateSpreadsheetInput
		expectedOutput taskCreateSpreadsheetOutput
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "ok - create spreadsheet with single sheet",
			input: taskCreateSpreadsheetInput{
				Title: "Test Spreadsheet",
				Sheets: []sheet{
					{
						Name:    "sheet1",
						Headers: []string{"Header1", "Header2"},
					},
				},
			},
			expectedOutput: taskCreateSpreadsheetOutput{
				SharedLink: "https://test-spreadsheet-url",
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "ok - create spreadsheet with multiple sheets",
			input: taskCreateSpreadsheetInput{
				Title: "Test Spreadsheet Multiple",
				Sheets: []sheet{
					{
						Name:    "sheet1",
						Headers: []string{"Header1", "Header2"},
					},
					{
						Name:    "sheet2",
						Headers: []string{"Header3", "Header4"},
					},
				},
			},
			expectedOutput: taskCreateSpreadsheetOutput{
				SharedLink: "https://test-spreadsheet-url",
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "ok - create spreadsheet without headers",
			input: taskCreateSpreadsheetInput{
				Title: "Test Spreadsheet No Headers",
				Sheets: []sheet{
					{
						Name: "sheet1",
					},
				},
			},
			expectedOutput: taskCreateSpreadsheetOutput{
				SharedLink: "https://test-spreadsheet-url",
			},
			expectErr:      false,
			expectedErrMsg: "",
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
				Task:      taskCreateSpreadsheet,
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
				case *taskCreateSpreadsheetInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput *taskCreateSpreadsheetOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*taskCreateSpreadsheetOutput)
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
				c.Assert(capturedOutput.SharedLink, qt.Equals, tc.expectedOutput.SharedLink)
			}
		})
	}
}
