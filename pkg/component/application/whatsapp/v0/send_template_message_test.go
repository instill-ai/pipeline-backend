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

const (
	token = "test_token"
)

type MockWhatsappClientSendTemplate struct{}

func (c *MockWhatsappClientSendTemplate) SendMessageAPI(req interface{}, resp interface{}, PhoneNumberID string) error {
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
				"id": "wamid.HBgMODg2OTg3MTIyNjY4FQIAERgSMjREQjI0Q0FDQkZCQjU1QjYwAA==",
				"message_status": "accepted"
			}
		]
	}`

	err := json.Unmarshal([]byte(jsonData), resp)
	if err != nil {
		return err
	}

	return nil
}

func TestComponent_ExecuteSendTemplateMessageTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{}
	component := Init(bc)

	wantOutput := TaskSendTemplateMessageOutput{
		WaID:          "886901234567",
		ID:            "wamid.HBgMODg2OTg3MTIyNjY4FQIAERgSMjREQjI0Q0FDQkZCQjU1QjYwAA==",
		MessageStatus: "accepted",
	}

	testcases := []struct {
		name       string
		task       string
		input      interface{}
		wantOutput TaskSendTemplateMessageOutput
		wantErr    string
	}{
		{
			name: "ok - send text-based template message",
			task: taskSendTextBasedTemplateMessage,
			input: TaskSendTextBasedTemplateMessageInput{
				PhoneNumberID:    "012345678901234",
				To:               "886901234567",
				TemplateName:     "random_text_template",
				LanguageCode:     "en_us",
				HeaderParameters: []string{"headerparameter1, headerparameter2"},
				BodyParameters:   []string{"bodyparameter1, bodyparameter2"},
				ButtonParameters: []string{"0;copy_code;randomvalue"},
			},
			wantOutput: wantOutput,
			wantErr:    "",
		},
		{
			name: "nok - send text-based template message: button format is incorrect",
			task: taskSendTextBasedTemplateMessage,
			input: TaskSendTextBasedTemplateMessageInput{
				PhoneNumberID:    "012345678901234",
				To:               "886901234567",
				TemplateName:     "random_text_template",
				LanguageCode:     "en_us",
				HeaderParameters: []string{"headerparameter1, headerparameter2"},
				BodyParameters:   []string{"bodyparameter1, bodyparameter2"},
				ButtonParameters: []string{"0;randomvalue"},
			},
			wantOutput: wantOutput,
			wantErr:    "format is wrong, it must be 'button_index;button_type;value_of_the_parameter'. Example: 0;quick_reply;randomvalue",
		},
		{
			name: "ok - send media-based template message",
			task: taskSendMediaBasedTemplateMessage,
			input: TaskSendMediaBasedTemplateMessageInput{
				PhoneNumberID:    "012345678901234",
				To:               "886901234567",
				TemplateName:     "random_document_template",
				LanguageCode:     "en_us",
				MediaType:        "document",
				IDOrLink:         "https://www.random.com/random.pdf",
				Filename:         "random.pdf",
				BodyParameters:   []string{"bodyparameter1, bodyparameter2"},
				ButtonParameters: []string{"0;url;websiteurl"},
			},
			wantOutput: wantOutput,
			wantErr:    "",
		},
		{
			name: "ok - send location-based template message",
			task: taskSendLocationBasedTemplateMessage,
			input: TaskSendLocationBasedTemplateMessageInput{
				PhoneNumberID:    "012345678901234",
				To:               "886901234567",
				TemplateName:     "random_location_template",
				LanguageCode:     "en_us",
				Latitude:         25.123456,
				Longitude:        121.123456,
				LocationName:     "A random location",
				Address:          "A random address",
				BodyParameters:   []string{"bodyparameter1, bodyparameter2"},
				ButtonParameters: []string{"0;quick_reply;randompayload"},
			},
			wantOutput: wantOutput,
			wantErr:    "",
		},
		{
			name: "ok - send authentication template message",
			task: taskSendAuthenticationTemplateMessage,
			input: TaskSendAuthenticationTemplateMessageInput{
				PhoneNumberID:   "012345678901234",
				To:              "886901234567",
				TemplateName:    "random_authentication_template",
				LanguageCode:    "en_us",
				OneTimePassword: "a12345",
			},
			wantOutput: wantOutput,
			wantErr:    "",
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
				client:             &MockWhatsappClientSendTemplate{},
			}

			switch tc.task {
			case taskSendTextBasedTemplateMessage:
				e.execute = e.SendTextBasedTemplateMessage
			case taskSendMediaBasedTemplateMessage:
				e.execute = e.SendMediaBasedTemplateMessage
			case taskSendLocationBasedTemplateMessage:
				e.execute = e.SendLocationBasedTemplateMessage
			case taskSendAuthenticationTemplateMessage:
				e.execute = e.SendAuthenticationTemplateMessage
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
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}
