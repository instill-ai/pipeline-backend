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

func TestGetRow(t *testing.T) {
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
		if r.URL.Path == "/v4/spreadsheets/test-id/values/sheet1!2:2" && r.Method == "GET" {
			// Return mock row response
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"values": [["Value1", "Value2", "Value3"]]
			}`))
			return
		}
		http.Error(w, fmt.Sprintf("not found: %s %s", r.Method, r.URL.Path), http.StatusNotFound)
	}))
	defer ts.Close()

	testCases := []struct {
		name           string
		input          taskGetRowInput
		expectedOutput taskGetRowOutput
		expectErr      bool
		expectedErrMsg string
	}{
		{
			name: "ok - get row",
			input: taskGetRowInput{
				SharedLink: "https://docs.google.com/spreadsheets/d/test-id",
				SheetName:  "sheet1",
				RowNumber:  2,
			},
			expectedOutput: taskGetRowOutput{
				Row: Row{
					RowNumber: 2,
					RowValue: map[string]format.Value{
						"Header1": data.NewString("Value1"),
						"Header2": data.NewString("Value2"),
						"Header3": data.NewString("Value3"),
					},
				},
			},
			expectErr:      false,
			expectedErrMsg: "",
		},
		{
			name: "error - invalid spreadsheet ID",
			input: taskGetRowInput{
				SharedLink: "invalid-link",
				SheetName:  "sheet1",
				RowNumber:  1,
			},
			expectedOutput: taskGetRowOutput{},
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
				Task:      taskGetRow,
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
				case *taskGetRowInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput taskGetRowOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = *(output.(*taskGetRowOutput))
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
