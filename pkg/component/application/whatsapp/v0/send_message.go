package whatsapp

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// this file is used to handle 6 send tasks

// Tasks:
// 1. Send Text Message
// 2. Send Media Message
// 3. Send Location Message
// 4. Send Contact Message
// 5. Send Interactive CTA URL Button Message

// types of messages: https://developers.facebook.com/docs/whatsapp/cloud-api/guides/send-messages

// Objects (structs that are part of request struct; can be used in more than one task)

type languageObject struct {
	Code string `json:"code"`
}

type textObject struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url"`
}

type locationObject struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	Name      string `json:"name"`
	Address   string `json:"address"`
}

type interactiveObject struct {
	Type   string                `json:"type"`
	Header interface{}           `json:"header,omitempty"`
	Body   interactiveObjectBody `json:"body"`
	Footer *footerObject         `json:"footer,omitempty"`
	Action actionObject          `json:"action"`
}

type interactiveObjectBody struct {
	Text string `json:"text"`
}

type footerObject struct {
	Text string `json:"text"`
}

type actionObject struct {
	Name      string                `json:"name,omitempty"`
	Parameter actionObjectParameter `json:"parameters"`
}

type actionObjectParameter struct {
	DisplayText string `json:"display_text"`
	URL         string `json:"url"`
}

type mediaObject struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`     // if id is used, no need link.
	Filename string `json:"filename,omitempty"` // only for document. This will specify the format of the file displayed in WhatsApp
	Caption  string `json:"caption,omitempty"`  // cannot be used in template message
}

type templateObject struct {
	Name       string            `json:"name"`
	Language   languageObject    `json:"language"`
	Components []componentObject `json:"components,omitempty"`
}

// Component type is either header, body or button (footer doesn't have any parameter)
// Note:
// header can have various parameters: text, image, location, document and video
// body support text parameters
// button type is quick_reply, url and copy_code.

type componentObject struct {
	Type          string        `json:"type"`
	Parameters    []interface{} `json:"parameters"`
	ButtonSubType string        `json:"sub_type,omitempty"` // only used when the type is button. Type of button
	ButtonIndex   string        `json:"index,omitempty"`    // only used when the type is button. Refers to the position index of the button
}

type contactObject struct {
	Name     nameObject    `json:"name"`
	Phones   []phoneObject `json:"phones,omitempty"`
	Emails   []emailObject `json:"emails,omitempty"`
	Birthday string        `json:"birthday,omitempty"`
}

type nameObject struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name"`
	MiddleName    string `json:"middle_name,omitempty"`
	LastName      string `json:"last_name,omitempty"`
}

type emailObject struct {
	Email string `json:"email,omitempty"`
	Type  string `json:"type,omitempty"`
}

type phoneObject struct {
	Phone string `json:"phone,omitempty"`
	Type  string `json:"type,omitempty"`
}

// Header Parameters (can be used in componentObject and interactiveObject)

// used when the header type is text (also used for body)
type textParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// used when the header type is image
type imageParameter struct {
	Type  string      `json:"type"`
	Image mediaObject `json:"image"`
}

// used when the header type is video
type videoParameter struct {
	Type  string      `json:"type"`
	Video mediaObject `json:"video"`
}

// used when the header type is document
type documentParameter struct {
	Type     string      `json:"type"`
	Document mediaObject `json:"document"`
}

// used when the header type is location
type locationParameter struct {
	Type     string         `json:"type"`
	Location locationObject `json:"location"`
}

// used for button component
type buttonParameter struct {
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
	Text    string `json:"text,omitempty"`
}

// structs that are part of API response (used in all tasks)

type contact struct {
	Input string `json:"input"`
	WaID  string `json:"wa_id"`
}

type message struct {
	ID            string `json:"id"`
	MessageStatus string `json:"message_status,omitempty"`
}

// Send Message Response and Output.
// Used in all the tasks in this file.

type TaskSendMessageResp struct {
	MessagingProduct string    `json:"messaging_product"`
	Contacts         []contact `json:"contacts"`
	Messages         []message `json:"messages"`
}

// No message status in normal send message tasks
type TaskSendMessageOutput struct {
	WaID string `json:"recipient-wa-id"`
	ID   string `json:"message-id"`
}

