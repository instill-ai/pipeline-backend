package freshdesk

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestComponent_ExecuteGetAllTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetAllMock.
		When("Tickets", false, "").
		Then([]TaskGetAllResponse{
			{ID: 1},
			{ID: 2},
			{ID: 3},
			{ID: 4},
			{ID: 5},
		}, "https://yourdomain.freshdesk.com/api/v2/tickets?page=2", nil)

	FreshdeskClientMock.GetAllMock.
		When("Tickets", true, "https://yourdomain.freshdesk.com/api/v2/tickets?page=2").
		Then([]TaskGetAllResponse{
			{ID: 6},
			{ID: 7},
		}, "", nil)

	tc := struct {
		name       string
		input      TaskGetAllInput
		wantOutput TaskGetAllOutput
	}{
		name: "ok - task get all",
		input: TaskGetAllInput{
			ObjectType: "Tickets",
			Length:     7,
		},
		wantOutput: TaskGetAllOutput{
			IDs:      []int64{1, 2, 3, 4, 5, 6, 7},
			IDLength: 7,
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetAll},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetAll

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {

			outJSON, err := protojson.Marshal(output)
			c.Assert(err, qt.IsNil)

			c.Check(outJSON, qt.JSONEquals, tc.wantOutput)
			return nil
		})
		eh.ErrorMock.Optional()

		err = e.Execute(ctx, []*base.Job{job})

		c.Assert(err, qt.IsNil)

	})
}
