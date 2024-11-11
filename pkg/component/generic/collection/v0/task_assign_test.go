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

func TestAssign(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          assignInput
		expectErr      bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - simple primitive type assignment (int)",
			input: assignInput{
				Data:  nil,
				Path:  "",
				Value: data.NewNumberFromInteger(10),
			},
			expectErr:      false,
			expected:       data.NewNumberFromInteger(10),
			expectedErrMsg: "",
		},
		{
			name: "ok - simple primitive type assignment (string)",
			input: assignInput{
				Data:  nil,
				Path:  "",
				Value: data.NewString("a"),
			},
			expectErr:      false,
			expected:       data.NewString("a"),
			expectedErrMsg: "",
		},
		{
			name: "ok - simple array index",
			input: assignInput{
				Data:  data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
				Path:  ".[1]",
				Value: data.NewNumberFromInteger(10),
			},
			expectErr: false,
			expected: data.Array{
				data.NewNumberFromInteger(1),
				data.NewNumberFromInteger(10),
				data.NewNumberFromInteger(3),
			},
			expectedErrMsg: "",
		},
		{
			name: "ok - simple object key",
			input: assignInput{
				Data: data.Map{
					"name": data.NewString("John"),
					"age":  data.NewNumberFromInteger(30),
				},
				Path:  "name",
				Value: data.NewString("Jane"),
			},
			expectErr: false,
			expected: data.Map{
				"name": data.NewString("Jane"),
				"age":  data.NewNumberFromInteger(30),
			},
			expectedErrMsg: "",
		},
		{
			name: "ok - nested array and object",
			input: assignInput{
				Data: data.Array{
					data.Map{
						"name": data.NewString("John"),
						"age":  data.NewNumberFromInteger(30),
					},
					data.Map{
						"name": data.NewString("Jane"),
						"age":  data.NewNumberFromInteger(25),
					},
				},
				Path:  ".[0].name",
				Value: data.NewString("Johnny"),
			},
			expectErr: false,
			expected: data.Array{
				data.Map{
					"name": data.NewString("Johnny"),
					"age":  data.NewNumberFromInteger(30),
				},
				data.Map{
					"name": data.NewString("Jane"),
					"age":  data.NewNumberFromInteger(25),
				},
			},
			expectedErrMsg: "",
		},
		{
			name: "ok - deep nested path",
			input: assignInput{
				Data: data.Map{
					"users": data.Array{
						data.Map{
							"metadata": data.Map{
								"tags": data.Array{
									data.NewString("tag1"),
									data.NewString("tag2"),
								},
							},
						},
					},
				},
				Path:  "users.[0].metadata.tags.[1]",
				Value: data.NewString("new-tag"),
			},
			expectErr: false,
			expected: data.Map{
				"users": data.Array{
					data.Map{
						"metadata": data.Map{
							"tags": data.Array{
								data.NewString("tag1"),
								data.NewString("new-tag"),
							},
						},
					},
				},
			},
			expectedErrMsg: "",
		},
		{
			name: "ok - create nested structure",
			input: assignInput{
				Data:  nil,
				Path:  "users.[0].name",
				Value: data.NewString("John"),
			},
			expectErr: false,
			expected: data.Map{
				"users": data.Array{
					data.Map{
						"name": data.NewString("John"),
					},
				},
			},
			expectedErrMsg: "",
		},
		{
			name: "nok - invalid array index",
			input: assignInput{
				Data:  data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
				Path:  ".[invalid]",
				Value: data.NewNumberFromInteger(10),
			},
			expectErr:      true,
			expected:       nil,
			expectedErrMsg: "invalid array index: invalid",
		},
		{
			name: "nok - array index out of bounds",
			input: assignInput{
				Data:  data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
				Path:  ".[5]",
				Value: data.NewNumberFromInteger(10),
			},
			expectErr:      true,
			expected:       nil,
			expectedErrMsg: "array index out of bounds: 5 (array length: 2)",
		},
		{
			name: "nok - invalid path on primitive",
			input: assignInput{
				Data:  data.NewString("simple string"),
				Path:  ".key",
				Value: data.NewString("value"),
			},
			expectErr:      true,
			expected:       nil,
			expectedErrMsg: "cannot set key 'key' on non-object value",
		},
		{
			name: "nok - type mismatch",
			input: assignInput{
				Data:  data.NewNumberFromInteger(42),
				Path:  ".name",
				Value: data.NewString("John"),
			},
			expectErr:      true,
			expected:       nil,
			expectedErrMsg: "cannot set key 'name' on non-object value",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskAssign,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *assignInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput assignOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(assignOutput)
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
