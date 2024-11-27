package googlesheets

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestDeleteMultipleRows(t *testing.T) {
	c := qt.New(t)

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v4/spreadsheets/test-id" && r.Method == "GET" {
			// Return mock spreadsheet response with sheet info
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"sheets": [
					{
						"properties": {
							"sheetId": 0,
							"title": "sheet1"
						}
					}
				]
			}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/test-id:batchUpdate" && r.Method == "POST" {
			// Return mock batch update response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{}`))
			return
		}
		http.Error(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path), http.StatusNotFound)
	}))
	defer ts.Close()

	testCases := []struct {
		name           string
		input          taskDeleteMultipleRowsInput
		expectedOutput taskDeleteMultipleRowsOutput
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "ok - delete multiple rows",
			input: taskDeleteMultipleRowsInput{
				SharedLink: "https://docs.google.com/spreadsheets/d/test-id",
				SheetName:  "sheet1",
				RowNumbers: []int{2, 4},
			},
			expectedOutput: taskDeleteMultipleRowsOutput{
				Success: true,
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "error - invalid spreadsheet ID",
			input: taskDeleteMultipleRowsInput{
				SharedLink: "invalid-link",
				SheetName:  "sheet1",
				RowNumbers: []int{2, 4},
			},
			expectedOutput: taskDeleteMultipleRowsOutput{},
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
				Task:      taskDeleteMultipleRows,
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
				case *taskDeleteMultipleRowsInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput taskDeleteMultipleRowsOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = *(output.(*taskDeleteMultipleRowsOutput))
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
