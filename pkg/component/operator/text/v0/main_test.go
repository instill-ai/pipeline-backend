package text

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestOperator(t *testing.T) {
	c := quicktest.New(t)

	// Define test cases for chunking text and error scenarios
	testcases := []struct {
		name  string
		task  string
		input structpb.Struct
	}{
		{
			name: "chunk texts successfully",
			task: "TASK_CHUNK_TEXT",
			input: structpb.Struct{
				Fields: map[string]*structpb.Value{
					"text": {Kind: &structpb.Value_StringValue{StringValue: "Hello world. This is a test."}},
					"strategy": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"setting": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
								Fields: map[string]*structpb.Value{
									"chunk-method": {Kind: &structpb.Value_StringValue{StringValue: "Token"}},
								},
							}}},
						},
					}}},
				},
			},
		},
		{
			name:  "error case - unsupported task",
			task:  "FAKE_TASK",
			input: structpb.Struct{},
		},
	}

	// Initialize the base component
	bc := base.Component{}
	ctx := context.Background()

	for _, tc := range testcases {
		c.Run(tc.name, func(c *quicktest.C) {
			// Initialize the component
			component := Init(bc)
			c.Assert(component, quicktest.IsNotNil)

			// Create an execution for the task
			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      tc.task,
			})
			c.Assert(err, quicktest.IsNil)
			c.Assert(execution, quicktest.IsNotNil)

			// Mock inputs and outputs
			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(&tc.input, nil)

			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) error {
				if tc.name == "error case - unsupported task" {
					c.Assert(output, quicktest.IsNil)
					return nil
				}
				// Validate output for chunking texts
				c.Assert(output.Fields["chunks"], quicktest.IsNotNil) // Assuming the output has a field "chunks"
				return nil
			})

			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.name == "error case - unsupported task" {
					c.Assert(err, quicktest.ErrorMatches, "not supported task: FAKE_TASK")
				}
			})

			// Execute the job and assert the results
			err = execution.Execute(ctx, []*base.Job{job})
			c.Assert(err, quicktest.IsNil)
		})
	}
}
