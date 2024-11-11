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

func TestSymmetricDifference(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name           string
		input          symmetricDifferenceInput
		expectErr      bool
		expectedErrMsg string
		expected       format.Value
	}{
		{
			name: "ok - simple string symmetric difference",
			input: symmetricDifferenceInput{
				Data: []format.Value{
					data.Array{
						data.NewString("a"),
						data.NewString("b"),
					},
					data.Array{
						data.NewString("b"),
						data.NewString("c"),
					},
				},
			},
			expectErr: false,
			expected: data.Array{
				data.NewString("a"),
				data.NewString("c"),
			},
		},
		{
			name: "ok - object symmetric difference",
			input: symmetricDifferenceInput{
				Data: []format.Value{
					data.Map{
						"a": data.NewString("1"),
						"b": data.NewString("2"),
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
				"c": data.NewString("3"),
			},
		},
		{
			name: "ok - single input array",
			input: symmetricDifferenceInput{
				Data: []format.Value{
					data.Array{
						data.NewString("a"),
						data.NewString("b"),
					},
				},
			},
			expectErr: false,
			expected: data.Array{
				data.NewString("a"),
				data.NewString("b"),
			},
		},
		{
			name: "ok - empty arrays",
			input: symmetricDifferenceInput{
				Data: []format.Value{
					data.Array{},
					data.Array{},
				},
			},
			expectErr: false,
			expected:  data.Array{},
		},
		{
			name: "nok - mixed types",
			input: symmetricDifferenceInput{
				Data: []format.Value{
					data.Array{data.NewString("a")},
					data.Map{"b": data.NewString("2")},
				},
			},
			expectErr:      true,
			expectedErrMsg: "all elements must be of the same type: expected data.Array, got data.Map at index 1",
		},
		{
			name: "nok - unsupported type",
			input: symmetricDifferenceInput{
				Data: []format.Value{
					data.NewString("a"),
					data.NewString("b"),
				},
			},
			expectErr:      true,
			expectedErrMsg: "unsupported type for symmetric difference: *data.stringData (must be either array or object)",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskSymmetricDifference,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)

			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *symmetricDifferenceInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput symmetricDifferenceOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(symmetricDifferenceOutput)
				return nil
			})

			var executionErr error
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				executionErr = err
			})

			if tc.expectErr {
				ow.WriteDataMock.Optional()
			} else {
				eh.ErrorMock.Optional()
			}

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
