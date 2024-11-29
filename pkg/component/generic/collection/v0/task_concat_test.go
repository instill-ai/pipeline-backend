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

func TestConcat(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          concatInput
		expectErr      bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - concat two simple arrays",
			input: concatInput{
				Data: []format.Value{
					data.Array{
						data.NewString("a"),
						data.NewString("b"),
					},
					data.Array{
						data.NewString("c"),
						data.NewString("d"),
					},
				},
			},
			expectErr: false,
			expected: data.Array{
				data.NewString("a"),
				data.NewString("b"),
				data.NewString("c"),
				data.NewString("d"),
			},
		},
		{
			name: "ok - concat three simple arrays",
			input: concatInput{
				Data: []format.Value{
					data.Array{
						data.NewString("a"),
						data.NewString("b"),
					},
					data.Array{
						data.NewString("c"),
						data.NewString("d"),
					},
					data.Array{
						data.NewString("e"),
						data.NewString("f"),
					},
				},
			},
			expectErr: false,
			expected: data.Array{
				data.NewString("a"),
				data.NewString("b"),
				data.NewString("c"),
				data.NewString("d"),
				data.NewString("e"),
				data.NewString("f"),
			},
		},
		{
			name: "ok - concat empty arrays",
			input: concatInput{
				Data: []format.Value{data.Array{}, data.Array{}},
			},
			expectErr: false,
			expected:  data.Array{},
		},
		{
			name: "ok - merge empty objects",
			input: concatInput{
				Data: []format.Value{data.Map{}, data.Map{}},
			},
			expectErr: false,
			expected:  data.Map{},
		},
		{
			name: "ok - merge objects",
			input: concatInput{
				Data: []format.Value{
					data.Map{"a": data.NewNumberFromInteger(1), "b": data.NewNumberFromInteger(2)},
					data.Map{"b": data.NewNumberFromInteger(3), "c": data.NewNumberFromInteger(4)},
					data.Map{"d": data.NewNumberFromInteger(5)},
				},
			},
			expectErr: false,
			expected: data.Map{
				"a": data.NewNumberFromInteger(1),
				"b": data.NewNumberFromInteger(3),
				"c": data.NewNumberFromInteger(4),
				"d": data.NewNumberFromInteger(5),
			},
		},
		{
			name: "nok - mixed types",
			input: concatInput{
				Data: []format.Value{
					data.Map{"a": data.NewNumberFromInteger(1)},
					data.Array{data.NewNumberFromInteger(2)},
				},
			},
			expectErr:      true,
			expectedErrMsg: "all elements must be of the same type: expected data.Map, got data.Array at index 1",
		},
		{
			name: "nok - non-collection input elements (all same type)",
			input: concatInput{
				Data: []format.Value{
					data.NewString("a"),
					data.NewString("b"),
					data.NewString("c"),
				},
			},
			expectErr:      true,
			expectedErrMsg: "unsupported type for concatenation: *data.stringData (must be either array or object)",
		},
		{
			name: "nok - non-collection input elements (all different types)",
			input: concatInput{
				Data: []format.Value{
					data.NewString("a"),
					data.NewNumberFromInteger(1),
					data.NewBoolean(true),
					data.NewNull(),
				},
			},
			expectErr:      true,
			expectedErrMsg: "all elements must be of the same type: expected *data.stringData, got *data.numberData at index 1",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskConcat,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Generate mock job
			ir, ow, eh, job := mock.GenerateMockJob(c)

			// Set up input mock
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *concatInput:
					*input = tc.input
				}
				return nil
			})

			// Set up output capture
			var capturedOutput concatOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(concatOutput)
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
				c.Assert(executionErr.Error(), qt.Contains, tc.expectedErrMsg)
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(capturedOutput.Data, qt.DeepEquals, tc.expected)
			}
		})
	}
}
