package whatsapp

import (
	"context"
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type MockWhatsAppClientSendMessage struct{}

// If you send a normal whatsapp message and not a template, message status will not be returned by the API

func (c *MockWhatsAppClientSendMessage) SendMessageAPI(req interface{}, resp interface{}, PhoneNumberID string) error {

	jsonData := `{
		"messaging_product": "whatsapp",
		"contacts": [
			{
				"input": "886901234567",
				"wa_id": "886901234567"
			}
		],
		"messages": [
			{
				"id": "wamid.HBgMODg2OTg3MTIyNjY4FQIAERgSMjREQjI0Q0FDQkZCQjU1QjYwAA=="
			}
		]
	}`

	err := json.Unmarshal([]byte(jsonData), resp)

	if err != nil {
		return err
	}

	return nil
}

func TestComponent_ExecuteSendMessageTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{}
	component := Init(bc)

	wantOutput := TaskSendMessageOutput{
		WaID: "886901234567",
		ID:   "wamid.HBgMODg2OTg3MTIyNjY4FQIAERgSMjREQjI0Q0FDQkZCQjU1QjYwAA==",
	}

	testcases := []struct {
		name       string
		task       string
		input      interface{}
		wantOutput TaskSendMessageOutput
	}{
		{
			name: "ok - send text message",
			task: taskSendTextMessage,
			input: TaskSendTextMessageInput{
				PhoneNumberID: "012345678901234",
				To:            "886901234567",
				Body:          "A random message",
				PreviewURL:    "false",
			},
			wantOutput: wantOutput,
		},
		{
			name: "ok - send media message",
			task: taskSendMediaMessage,
			input: TaskSendMediaMessageInput{
				PhoneNumberID: "012345678901234",
				To:            "886901234567",
				MediaType:     "image",
				IDOrLink:      "https://www.example.com/image.jpg",
				Caption:       "A random image",
			},
			wantOutput: wantOutput,
		},
		{
			name: "ok - send location message",
			task: taskSendLocationMessage,
			input: TaskSendLocationMessageInput{
				PhoneNumberID: "012345678901234",
				To:            "886901234567",
				Latitude:      25.123456,
				Longitude:     121.123456,
				LocationName:  "A random location",
				Address:       "A random address",
			},
			wantOutput: wantOutput,
		},
		{
			name: "ok - send contact message",
			task: taskSendContactMessage,
			input: TaskSendContactMessageInput{
				PhoneNumberID:   "012345678901234",
				To:              "886901234567",
				FirstName:       "First",
				LastName:        "Last",
				PhoneNumber:     "886999999999",
				PhoneNumberType: "WORK",
				Email:           "random@gmail.com",
				EmailType:       "WORK",
				Birthdate:       "1990-01-01",
			},
			wantOutput: wantOutput,
		},
		{
			name: "ok - send interactive cta url button message",
			task: taskSendInteractiveCTAURLButtonMessage,
			input: TaskSendInteractiveCTAURLButtonMessageInput{
				PhoneNumberID:     "012345678901234",
				To:                "886901234567",
				HeaderText:        "A random header",
				BodyText:          "A random body",
				FooterText:        "A random footer",
				ButtonDisplayText: "A random button",
				ButtonURL:         "https://www.example.com",
			},
			wantOutput: wantOutput,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {

			setup, err := structpb.NewStruct(map[string]any{
				"token": token,
			})
			c.Assert(err, qt.IsNil)

			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: component, SystemVariables: nil, Setup: setup, Task: tc.task},
				client:             &MockWhatsAppClientSendMessage{},
			}

			switch tc.task {
			case taskSendTextMessage:
				e.execute = e.SendTextMessage
			case taskSendMediaMessage:
				e.execute = e.TaskSendMediaMessage
			case taskSendLocationMessage:
				e.execute = e.TaskSendLocationMessage
			case taskSendContactMessage:
				e.execute = e.TaskSendContactMessage
			case taskSendInteractiveCTAURLButtonMessage:
				e.execute = e.TaskSendInteractiveCTAURLButtonMessage
			}

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
}