// ----------------------- Tasks -----------------------

// Task 1: Send Text Message Task

type TaskSendTextMessageInput struct {
	PhoneNumberID string `json:"phone-number-id"`
	To            string `json:"to"`
	Body          string `json:"body"`
	PreviewURL    string `json:"preview-url"`
}

type TaskSendTextMessageReq struct {
	MessagingProduct string     `json:"messaging_product"`
	To               string     `json:"to"`
	Type             string     `json:"type"`
	Text             textObject `json:"text"`
}

func (e *execution) SendTextMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendTextMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendTextMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "text",
		Text: textObject{
			Body:       inputStruct.Body,
			PreviewURL: inputStruct.PreviewURL == "true",
		},
	}

	resp := TaskSendMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskSendMessageOutput{
		WaID: resp.Contacts[0].WaID,
		ID:   resp.Messages[0].ID,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 2: Send Media Message Task

type TaskSendMediaMessageInput struct {
	PhoneNumberID string `json:"phone-number-id"`
	To            string `json:"to"`
	MediaType     string `json:"media-type"`
	IDOrLink      string `json:"id-or-link"`
	Caption       string `json:"caption"`  //cannot be used in audio
	Filename      string `json:"filename"` //only for document

}

type TaskSendMediaMessageReq struct {
	MessagingProduct string       `json:"messaging_product"`
	To               string       `json:"to"`
	Type             string       `json:"type"`
	Document         *mediaObject `json:"document,omitempty,"`
	Audio            *mediaObject `json:"audio,omitempty"`
	Image            *mediaObject `json:"image,omitempty"`
	Video            *mediaObject `json:"video,omitempty"`
}

func (e *execution) TaskSendMediaMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendMediaMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendMediaMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
	}

	var id string
	var link string
	if strings.HasPrefix(inputStruct.IDOrLink, "http") {
		link = inputStruct.IDOrLink
	} else {
		id = inputStruct.IDOrLink
	}

	switch inputStruct.MediaType {
	case "document":
		req.Type = "document"
		req.Document = &mediaObject{
			ID:       id,
			Link:     link,
			Caption:  inputStruct.Caption,
			Filename: inputStruct.Filename,
		}
	case "audio":
		req.Type = "audio"
		req.Audio = &mediaObject{
			ID:   id,
			Link: link,
		}
	case "image":
		req.Type = "image"
		req.Image = &mediaObject{
			ID:      id,
			Link:    link,
			Caption: inputStruct.Caption,
		}
	case "video":
		req.Type = "video"
		req.Video = &mediaObject{
			ID:      id,
			Link:    link,
			Caption: inputStruct.Caption,
		}
	default:
		return nil, fmt.Errorf("unsupported media type")
	}

	resp := TaskSendMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskSendMessageOutput{
		WaID: resp.Contacts[0].WaID,
		ID:   resp.Messages[0].ID,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 3: Send Location Message Task
type TaskSendLocationMessageInput struct {
	PhoneNumberID string  `json:"phone-number-id"`
	To            string  `json:"to"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
	LocationName  string  `json:"location-name"`
	Address       string  `json:"address"`
}

type TaskSendLocationMessageReq struct {
	MessagingProduct string         `json:"messaging_product"`
	To               string         `json:"to"`
	Type             string         `json:"type"`
	Location         locationObject `json:"location"`
}

func (e *execution) TaskSendLocationMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendLocationMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendLocationMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "location",
		Location: locationObject{
			Latitude:  fmt.Sprintf("%f", inputStruct.Latitude),
			Longitude: fmt.Sprintf("%f", inputStruct.Longitude),
			Name:      inputStruct.LocationName,
			Address:   inputStruct.Address,
		},
	}

	resp := TaskSendMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskSendMessageOutput{
		WaID: resp.Contacts[0].WaID,
		ID:   resp.Messages[0].ID,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Task 4: Send Contact Message Task
type TaskSendContactMessageInput struct {
	PhoneNumberID   string `json:"phone-number-id"`
	To              string `json:"to"`
	FirstName       string `json:"first-name"`
	MiddleName      string `json:"middle-name"`
	LastName        string `json:"last-name"`
	PhoneNumber     string `json:"phone-number"`
	PhoneNumberType string `json:"phone-number-type"`
	Email           string `json:"email"`
	EmailType       string `json:"email-type"`
	Birthdate       string `json:"birthday"`
}

type TaskSendContactMessageReq struct {
	MessagingProduct string          `json:"messaging_product"`
	To               string          `json:"to"`
	Type             string          `json:"type"`
	Contacts         []contactObject `json:"contacts"`
}

func (e *execution) TaskSendContactMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendContactMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	var formattedName string
	if inputStruct.FirstName != "" {
		formattedName += inputStruct.FirstName
	}

	if inputStruct.MiddleName != "" {
		formattedName += " " + inputStruct.MiddleName
	}

	if inputStruct.LastName != "" {
		formattedName += " " + inputStruct.LastName
	}

	contact := contactObject{
		Name: nameObject{
			FormattedName: formattedName,
			FirstName:     inputStruct.FirstName,
			MiddleName:    inputStruct.MiddleName,
			LastName:      inputStruct.LastName,
		},
		Emails: []emailObject{
			{
				Email: inputStruct.Email,
				Type:  inputStruct.EmailType,
			},
		},
		Birthday: inputStruct.Birthdate,
	}

	req := TaskSendContactMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "contacts",
		Contacts:         []contactObject{contact},
	}

	if inputStruct.PhoneNumber != "" {
		if inputStruct.PhoneNumberType == "none" {
			return nil, fmt.Errorf("please remember to set the phone number type when you input the phone number")
		}

		req.Contacts[0].Phones = append(req.Contacts[0].Phones, phoneObject{
			Phone: inputStruct.PhoneNumber,
			Type:  inputStruct.PhoneNumberType,
		})

	}

	if inputStruct.Email != "" {
		if inputStruct.EmailType == "none" {
			return nil, fmt.Errorf("please remember to set the email type when you input the email")
		}

		req.Contacts[0].Emails = append(req.Contacts[0].Emails, emailObject{
			Email: inputStruct.Email,
			Type:  inputStruct.EmailType,
		})
	}

	resp := TaskSendMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskSendMessageOutput{
		WaID: resp.Contacts[0].WaID,
		ID:   resp.Messages[0].ID,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil

}

// Task 5: Send Interactive CTA URL Button Message Task

type TaskSendInteractiveCTAURLButtonMessageInput struct {
	PhoneNumberID     string `json:"phone-number-id"`
	To                string `json:"to"`
	HeaderText        string `json:"header-text"`
	BodyText          string `json:"body-text"`
	FooterText        string `json:"footer-text"`
	ButtonDisplayText string `json:"button-display-text"`
	ButtonURL         string `json:"button-url"`
}

type TaskSendInteractiveCTAURLButtonMessageReq struct {
	MessagingProduct string            `json:"messaging_product"`
	To               string            `json:"to"`
	Type             string            `json:"type"`
	Interactive      interactiveObject `json:"interactive"`
}

func (e *execution) TaskSendInteractiveCTAURLButtonMessage(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskSendInteractiveCTAURLButtonMessageInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	req := TaskSendInteractiveCTAURLButtonMessageReq{
		MessagingProduct: "whatsapp",
		To:               inputStruct.To,
		Type:             "interactive",
		Interactive: interactiveObject{
			Type: "cta_url",
			Body: interactiveObjectBody{
				Text: inputStruct.BodyText,
			},
			Action: actionObject{
				Name: "cta_url",
				Parameter: actionObjectParameter{
					DisplayText: inputStruct.ButtonDisplayText,
					URL:         inputStruct.ButtonURL,
				},
			},
		},
	}

	if inputStruct.HeaderText != "" {
		req.Interactive.Header = textParameter{
			Type: "text",
			Text: inputStruct.HeaderText,
		}
	}

	if inputStruct.FooterText != "" {
		req.Interactive.Footer = &footerObject{
			Text: inputStruct.FooterText,
		}
	}

	resp := TaskSendMessageResp{}
	err = e.client.SendMessageAPI(&req, &resp, inputStruct.PhoneNumberID)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskSendMessageOutput{
		WaID: resp.Contacts[0].WaID,
		ID:   resp.Messages[0].ID,
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
