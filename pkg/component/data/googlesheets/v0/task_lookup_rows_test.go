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
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestLookupRows(t *testing.T) {
	c := qt.New(t)

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet1!1:1" && r.Method == "GET" {
			// Return mock headers response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"values": [["Header1", "Header2", "Header3"]]
			}`))
			return
		}
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet1" && r.Method == "GET" {
			// Return mock all rows response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"values": [
					["Header1", "Header2", "Header3"],
					["Value1", "Value2", "Value3"],
					["Value4", "Value5", "Value6"],
					["Value7", "Value8", "Value9"]
				]
			}`))
			return
		}
		http.Error(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path), http.StatusNotFound)
	}))
	defer ts.Close()

	testCases := []struct {
		name           string
		input          taskLookupRowsInput
		expectedOutput taskLookupRowsOutput
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "ok - lookup rows",
			input: taskLookupRowsInput{
				SharedLink: "https://docs.google.com/spreadsheets/d/test-id",
				SheetName:  "sheet1",
				ColumnName: "Header2",
				Value:      "Value2",
			},
			expectedOutput: taskLookupRowsOutput{
				Rows: []Row{
					{
						RowNumber: 2,
						RowValue: map[string]format.Value{
							"Header1": data.NewString("Value1"),
							"Header2": data.NewString("Value2"),
							"Header3": data.NewString("Value3"),
						},
					},
				},
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "error - invalid spreadsheet ID",
			input: taskLookupRowsInput{
				SharedLink: "invalid-link",
				SheetName:  "sheet1",
				ColumnName: "Header1",
				Value:      "Value1",
			},
			expectedOutput: taskLookupRowsOutput{},
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
				Task:      taskLookupRows,
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
				case *taskLookupRowsInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput taskLookupRowsOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = *(output.(*taskLookupRowsOutput))
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
