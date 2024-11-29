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

func TestSplit(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          splitInput
		expectErr      bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - split array into equal groups",
			input: splitInput{
				Data: data.Array{
					data.NewNumberFromInteger(1),
					data.NewNumberFromInteger(2),
					data.NewNumberFromInteger(3),
					data.NewNumberFromInteger(4),
				},
				Size: 2,
			},
			expectErr: false,
			expected: data.Array{
				data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
				data.Array{data.NewNumberFromInteger(3), data.NewNumberFromInteger(4)},
			},
		},
		{
			name: "ok - split array with uneven groups",
			input: splitInput{
				Data: data.Array{
					data.NewString("a"),
					data.NewString("b"),
					data.NewString("c"),
				},
				Size: 2,
			},
			expectErr: false,
			expected: data.Array{
				data.Array{data.NewString("a"), data.NewString("b")},
				data.Array{data.NewString("c")},
			},
		},
		{
			name: "ok - split empty array",
			input: splitInput{
				Data: data.Array{},
				Size: 2,
			},
			expectErr: false,
			expected:  data.Array{},
		},
		{
			name: "ok - split object into chunks",
			input: splitInput{
				Data: data.Map{
					"a": data.NewString("1"),
					"b": data.NewString("2"),
					"c": data.NewString("3"),
					"d": data.NewString("4"),
				},
				Size: 2,
			},
			expectErr: false,
			expected: data.Array{
				data.Map{
					"a": data.NewString("1"),
					"b": data.NewString("2"),
				},
				data.Map{
					"c": data.NewString("3"),
					"d": data.NewString("4"),
				},
			},
		},
		{
			name: "ok - split object with uneven properties",
			input: splitInput{
				Data: data.Map{
					"a": data.NewString("1"),
					"b": data.NewString("2"),
					"c": data.NewString("3"),
				},
				Size: 2,
			},
			expectErr: false,
			expected: data.Array{
				data.Map{
					"a": data.NewString("1"),
					"b": data.NewString("2"),
				},
				data.Map{
					"c": data.NewString("3"),
				},
			},
		},
		{
			name: "ok - split object with consistent ordering",
			input: splitInput{
				Data: data.Map{
					"c": data.NewString("3"),
					"a": data.NewString("1"),
					"d": data.NewString("4"),
					"b": data.NewString("2"),
				},
				Size: 2,
			},
			expectErr: false,
			expected: data.Array{
				data.Map{
					"a": data.NewString("1"),
					"b": data.NewString("2"),
				},
				data.Map{
					"c": data.NewString("3"),
					"d": data.NewString("4"),
				},
			},
		},
		{
			name: "ok - split empty object",
			input: splitInput{
				Data: data.Map{},
				Size: 2,
			},
			expectErr: false,
			expected:  data.Array{},
		},
		{
			name: "ok - split array with nested structures",
			input: splitInput{
				Data: data.Array{
					data.Map{"x": data.NewNumberFromInteger(1)},
					data.Array{data.NewString("a")},
					data.Map{"y": data.NewNumberFromInteger(2)},
					data.Array{data.NewString("b")},
				},
				Size: 2,
			},
			expectErr: false,
			expected: data.Array{
				data.Array{
					data.Map{"x": data.NewNumberFromInteger(1)},
					data.Array{data.NewString("a")},
				},
				data.Array{
					data.Map{"y": data.NewNumberFromInteger(2)},
					data.Array{data.NewString("b")},
				},
			},
		},
		{
			name: "nok - unsupported type (number)",
			input: splitInput{
				Data: data.NewNumberFromInteger(42),
				Size: 2,
			},
			expectErr:      true,
			expectedErrMsg: "unsupported type for split: *data.numberData (must be array or object)",
		},
		{
			name: "nok - invalid size (zero) for array",
			input: splitInput{
				Data: data.Array{
					data.NewString("a"),
					data.NewString("b"),
				},
				Size: 0,
			},
			expectErr:      true,
			expectedErrMsg: "size must be greater than 0 for array splitting",
		},
		{
			name: "nok - invalid size (negative) for object",
			input: splitInput{
				Data: data.Map{
					"a": data.NewString("1"),
					"b": data.NewString("2"),
				},
				Size: -1,
			},
			expectErr:      true,
			expectedErrMsg: "size must be greater than 0 for object splitting",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskSplit,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *splitInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput splitOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(splitOutput)
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
				c.Assert(executionErr.Error(), qt.Equals, tc.expectedErrMsg)
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(capturedOutput.Data, qt.DeepEquals, tc.expected)
			}
		})
	}
}
