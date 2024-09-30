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

func TestComponent_ExecuteGetContactTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetContactMock.
		When(154023114559).
		Then(
			&TaskGetContactResponse{
				Name:           "Fake Contact",
				Email:          "matt.rogers@freshdesk.com",
				TimeZone:       "Casablanca",
				CompanyID:      154001113950,
				ViewAllTickets: false,
				Language:       "en",
				Active:         true,
				CreatedAt:      "2024-08-18T03:48:03Z",
				UpdatedAt:      "2024-08-22T06:51:54Z",
			}, nil)

	tc := struct {
		name       string
		input      TaskGetContactInput
		wantOutput TaskGetContactOutput
	}{
		name: "ok - task get contact",
		input: TaskGetContactInput{
			ContactID: 154023114559,
		},
		wantOutput: TaskGetContactOutput{
			Name:              "Fake Contact",
			Email:             "matt.rogers@freshdesk.com",
			TimeZone:          "Casablanca",
			CompanyID:         154001113950,
			ViewAllTickets:    false,
			Language:          "English",
			Active:            true,
			CreatedAt:         "2024-08-18 03:48:03 UTC",
			UpdatedAt:         "2024-08-22 06:51:54 UTC",
			OtherEmails:       []string{},
			OtherPhoneNumbers: []string{},
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskGetContact},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetContact

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

func TestComponent_ExecuteCreateContactTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	connector := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.CreateContactMock.
		When(
			&TaskCreateContactReq{
				Name:           "New Contact",
				Email:          "newcontact@gmail.com",
				Phone:          "1234567890",
				Description:    "New Contact Description",
				Address:        "New Contact Address",
				JobTitle:       "New Contact Job Title",
				Tags:           []string{"tag1", "tag2"},
				Language:       "zh-TW",
				TimeZone:       "Taipei",
				ViewAllTickets: false,
				CompanyID:      154001113950,
				OtherCompanies: []taskCreateContactReqOtherCompany{
					{
						CompanyID:      154001162614,
						ViewAllTickets: true,
					},
				},
				OtherEmails: []string{"otherrandomemail@gmail.com"},
			}).
		Then(&TaskCreateContactResponse{
			ID:        154023626050,
			CreatedAt: "2024-08-29T06:24:40Z",
		}, nil)

	tc := struct {
		name       string
		input      TaskCreateContactInput
		wantOutput TaskCreateContactOutput
	}{
		name: "ok - task create contact",
		input: TaskCreateContactInput{
			Name:           "New Contact",
			Email:          "newcontact@gmail.com",
			Phone:          "1234567890",
			Description:    "New Contact Description",
			Address:        "New Contact Address",
			JobTitle:       "New Contact Job Title",
			Tags:           []string{"tag1", "tag2"},
			Language:       "Chinese (Traditional)",
			TimeZone:       "Taipei",
			ViewAllTickets: false,
			CompanyID:      154001113950,
			OtherCompanies: []string{"154001162614;true"},
			OtherEmails:    []string{"otherrandomemail@gmail.com"},
		},
		wantOutput: TaskCreateContactOutput{
			ID:        154023626050,
			CreatedAt: "2024-08-29 06:24:40 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskCreateContact},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskCreateContact

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
