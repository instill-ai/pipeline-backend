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

func TestComponent_ExecuteGetCompanyTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetCompanyMock.
		When(154001162614).
		Then(
			&TaskGetCompanyResponse{
				Name:        "Fake Company",
				Description: "This is a fake company",
				Note:        "A note for fake company",
				Domains:     []string{"random@company.com", "random2@company.com"},
				HealthScore: "Doing okay",
				AccountTier: "Premium",
				RenewalDate: "2024-08-29T00:00:00Z",
				Industry:    "Diversified Consumer Services",
				CreatedAt:   "2024-08-29T06:25:48Z",
				UpdatedAt:   "2024-08-29T06:25:48Z",
			}, nil)

	tc := struct {
		name       string
		input      TaskGetCompanyInput
		wantOutput TaskGetCompanyOutput
	}{
		name: "ok - task get company",
		input: TaskGetCompanyInput{
			CompanyID: 154001162614,
		},
		wantOutput: TaskGetCompanyOutput{
			Name:        "Fake Company",
			Description: "This is a fake company",
			Note:        "A note for fake company",
			Domains:     []string{"random@company.com", "random2@company.com"},
			HealthScore: "Doing okay",
			AccountTier: "Premium",
			RenewalDate: "2024-08-29 00:00:00 UTC",
			Industry:    "Diversified Consumer Services",
			CreatedAt:   "2024-08-29 06:25:48 UTC",
			UpdatedAt:   "2024-08-29 06:25:48 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetCompany},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetCompany

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

func TestComponent_ExecuteCreateCompanyTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.CreateCompanyMock.
		When(
			&TaskCreateCompanyReq{
				Name:        "Fake Company",
				Description: "This is a fake company",
				Note:        "A note for fake company",
				Domains:     []string{"randomdomain.com", "randomdomain2.com"},
				HealthScore: "At risk",
				AccountTier: "Basic",
				RenewalDate: "2024-08-30",
				Industry:    "Automotive",
			}).
		Then(&TaskCreateCompanyResponse{
			ID:        154001162922,
			CreatedAt: "2024-08-29T08:23:15Z",
		}, nil)

	tc := struct {
		name       string
		input      TaskCreateCompanyInput
		wantOutput TaskCreateCompanyOutput
	}{
		name: "ok - task create company",
		input: TaskCreateCompanyInput{
			Name:        "Fake Company",
			Description: "This is a fake company",
			Note:        "A note for fake company",
			Domains:     []string{"randomdomain.com", "randomdomain2.com"},
			HealthScore: "At risk",
			AccountTier: "Basic",
			RenewalDate: "2024-08-30",
			Industry:    "Automotive",
		},
		wantOutput: TaskCreateCompanyOutput{
			ID:        154001162922,
			CreatedAt: "2024-08-29 08:23:15 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskCreateCompany},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskCreateCompany

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
