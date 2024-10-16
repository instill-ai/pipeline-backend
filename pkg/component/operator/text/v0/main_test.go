package hello

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

func TestOperator_Execute(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	c.Run("ok - greet", func(c *qt.C) {
		exec, err := component.CreateExecution(base.ComponentExecution{
			Component: component,
			Task:      taskGreet,
		})
		c.Assert(err, qt.IsNil)

		pbIn, err := structpb.NewStruct(map[string]any{"target": "bolero-wombat"})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(&pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) error {
			// Check JSON in the output string.
			greeting := output.Fields["greeting"].GetStringValue()
			c.Check(greeting, qt.Equals, "Hello, bolero-wombat!")
			return nil
		})
		eh.ErrorMock.Optional()

		err = exec.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)
	})

	c.Run("nok - invalid greetee", func(c *qt.C) {
		x, err := component.CreateExecution(base.ComponentExecution{
			Component: component,
			Task:      taskGreet,
		})
		c.Assert(err, qt.IsNil)

		pbIn, err := structpb.NewStruct(map[string]any{"target": "Voldemort"})
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) error {
			// Check JSON in the output string.
			greeting := output.Fields["greeting"].GetStringValue()
			c.Check(greeting, qt.Equals, "Hello, bolero-wombat!")
			return nil
		})
		eh.ErrorMock.Optional()

		err = x.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.ErrorMatches, "invalid greetee")
		c.Assert(errmsg.Message(err), qt.Matches, "He-Who-Must-Not-Be-Named can't be greeted.")
	})
}

func TestOperator_CreateExecution(t *testing.T) {
	c := qt.New(t)

	bc := base.Component{Logger: zap.NewNop()}
	operator := Init(bc)

	c.Run("nok - unsupported task", func(c *qt.C) {
		task := "FOOBAR"

		_, err := operator.CreateExecution(base.ComponentExecution{
			Component: operator,
			Task:      task,
		})
		c.Check(err, qt.ErrorMatches, "unsupported task")
	})
}

