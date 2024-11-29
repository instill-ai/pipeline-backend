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

func TestIntersection(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          intersectionInput
		expectErr    bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - intersection of two arrays",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
					data.Array{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3), data.NewNumberFromInteger(4)},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
		},
		{
			name: "ok - empty arrays",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{},
					data.Array{},
				},
			},
			expectErr: false,
			expected:    data.Array{},
		},
		{
			name: "ok - single array",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewString("a"), data.NewString("b"), data.NewString("c")},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewString("a"), data.NewString("b"), data.NewString("c")},
		},
		{
			name: "ok - multiple arrays with common elements",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewString("a"), data.NewString("b"), data.NewString("c")},
					data.Array{data.NewString("b"), data.NewString("c"), data.NewString("d")},
					data.Array{data.NewString("c"), data.NewString("d"), data.NewString("e")},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewString("c")},
		},
		{
			name: "ok - arrays with no intersection",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewString("a"), data.NewString("b")},
					data.Array{data.NewString("c"), data.NewString("d")},
				},
			},
			expectErr: false,
			expected:    data.Array{},
		},
		{
			name: "ok - arrays with mixed types",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewString("a"), data.NewNumberFromInteger(1), data.NewString("b")},
					data.Array{data.NewNumberFromInteger(1), data.NewString("b"), data.NewString("c")},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewNumberFromInteger(1), data.NewString("b")},
		},
		{
			name: "ok - arrays with duplicates",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewString("a"), data.NewString("a"), data.NewString("b")},
					data.Array{data.NewString("a"), data.NewString("b"), data.NewString("b")},
				},
			},
			expectErr: false,
			expected:    data.Array{data.NewString("a"), data.NewString("b")},
		},
		{
			name: "ok - object intersection",
			input: intersectionInput{
				Data: []format.Value{
					data.Map{"a": data.NewString("1"), "b": data.NewString("2")},
					data.Map{"b": data.NewString("2"), "c": data.NewString("3")},
				},
			},
			expectErr: false,
			expected:    data.Map{"b": data.NewString("2")},
		},
		{
			name: "ok - empty input",
			input: intersectionInput{
				Data: []format.Value{},
			},
			expectErr: false,
			expected:    data.Array{},
		},
		{
			name: "nok - unsupported type (string)",
			input: intersectionInput{
				Data: []format.Value{
					data.NewString("a"),
					data.NewString("b"),
				},
			},
			expectErr:    true,
			expectedErrMsg: "unsupported type for intersection: *data.stringData (must be either array or object)",
			expected:       nil,
		},
		{
			name: "nok - unsupported type (number)",
			input: intersectionInput{
				Data: []format.Value{
					data.NewNumberFromInteger(1),
					data.NewNumberFromInteger(2),
				},
			},
			expectErr:    true,
			expectedErrMsg: "unsupported type for intersection: *data.numberData (must be either array or object)",
			expected:       nil,
		},
		{
			name: "nok - mixed array and object",
			input: intersectionInput{
				Data: []format.Value{
					data.Array{data.NewString("a"), data.NewString("b")},
					data.Map{"a": data.NewString("1")},
				},
			},
			expectErr:    true,
			expectedErrMsg: "all elements must be of the same type: expected data.Array, got data.Map at index 1",
			expected:       nil,
		},
		{
			name: "nok - mixed object and array",
			input: intersectionInput{
				Data: []format.Value{
					data.Map{"a": data.NewString("1")},
					data.Array{data.NewString("a"), data.NewString("b")},
				},
			},
			expectErr:    true,
			expectedErrMsg: "all elements must be of the same type: expected data.Map, got data.Array at index 1",
			expected:       nil,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskIntersection,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *intersectionInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput intersectionOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(intersectionOutput)
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
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(capturedOutput.Data, qt.DeepEquals, tc.expected)
			}
		})
	}
}
