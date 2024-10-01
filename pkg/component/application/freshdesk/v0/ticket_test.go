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
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

const (
	apiKey = "xxxxxxxxxxxxxxxxxxxx"
	domain = "yourdomain"
)

func TestComponent_ExecuteGetTicketTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetTicketMock.
		When(12).
		Then(&TaskGetTicketResponse{
			Subject:                "Test Ticket",
			DescriptionText:        "This is a test ticket",
			Source:                 1,
			Status:                 2,
			Priority:               1,
			TicketType:             "Feature Request",
			Tags:                   []string{"tag1", "tag2"},
			RequesterID:            154023592736,
			ProductID:              154000125389,
			ToEmails:               []string{"test1@gmail.com"},
			Spam:                   false,
			DueBy:                  "2024-08-28T21:00:00Z",
			IsEscalated:            false,
			FirstResponseDueBy:     "2024-08-26T21:00:00Z",
			FirstResponseEscalated: true,
			SentimentScore:         21,
			InitialSentimentScore:  21,
		}, nil)

	tc := struct {
		name       string
		input      TaskGetTicketInput
		wantOutput TaskGetTicketOutput
	}{
		name: "ok - task get ticket",
		input: TaskGetTicketInput{
			TicketID: 12,
		},
		wantOutput: TaskGetTicketOutput{
			Subject:                "Test Ticket",
			DescriptionText:        "This is a test ticket",
			Source:                 "Email",
			Status:                 "Open",
			Priority:               "Low",
			TicketType:             "Feature Request",
			Tags:                   []string{"tag1", "tag2"},
			RequesterID:            154023592736,
			ProductID:              154000125389,
			ToEmails:               []string{"test1@gmail.com"},
			Spam:                   false,
			DueBy:                  "2024-08-28 21:00:00 UTC",
			IsEscalated:            false,
			FirstResponseDueBy:     "2024-08-26 21:00:00 UTC",
			FirstResponseEscalated: true,
			SentimentScore:         21,
			InitialSentimentScore:  21,
			AssociationType:        "No association",
			CCEmails:               []string{},
			ForwardEmails:          []string{},
			ReplyCCEmails:          []string{},
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetTicket},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetTicket

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
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

func TestComponent_ExecuteCreateTicketTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.CreateTicketMock.
		When(&TaskCreateTicketReq{
			Subject:     "Create Test Ticket",
			Description: "Create a test ticket",
			Source:      1,
			Status:      5,
			Priority:    2,
			RequesterID: 154023592736,
			ProductID:   154000125389,
			GroupID:     154000444629,
		}).
		Then(&TaskCreateTicketResponse{
			ID:        50,
			CreatedAt: "2024-08-28T21:00:00Z",
		}, nil)

	tc := struct {
		name       string
		input      TaskCreateTicketInput
		wantOutput TaskCreateTicketOutput
	}{
		name: "ok - task create ticket",
		input: TaskCreateTicketInput{
			Subject:     "Create Test Ticket",
			Description: "Create a test ticket",
			Source:      "Email",
			Status:      "Closed",
			Priority:    "Medium",
			RequesterID: 154023592736,
			ProductID:   154000125389,
			GroupID:     154000444629,
		},
		wantOutput: TaskCreateTicketOutput{
			ID:        50,
			CreatedAt: "2024-08-28 21:00:00 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskCreateTicket},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskCreateTicket

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
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

func TestComponent_ExecuteReplyToTicketTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.ReplyToTicketMock.
		When(50, &TaskReplyToTicketReq{
			Body:      "This is a test reply to a ticket",
			FromEmail: "randomemail@gmail.com",
		}).
		Then(&TaskReplyToTicketResponse{
			ConversationID: 154041545463,
			CreatedAt:      "2024-08-28T15:02:20Z",
		}, nil)

	tc := struct {
		name       string
		input      TaskReplyToTicketInput
		wantOutput TaskReplyToTicketOutput
	}{
		name: "ok - task reply to ticket",
		input: TaskReplyToTicketInput{
			TicketID:  50,
			Body:      "This is a test reply to a ticket",
			FromEmail: "randomemail@gmail.com",
		},
		wantOutput: TaskReplyToTicketOutput{
			ConversationID: 154041545463,
			CreatedAt:      "2024-08-28 15:02:20 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskReplyToTicket},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskReplyToTicket

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
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

func TestComponent_ExecuteCreateTicketNoteTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.CreateTicketNoteMock.
		When(50, &TaskCreateTicketNoteReq{
			Body:         "This is a test note for a ticket",
			NotifyEmails: []string{"agentemail@gmail.com"},
			Private:      true,
			Incoming:     false,
		}).
		Then(&TaskCreateTicketNoteResponse{
			ConversationID: 154041548817,
			CreatedAt:      "2024-08-28T15:22:48Z",
		}, nil)

	tc := struct {
		name       string
		input      TaskCreateTicketNoteInput
		wantOutput TaskCreateTicketNoteOutput
	}{
		name: "ok - task create ticket note",
		input: TaskCreateTicketNoteInput{
			TicketID:     50,
			Body:         "This is a test note for a ticket",
			NotifyEmails: []string{"agentemail@gmail.com"},
			Private:      true,
			Incoming:     false,
		},
		wantOutput: TaskCreateTicketNoteOutput{
			ConversationID: 154041548817,
			CreatedAt:      "2024-08-28 15:22:48 UTC",
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskCreateTicketNote},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskCreateTicketNote

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
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

func TestComponent_ExecuteGetAllConversationsTask(t *testing.T) {
	mc := minimock.NewController(t)
	c := qt.New(t)
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	ctx := context.Background()

	FreshdeskClientMock := NewFreshdeskInterfaceMock(mc)

	FreshdeskClientMock.GetAllConversationsMock.
		When(50, false, "").
		Then([]TaskGetAllConversationsResponse{
			{
				BodyText:       "Conversation - a reply",
				ConversationID: 154041545463,
				Private:        false,
				Incoming:       false,
				ToEmails:       []string{"randomemail@gmail.com"},
				FromEmail:      "theonewhoaddorreply@gmail.com",
				CCEmails:       []string{},
				BCCEmails:      []string{},
				UserID:         154023114553,
				CreatedAt:      "2024-08-28T15:02:20Z",
			},
			{
				BodyText:       "Conversation - a note",
				ConversationID: 154041548817,
				Private:        false,
				Incoming:       false,
				ToEmails:       []string{}, // normally empty for a note
				CCEmails:       []string{},
				BCCEmails:      []string{},
				UserID:         154023114553,
				CreatedAt:      "2024-08-28T15:22:48Z",
			},
		}, "", nil)

	tc := struct {
		name       string
		input      TaskGetAllConversationsInput
		wantOutput TaskGetAllConversationsOutput
	}{
		name: "ok - task get all conversations",
		input: TaskGetAllConversationsInput{
			TicketID: 50,
		},
		wantOutput: TaskGetAllConversationsOutput{
			Conversations: []taskGetAllConversationsOutputConversation{
				{
					BodyText:       "Conversation - a reply",
					ConversationID: 154041545463,
					Private:        false,
					Incoming:       false,
					ToEmails:       []string{"randomemail@gmail.com"},
					FromEmail:      "theonewhoaddorreply@gmail.com",
					CCEmails:       []string{},
					BCCEmails:      []string{},
					UserID:         154023114553,
					CreatedAt:      "2024-08-28 15:02:20 UTC",
				},
				{
					BodyText:       "Conversation - a note",
					ConversationID: 154041548817,
					Private:        false,
					Incoming:       false,
					ToEmails:       []string{}, // normally empty for a note
					CCEmails:       []string{},
					BCCEmails:      []string{},
					UserID:         154023114553,
					CreatedAt:      "2024-08-28 15:22:48 UTC",
				},
			},
			ConversationsLength: 2,
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"api-key": apiKey,
			"domain":  domain,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetAllConversations},
			client:             FreshdeskClientMock,
		}
		e.execute = e.TaskGetAllConversations

		pbIn, err := base.ConvertToStructpb(tc.input)
		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := mock.GenerateMockJob(c)
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
