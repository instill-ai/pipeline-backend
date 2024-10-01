package hubspot

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// mockClient is in contact_test.go

// Mock Thread struct and its functions
type MockThread struct{}

func (s *MockThread) Get(threadID string, param string) (*TaskGetThreadResp, error) {

	var fakeThread TaskGetThreadResp
	if threadID == "7509711154" {
		fakeThread = TaskGetThreadResp{
			Results: []taskGetThreadRespResult{
				{
					CreatedAt: "2024-07-02T10:42:15Z",
					Senders: []taskGetThreadRespUser{
						{
							Name: "Brian Halligan (Sample Contact)",
							DeliveryIdentifier: taskGetThreadRespIdentifier{
								Type:  "HS_EMAIL_ADDRESS",
								Value: "bh@hubspot.com",
							},
						},
					},
					Recipients: []taskGetThreadRespUser{
						{
							DeliveryIdentifier: taskGetThreadRespIdentifier{
								Type:  "HS_EMAIL_ADDRESS",
								Value: "fake_email@gmail.com",
							},
						},
					},
					Text:             "Just random content inside",
					Subject:          "A fake message",
					ChannelID:        "1002",
					ChannelAccountID: "638727358",
					Type:             "MESSAGE",
				},
			},
		}
	}

	return &fakeThread, nil
}

func (s *MockThread) Insert(threadID string, message *TaskInsertMessageReq) (*TaskInsertMessageResp, error) {

	res := &TaskInsertMessageResp{}
	if threadID == "7509711154" {
		res.Status = taskInsertMessageRespStatusType{
			StatusType: "SENT",
		}
	}
	return res, nil
}

func TestComponent_ExecuteGetThreadTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)
	tc := struct {
		name     string
		input    string
		wantResp TaskGetThreadOutput
	}{
		name:  "ok - get thread",
		input: "7509711154",
		wantResp: TaskGetThreadOutput{
			Results: []taskGetThreadOutputResult{
				{
					CreatedAt: "2024-07-02T10:42:15Z",
					Sender: taskGetThreadOutputSender{
						Name:  "Brian Halligan (Sample Contact)",
						Type:  "HS_EMAIL_ADDRESS",
						Value: "bh@hubspot.com",
					},
					Recipients: []taskGetThreadOutputRecipient{
						{
							Type:  "HS_EMAIL_ADDRESS",
							Value: "fake_email@gmail.com",
						},
					},
					Text:             "Just random content inside",
					Subject:          "A fake message",
					ChannelID:        "1002",
					ChannelAccountID: "638727358",
				},
			},
			NoOfMessages: 1,
		},
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"token": bearerToken,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskGetThread},
			client:             createMockClient(),
		}
		e.execute = e.GetThread

		pbInput, err := structpb.NewStruct(map[string]any{
			"thread-id": tc.input,
		})

		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbInput, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			resJSON, err := protojson.Marshal(output)
			c.Assert(err, qt.IsNil)

			c.Check(resJSON, qt.JSONEquals, tc.wantResp)
			return nil
		})
		eh.ErrorMock.Optional()
		err = e.Execute(ctx, []*base.Job{job})
		c.Assert(err, qt.IsNil)

	})
}

func TestComponent_ExecuteInsertMessageTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{Logger: zap.NewNop()}
	component := Init(bc)

	tc := struct {
		name     string
		input    TaskInsertMessageInput
		wantResp string
	}{

		name: "ok - insert message",
		input: TaskInsertMessageInput{
			ThreadID:         "7509711154",
			SenderActorID:    "A-12345678",
			Recipients:       []string{"randomemail@gmail.com"},
			ChannelAccountID: "123456789",
			Subject:          "A fake message",
			Text:             "A message with random content inside",
		},
		wantResp: "SENT",
	}

	c.Run(tc.name, func(c *qt.C) {
		setup, err := structpb.NewStruct(map[string]any{
			"token": bearerToken,
		})
		c.Assert(err, qt.IsNil)

		e := &execution{
			ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: taskInsertMessage},
			client:             createMockClient(),
		}
		e.execute = e.InsertMessage

		pbInput, err := base.ConvertToStructpb(tc.input)

		c.Assert(err, qt.IsNil)

		ir, ow, eh, job := base.GenerateMockJob(c)
		ir.ReadMock.Return(pbInput, nil)
		ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
			resString := output.Fields["status"].GetStringValue()
			c.Check(resString, qt.Equals, tc.wantResp)
			return nil
		})
		eh.ErrorMock.Optional()
		err = e.Execute(ctx, []*base.Job{job})

		c.Assert(err, qt.IsNil)

	})

}
