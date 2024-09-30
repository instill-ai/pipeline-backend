package whatsapp

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// this file is used to handle send template message. Send template message will be divided into 4 tasks:
// 1. Send Text-Based Template Message
// 2. Send Media-Based Template Message
// 3. Send Location-Based Template Message
// 4. Send Authentication Template Message

// Documentation API
// send template task: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-message-templates
// Note1: in this documentation API, there is interactive message template, which is not listed in the above supported tasks file. However,  all tasks mentioned above actually supported interactive template message as well. Interactive message template is basically template with buttons, which is supported in all the tasks.
// Note2: Send Catalog Template is not supported yet due to the lack of real phone number to test the API.

// Send Template Message Request, Response and Output.
// Used in all the tasks in this file.

type TaskSendTemplateMessageReq struct {
	MessagingProduct string         `json:"messaging_product"`
	To               string         `json:"to"`
	Type             string         `json:"type"`
	Template         templateObject `json:"template"`
}

type TaskSendTemplateMessageResp struct {
	MessagingProduct string    `json:"messaging_product"`
	Contacts         []contact `json:"contacts"`
	Messages         []message `json:"messages"`
}

type TaskSendTemplateMessageOutput struct {
	WaID          string `json:"recipient-wa-id"`
	ID            string `json:"message-id"`
	MessageStatus string `json:"message-status,omitempty"`
}

// ----------------------- Tasks -----------------------

// Task 1: Send Text-Based Template Message

type TaskSendTextBasedTemplateMessageInput struct {
	PhoneNumberID    string   `json:"phone-number-id"`
	To               string   `json:"to"`
	TemplateName     string   `json:"name"`
	LanguageCode     string   `json:"language-code"`
	HeaderParameters []string `json:"header-parameters"`
	BodyParameters   []string `json:"body-parameters"`
	ButtonParameters []string `json:"button-parameters"`
}

func (e *execution) SendTextBasedTemplateMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendTextBasedTemplateMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendTemplateMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "template",
		Template: templateObject{
			Name: inputStruct.TemplateName,
			Language: languageObject{
				Code: inputStruct.LanguageCode,
			},
		},
	}

	// the text header type can have 0 parameter, so there is no need to have an error message if there is no parameter
	if len(inputStruct.HeaderParameters) != 0 {
		headerComponent := componentObject{
			Type:       "header",
			Parameters: make([]interface{}, len(inputStruct.HeaderParameters)),
		}

		for index, value := range inputStruct.HeaderParameters {
			headerComponent.Parameters[index] = textParameter{
				Type: "text",
				Text: value,
			}
		}
		req.Template.Components = append(req.Template.Components, headerComponent)
	}

	// create a body component if there is any body parameters

	if len(inputStruct.BodyParameters) != 0 {
		bodyComponent := componentObject{
			Type:       "body",
			Parameters: make([]interface{}, len(inputStruct.BodyParameters)),
		}

		for index, value := range inputStruct.BodyParameters {
			bodyComponent.Parameters[index] = textParameter{
				Type: "text",
				Text: value,
			}
		}

		req.Template.Components = append(req.Template.Components, bodyComponent)
	}

	// create button component if there is any
	// one parameter -> one button component

	for _, value := range inputStruct.ButtonParameters {
		splitParam := strings.Split(value, ";")

		if len(splitParam) != 3 {
			return nil, fmt.Errorf("format is wrong, it must be 'button_index;button_type;value_of_the_parameter'. Example: 0;quick_reply;randomvalue")
		}

		if splitParam[1] == "copy_code" && len(splitParam[2]) > 15 {
			return nil, fmt.Errorf("copy code button value cannot be more than 15 characters. It is now %d characters", len(splitParam[2]))
		}

		var param buttonParameter
		if splitParam[1] == "quick_reply" || splitParam[1] == "copy_code" {
			param = buttonParameter{
				Type:    "payload",
				Payload: splitParam[2],
			}

		} else if splitParam[1] == "url" {
			param = buttonParameter{
				Type: "text",
				Text: splitParam[2],
			}

		} else {
			return nil, fmt.Errorf("wrong button_type. button_type is either 'quick_reply', 'url' or 'copy_code'")
		}

		buttonComponent := componentObject{
			Type:          "button",
			ButtonSubType: splitParam[1],
			ButtonIndex:   splitParam[0],
		}

		buttonComponent.Parameters = append(buttonComponent.Parameters, param)

		req.Template.Components = append(req.Template.Components, buttonComponent)
	}

	resp := TaskSendTemplateMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	// only take the first index because we are sending a template to an individual, so there will only be one contact and one message.
	outputStruct := TaskSendTemplateMessageOutput{
		WaID:          resp.Contacts[0].WaID,
		ID:            resp.Messages[0].ID,
		MessageStatus: resp.Messages[0].MessageStatus,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil

}

// Task 2: Send Media-Based Template Message

