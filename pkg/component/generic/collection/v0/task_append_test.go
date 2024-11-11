package collection

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestAppend(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          appendInput
		expectErr      bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - append primitives",
			input: appendInput{
				Data:  data.NewString("first"),
				Value: data.NewString("second"),
			},
			expectErr:      false,
			expectedErrMsg: "",
			expected:       data.Array([]format.Value{data.NewString("first"), data.NewString("second")}),
		},
		{
			name: "ok - append a primitive to an array",
			input: appendInput{
				Data:  data.Array([]format.Value{data.NewString("a"), data.NewString("b")}),
				Value: data.NewString("c"),
			},
			expectErr:      false,
			expectedErrMsg: "",
			expected:       data.Array([]format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")}),
		},
		{
			name: "ok - append a primitive to an empty array",
			input: appendInput{
				Data:  data.Array([]format.Value{}),
				Value: data.NewNumberFromInteger(1),
			},
			expectErr:      false,
			expectedErrMsg: "",
			expected:       data.Array([]format.Value{data.NewNumberFromInteger(1)}),
		},
		{
			name: "ok - append an object to an array",
			input: appendInput{
				Data: data.Array([]format.Value{data.NewString("a"), data.NewString("b")}),
				Value: data.Map{
					"foo": data.NewString("a"),
				},
			},
			expectErr:      false,
			expectedErrMsg: "",
			expected:       data.Array([]format.Value{data.NewString("a"), data.NewString("b"), data.Map{"foo": data.NewString("a")}}),
		},
		{
			name: "ok - append a primitive to an object",
			input: appendInput{
				Data: data.Map{
					"foo": data.NewString("a"),
				},
				Value: data.NewString("b"),
			},
			expectErr:      false,
			expectedErrMsg: "",
			expected: data.Array{
				data.Map{
					"foo": data.NewString("a"),
				},
				data.NewString("b"),
			},
		},
		{
			name: "ok - append an object to an object",
			input: appendInput{
				Data: data.Map{
					"foo": data.NewString("a"),
				},
				Value: data.Map{
					"numbers": data.Array([]format.Value{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)}),
					"bar":     data.NewString("b"),
				},
			},
			expectErr:      false,
			expectedErrMsg: "",
			expected: data.Map{
				"foo":     data.NewString("a"),
				"numbers": data.Array([]format.Value{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)}),
				"bar":     data.NewString("b"),
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskAppend,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *appendInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput appendOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(appendOutput)
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
			err = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectErr {
				c.Assert(executionErr, qt.Not(qt.IsNil))
				if tc.expectedErrMsg != "" {
					c.Assert(executionErr.Error(), qt.Equals, tc.expectedErrMsg)
				}
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(capturedOutput.Data, qt.DeepEquals, tc.expected)
			}
		})
	}
}

func TestToArrayElement(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    interface{}
		expected format.Value
	}{
		{
			name:     "ok - string input",
			input:    "test",
			expected: data.NewString("test"),
		},
		{
			name:     "ok - integer input",
			input:    42,
			expected: data.NewNumberFromInteger(42),
		},
		{
			name:     "ok - float input",
			input:    3.14,
			expected: data.NewNumberFromFloat(3.14),
		},
		{
			name:     "ok - boolean input",
			input:    true,
			expected: data.NewBoolean(true),
		},
		{
			name:     "ok - format.Value input",
			input:    data.NewString("already value"),
			expected: data.NewString("already value"),
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			result := toArrayElement(tc.input)
			c.Assert(result, qt.DeepEquals, tc.expected)
		})
	}
}
