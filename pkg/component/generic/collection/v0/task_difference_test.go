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

func TestDifference(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          differenceInput
		expectErr    bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - simple string difference with multiple arrays",
			input: differenceInput{
				Data: []format.Value{
					data.Array{
						data.NewString("a"),
						data.NewString("b"),
						data.NewString("c"),
					},
					data.Array{
						data.NewString("b"),
					},
					data.Array{
						data.NewString("c"),
					},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewString("a")},
		},
		{
			name: "ok - empty array difference",
			input: differenceInput{
				Data: []format.Value{
					data.Array{},
					data.Array{},
				},
			},
			expectErr: false,
			expected:    data.Array{},
		},
		{
			name: "ok - number array difference",
			input: differenceInput{
				Data: []format.Value{
					data.Array{
						data.NewNumberFromInteger(1),
						data.NewNumberFromInteger(2),
						data.NewNumberFromInteger(3),
					},
					data.Array{
						data.NewNumberFromInteger(2),
						data.NewNumberFromInteger(3),
					},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewNumberFromInteger(1)},
		},
		{
			name: "ok - object difference",
			input: differenceInput{
				Data: []format.Value{
					data.Map{
						"a": data.NewString("1"),
						"b": data.NewString("2"),
						"c": data.NewString("3"),
					},
					data.Map{
						"b": data.NewString("2"),
						"c": data.NewString("3"),
					},
				},
			},
			expectErr: false,
			expected: data.Map{
				"a": data.NewString("1"),
			},
		},
		{
			name: "ok - empty array difference",
			input: differenceInput{
				Data: []format.Value{
					data.Array{},
					data.Array{},
				},
			},
			expectErr: false,
			expected:    data.Array{},
		},
		{
			name: "ok - empty object difference",
			input: differenceInput{
				Data: []format.Value{
					data.Map{},
					data.Map{},
				},
			},
			expectErr: false,
			expected:    data.Map{},
		},
		{
			name: "nok - mixed types",
			input: differenceInput{
				Data: []format.Value{
					data.Map{"a": data.NewNumberFromInteger(1)},
					data.Array{data.NewNumberFromInteger(2)},
				},
			},
			expectErr:    true,
			expectedErrMsg: "all elements must be of the same type: expected data.Map, got data.Array at index 1",
		},
		{
			name: "nok - non-collection input elements",
			input: differenceInput{
				Data: []format.Value{
					data.NewString("a"),
					data.NewString("b"),
				},
			},
			expectErr:    true,
			expectedErrMsg: "unsupported type for concatenation: *data.stringData (must be either array or object)",
		},
		{
			name: "nok - empty input",
			input: differenceInput{
				Data: []format.Value{},
			},
			expectErr: false,
			expected:    data.Array{},
		},
		{
			name: "nok - different value types in arrays",
			input: differenceInput{
				Data: []format.Value{
					data.Array{
						data.NewString("a"),
						data.NewNumberFromInteger(1),
					},
					data.Array{
						data.NewString("b"),
					},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewString("a"), data.NewNumberFromInteger(1)},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskDifference,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *differenceInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput differenceOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(differenceOutput)
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