type TaskSendMediaBasedTemplateMessageInput struct {
	PhoneNumberID    string   `json:"phone-number-id"`
	To               string   `json:"to"`
	TemplateName     string   `json:"name"`
	LanguageCode     string   `json:"language-code"`
	MediaType        string   `json:"media-type"`
	IDOrLink         string   `json:"id-or-link"`
	Filename         string   `json:"filename"` //only for document
	BodyParameters   []string `json:"body-parameters"`
	ButtonParameters []string `json:"button-parameters"`
}

func (e *execution) SendMediaBasedTemplateMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendMediaBasedTemplateMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendTemplateMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "template",
		Template: templateObject{
			Name: inputStruct.TemplateName,
			Language: languageObject{
				Code: inputStruct.LanguageCode,
			},
		},
	}

	// to assign HeaderParameters

	// create a header component

	headerComponent := componentObject{
		Type:       "header",
		Parameters: make([]interface{}, 1),
	}

	switch inputStruct.MediaType {
	case "image":

		if strings.HasPrefix(inputStruct.IDOrLink, "http") {
			headerComponent.Parameters[0] = imageParameter{
				Type: "image",
				Image: mediaObject{
					Link: inputStruct.IDOrLink,
				},
			}
		} else {
			headerComponent.Parameters[0] = imageParameter{
				Type: "image",
				Image: mediaObject{
					ID: inputStruct.IDOrLink,
				},
			}
		}
		req.Template.Components = append(req.Template.Components, headerComponent)

	case "video":

		if strings.HasPrefix(inputStruct.IDOrLink, "http") {
			headerComponent.Parameters[0] = videoParameter{
				Type: "video",
				Video: mediaObject{
					Link: inputStruct.IDOrLink,
				},
			}
		} else {
			headerComponent.Parameters[0] = videoParameter{
				Type: "video",
				Video: mediaObject{
					ID: inputStruct.IDOrLink,
				},
			}
		}
		req.Template.Components = append(req.Template.Components, headerComponent)

	case "document":

		if strings.HasPrefix(inputStruct.IDOrLink, "http") {
			headerComponent.Parameters[0] = documentParameter{
				Type: "document",
				Document: mediaObject{
					Link:     inputStruct.IDOrLink,
					Filename: inputStruct.Filename,
				},
			}
		} else {
			headerComponent.Parameters[0] = documentParameter{
				Type: "document",
				Document: mediaObject{
					ID:       inputStruct.IDOrLink,
					Filename: inputStruct.Filename,
				},
			}
		}
		req.Template.Components = append(req.Template.Components, headerComponent)

	}

	// create a body component if there is any body parameters

	if len(inputStruct.BodyParameters) != 0 {
		bodyComponent := componentObject{
			Type:       "body",
			Parameters: make([]interface{}, len(inputStruct.BodyParameters)),
		}

		for index, value := range inputStruct.BodyParameters {
			bodyComponent.Parameters[index] = textParameter{
				Type: "text",
				Text: value,
			}
		}

		req.Template.Components = append(req.Template.Components, bodyComponent)
	}

	// create button component if there is any
	// one parameter -> one button component

	for _, value := range inputStruct.ButtonParameters {
		splitParam := strings.Split(value, ";")

		if len(splitParam) != 3 {
			return nil, fmt.Errorf("format is wrong, it must be 'button_index;button_type;value_of_the_parameter'. Example: 0;quick_reply;randomvalue")
		}

		if splitParam[1] == "copy_code" && len(splitParam[2]) > 15 {
			return nil, fmt.Errorf("copy code button value cannot be more than 15 characters. It is now %d characters", len(splitParam[2]))
		}

		var param buttonParameter
		if splitParam[1] == "quick_reply" || splitParam[1] == "copy_code" {
			param = buttonParameter{
				Type:    "payload",
				Payload: splitParam[2],
			}

		} else if splitParam[1] == "url" {
			param = buttonParameter{
				Type: "text",
				Text: splitParam[2],
			}

		} else {
			return nil, fmt.Errorf("wrong button_type. button_type is either 'quick_reply', 'url' or 'copy_code'")
		}

		buttonComponent := componentObject{
			Type:          "button",
			ButtonSubType: splitParam[1],
			ButtonIndex:   splitParam[0],
		}

		buttonComponent.Parameters = append(buttonComponent.Parameters, param)

		req.Template.Components = append(req.Template.Components, buttonComponent)
	}

	resp := TaskSendTemplateMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	// only take the first index because we are sending a template to an individual, so there will only be one contact and one message.
	outputStruct := TaskSendTemplateMessageOutput{
		WaID:          resp.Contacts[0].WaID,
		ID:            resp.Messages[0].ID,
		MessageStatus: resp.Messages[0].MessageStatus,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 3: Send Location-Based Template Message

type TaskSendLocationBasedTemplateMessageInput struct {
	PhoneNumberID    string   `json:"phone-number-id"`
	To               string   `json:"to"`
	TemplateName     string   `json:"name"`
	LanguageCode     string   `json:"language-code"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	LocationName     string   `json:"location-name"`
	Address          string   `json:"address"`
	BodyParameters   []string `json:"body-parameters"`
	ButtonParameters []string `json:"button-parameters"`
}

func (e *execution) SendLocationBasedTemplateMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendLocationBasedTemplateMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendTemplateMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "template",
		Template: templateObject{
			Name: inputStruct.TemplateName,
			Language: languageObject{
				Code: inputStruct.LanguageCode,
			},
		},
	}

	// to assign HeaderParameters

	// create a header component

	headerComponent := componentObject{
		Type:       "header",
		Parameters: make([]interface{}, 1),
	}

	headerComponent.Parameters[0] =
		locationParameter{
			Type: "location",
			Location: locationObject{
				Latitude:  fmt.Sprintf("%f", inputStruct.Latitude),
				Longitude: fmt.Sprintf("%f", inputStruct.Longitude),
				Name:      inputStruct.LocationName,
				Address:   inputStruct.Address,
			},
		}

	req.Template.Components = append(req.Template.Components, headerComponent)

	// create a body component if there is any body parameters

	if len(inputStruct.BodyParameters) != 0 {
		bodyComponent := componentObject{
			Type:       "body",
			Parameters: make([]interface{}, len(inputStruct.BodyParameters)),
		}

		for index, value := range inputStruct.BodyParameters {
			bodyComponent.Parameters[index] = textParameter{
				Type: "text",
				Text: value,
			}
		}

		req.Template.Components = append(req.Template.Components, bodyComponent)
	}

	// create button component if there is any
	// one parameter -> one button component

	for _, value := range inputStruct.ButtonParameters {
		splitParam := strings.Split(value, ";")

		if len(splitParam) != 3 {
			return nil, fmt.Errorf("format is wrong, it must be 'button_index;button_type;value_of_the_parameter'. Example: 0;quick_reply;randomvalue")
		}

		if splitParam[1] == "copy_code" && len(splitParam[2]) > 15 {
			return nil, fmt.Errorf("copy code button value cannot be more than 15 characters. It is now %d characters", len(splitParam[2]))
		}

		var param buttonParameter
		if splitParam[1] == "quick_reply" || splitParam[1] == "copy_code" {
			param = buttonParameter{
				Type:    "payload",
				Payload: splitParam[2],
			}

		} else if splitParam[1] == "url" {
			param = buttonParameter{
				Type: "text",
				Text: splitParam[2],
			}

		} else {
			return nil, fmt.Errorf("wrong button_type. button_type is either 'quick_reply', 'url' or 'copy_code'")
		}

		buttonComponent := componentObject{
			Type:          "button",
			ButtonSubType: splitParam[1],
			ButtonIndex:   splitParam[0],
		}

		buttonComponent.Parameters = append(buttonComponent.Parameters, param)

		req.Template.Components = append(req.Template.Components, buttonComponent)
	}

	resp := TaskSendTemplateMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	// only take the first index because we are sending a template to an individual, so there will only be one contact and one message.
	outputStruct := TaskSendTemplateMessageOutput{
		WaID:          resp.Contacts[0].WaID,
		ID:            resp.Messages[0].ID,
		MessageStatus: resp.Messages[0].MessageStatus,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 4: Send Authentication Template Message

type TaskSendAuthenticationTemplateMessageInput struct {
	PhoneNumberID   string `json:"phone-number-id"`
	To              string `json:"to"`
	TemplateName    string `json:"name"`
	LanguageCode    string `json:"language-code"`
	OneTimePassword string `json:"one-time-password"`
}

func (e *execution) SendAuthenticationTemplateMessage(in *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskSendAuthenticationTemplateMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	if len(inputStruct.OneTimePassword) > 15 {
		return nil, fmt.Errorf("one-time password characters cannot be more than 15. It is now %d characters", len(inputStruct.OneTimePassword))
	}

	req := TaskSendTemplateMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "template",
		Template: templateObject{
			Name: inputStruct.TemplateName,
			Language: languageObject{
				Code: inputStruct.LanguageCode,
			},
		},
	}

	// authentication template has one body parameter, the one-time password

	bodyComponent := componentObject{
		Type: "body",
		Parameters: []interface{}{
			textParameter{
				Type: "text",
				Text: inputStruct.OneTimePassword,
			},
		},
	}

	req.Template.Components = append(req.Template.Components, bodyComponent)

	// authentication template has one button, used to copy the code

	buttonComponent := componentObject{
		Type:          "button",
		ButtonSubType: "url",
		ButtonIndex:   "0",
		Parameters: []interface{}{
			buttonParameter{
				Type: "text",
				Text: inputStruct.OneTimePassword,
			},
		},
	}

	req.Template.Components = append(req.Template.Components, buttonComponent)

	resp := TaskSendTemplateMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	// only take the first index because we are sending a template to an individual, so there will only be one contact and one message.
	outputStruct := TaskSendTemplateMessageOutput{
		WaID:          resp.Contacts[0].WaID,
		ID:            resp.Messages[0].ID,
		MessageStatus: resp.Messages[0].MessageStatus,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
