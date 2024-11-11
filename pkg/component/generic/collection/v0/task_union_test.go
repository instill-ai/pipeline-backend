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

func TestUnion(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          unionInput
		expectErr      bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - union of arrays",
			input: unionInput{
				Data: []format.Value{
					data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
					data.Array{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
				},
			},
			expectErr: false,
			expected:  data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
		},
		{
			name: "ok - union of objects",
			input: unionInput{
				Data: []format.Value{
					data.Map{"a": data.NewString("1"), "b": data.NewString("2")},
					data.Map{"b": data.NewString("3"), "c": data.NewString("3")},
				},
			},
			expectErr: false,
			expected:  data.Map{"a": data.NewString("1"), "b": data.NewString("3"), "c": data.NewString("3")},
		},
		{
			name: "ok - empty input",
			input: unionInput{
				Data: []format.Value{},
			},
			expectErr: false,
			expected:  data.Array{},
		},
		{
			name: "ok - union of nested arrays",
			input: unionInput{
				Data: []format.Value{
					data.Array{
						data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
						data.NewString("a"),
					},
					data.Array{
						data.Array{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
						data.NewString("a"),
					},
				},
			},
			expectErr: false,
			expected: data.Array{
				data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
				data.NewString("a"),
				data.Array{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
			},
		},
		{
			name: "ok - union of nested objects",
			input: unionInput{
				Data: []format.Value{
					data.Map{
						"a": data.NewString("1"),
						"b": data.Map{
							"x": data.NewNumberFromInteger(1),
							"y": data.NewNumberFromInteger(2),
						},
					},
					data.Map{
						"b": data.Map{
							"y": data.NewNumberFromInteger(3),
							"z": data.NewNumberFromInteger(4),
						},
						"c": data.NewString("3"),
					},
				},
			},
			expectErr: false,
			expected: data.Map{
				"a": data.NewString("1"),
				"b": data.Map{
					"y": data.NewNumberFromInteger(3),
					"z": data.NewNumberFromInteger(4),
				},
				"c": data.NewString("3"),
			},
		},
		{
			name: "ok - union of complex nested structures",
			input: unionInput{
				Data: []format.Value{
					data.Map{
						"arr": data.Array{
							data.NewNumberFromInteger(1),
							data.Map{"x": data.NewString("1")},
						},
						"obj": data.Map{
							"nested": data.Array{data.NewString("a")},
						},
					},
					data.Map{
						"arr": data.Array{
							data.NewNumberFromInteger(2),
							data.Map{"y": data.NewString("2")},
						},
						"obj": data.Map{
							"nested": data.Array{data.NewString("b")},
						},
					},
				},
			},
			expectErr: false,
			expected: data.Map{
				"arr": data.Array{
					data.NewNumberFromInteger(2),
					data.Map{"y": data.NewString("2")},
				},
				"obj": data.Map{
					"nested": data.Array{data.NewString("b")},
				},
			},
		},
		{
			name: "ok - union of arrays with mixed nested types",
			input: unionInput{
				Data: []format.Value{
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
			expectErr: false,
			expected: data.Array{
				data.Map{"x": data.NewNumberFromInteger(1)},
				data.Array{data.NewString("a")},
				data.Map{"y": data.NewNumberFromInteger(2)},
				data.Array{data.NewString("b")},
			},
		},
		{
			name: "ok - union with empty nested structures",
			input: unionInput{
				Data: []format.Value{
					data.Map{
						"empty_arr": data.Array{},
						"empty_obj": data.Map{},
					},
					data.Map{
						"empty_arr": data.Array{data.NewString("a")},
						"empty_obj": data.Map{"x": data.NewNumberFromInteger(1)},
					},
				},
			},
			expectErr: false,
			expected: data.Map{
				"empty_arr": data.Array{data.NewString("a")},
				"empty_obj": data.Map{"x": data.NewNumberFromInteger(1)},
			},
		},
		{
			name: "ok - deeply nested array union",
			input: unionInput{
				Data: []format.Value{
					data.Array{
						data.Array{
							data.Array{data.NewNumberFromInteger(1)},
						},
					},
					data.Array{
						data.Array{
							data.Array{data.NewNumberFromInteger(2)},
						},
					},
				},
			},
			expectErr: false,
			expected: data.Array{
				data.Array{
					data.Array{data.NewNumberFromInteger(1)},
				},
				data.Array{
					data.Array{data.NewNumberFromInteger(2)},
				},
			},
		},
		{
			name: "nok - mixed types",
			input: unionInput{
				Data: []format.Value{
					data.Array{data.NewString("a")},
					data.Map{"a": data.NewString("1")},
				},
			},
			expectErr:      true,
			expectedErrMsg: "all elements must be of the same type: expected data.Array, got data.Map at index 1",
		},
		{
			name: "nok - unsupported type",
			input: unionInput{
				Data: []format.Value{
					data.NewString("a"),
					data.NewString("b"),
				},
			},
			expectErr:      true,
			expectedErrMsg: "unsupported type for union: *data.stringData (must be either array or object)",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskUnion,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *unionInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput unionOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(unionOutput)
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
