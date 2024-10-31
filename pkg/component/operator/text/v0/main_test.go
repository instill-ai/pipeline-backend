package text

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

func TestOperator(t *testing.T) {
	c := quicktest.New(t)

	testcases := []struct {
		name  string
		task  string
		input ChunkTextInput
	}{
		{
			name: "chunk texts",
			task: "TASK_CHUNK_TEXT",
			input: ChunkTextInput{
				Text: "Hello world. This is a test.",
				Strategy: Strategy{
					Setting: Setting{
						ChunkMethod: "Token",
					},
				},
			},
		},
		{
			name:  "error case",
			task:  "FAKE_TASK",
			input: ChunkTextInput{},
		},
	}
	bc := base.Component{}
	ctx := context.Background()
	for i := range testcases {
		tc := &testcases[i]
		c.Run(tc.name, func(c *quicktest.C) {
			component := Init(bc)
			c.Assert(component, quicktest.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      tc.task,
			})
			c.Assert(err, quicktest.IsNil)
			c.Assert(execution, quicktest.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, v interface{}) error {
				*v.(*ChunkTextInput) = tc.input
				return nil
			})
			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output interface{}) error {
				if tc.name == "error case" {
					c.Assert(output, quicktest.IsNil)
					return nil
				}
				return nil
			})
			if tc.name == "error case" {
				ir.ReadDataMock.Optional()
			}
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.name == "error case" {
					c.Assert(err, quicktest.ErrorMatches, "not supported task: FAKE_TASK")
				}
			})
			err = execution.Execute(ctx, []*base.Job{job})
			c.Assert(err, quicktest.IsNil)

		})
	}
}
