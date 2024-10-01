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

func TestComponent_ExecuteGetProductTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetProductMock.
		When(154000129735).
		Then(
			&TaskGetProductResponse{
				Name:         "Fake Product",
				Description:  "This is a fake product",
				PrimaryEmail: "randomemail@gmail.com",
				CreatedAt:    "2024-08-29T09:35:16Z",
				UpdatedAt:    "2024-08-29T09:35:16Z",
				Default:      true,
			}, nil)

	tc := struct {
		name       string
		input      TaskGetProductInput
		wantOutput TaskGetProductOutput
	}{
		name: "ok - task get product",
		input: TaskGetProductInput{
			ProductID: 154000129735,
		},
		wantOutput: TaskGetProductOutput{
			Name:         "Fake Product",
			Description:  "This is a fake product",
			PrimaryEmail: "randomemail@gmail.com",
			CreatedAt:    "2024-08-29 09:35:16 UTC",
			UpdatedAt:    "2024-08-29 09:35:16 UTC",
			Default:      true,
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetProduct},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetProduct

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
