package hubspot

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	hubspot "github.com/belong-inc/go-hubspot"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// following go-hubspot sdk format
// Note: The conversation API is still in BETA, and hence, any of these struct can change in the future.

// API functions for Thread

type ThreadService interface {
	Get(threadID string, param string) (*TaskGetThreadResp, error)
	Insert(threadID string, message *TaskInsertMessageReq) (*TaskInsertMessageResp, error)
}

type ThreadServiceOp struct {
	threadPath string
	client     *hubspot.Client
}

func (s *ThreadServiceOp) Get(threadID string, param string) (*TaskGetThreadResp, error) {
	resource := &TaskGetThreadResp{}
	if err := s.client.Get(s.threadPath+"/"+threadID+"/messages"+param, resource, nil); err != nil {
		return nil, err
	}
	return resource, nil
}

func (s *ThreadServiceOp) Insert(threadID string, message *TaskInsertMessageReq) (*TaskInsertMessageResp, error) {
	resource := &TaskInsertMessageResp{}
	if err := s.client.Post(s.threadPath+"/"+threadID+"/messages", message, resource); err != nil {
		return nil, err
	}
	return resource, nil
}

// Get Thread

// Get Thread Input

type TaskGetThreadInput struct {
	ThreadID string `json:"thread-id"`
}

// Get Thread Reponse structs

type TaskGetThreadResp struct {
	Results []taskGetThreadRespResult `json:"results"`
	Paging  *taskGetThreadRespPaging  `json:"paging,omitempty"`
}

type taskGetThreadRespResult struct {
	CreatedAt        string                  `json:"createdAt"`
	Senders          []taskGetThreadRespUser `json:"senders,omitempty"`
	Recipients       []taskGetThreadRespUser `json:"recipients,omitempty"`
	Text             string                  `json:"text,omitempty"`
	Subject          string                  `json:"subject,omitempty"`
	ChannelID        string                  `json:"channelId,omitempty"`
	ChannelAccountID string                  `json:"channelAccountId,omitempty"`
	Type             string                  `json:"type,omitempty"`
}

type taskGetThreadRespPaging struct {
	Next struct {
		Link  string `json:"link"`
		After string `json:"after"`
	} `json:"next"`
}

type taskGetThreadRespUser struct {
	Name               string                      `json:"name,omitempty"`
	DeliveryIdentifier taskGetThreadRespIdentifier `json:"deliveryIdentifier,omitempty"`
	ActorID            string                      `json:"actorId,omitempty"` //only applicable to sender
}

type taskGetThreadRespIdentifier struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

// Get Thread Output structs

type TaskGetThreadOutput struct {
	Results      []taskGetThreadOutputResult `json:"results"`
	NoOfMessages int                         `json:"no-of-messages"`
}

type taskGetThreadOutputResult struct {
	CreatedAt        string                         `json:"created-at"`
	Sender           taskGetThreadOutputSender      `json:"sender,omitempty"`
	Recipients       []taskGetThreadOutputRecipient `json:"recipients,omitempty"`
	Text             string                         `json:"text"`
	Subject          string                         `json:"subject,omitempty"`
	ChannelID        string                         `json:"channel-id"`
	ChannelAccountID string                         `json:"channel-account-id"`
}

// It is named as sender-x so that it is clearer for the user that it is referring to the sender's information.

type taskGetThreadOutputSender struct {
	Name    string `json:"sender-name,omitempty"`
	Type    string `json:"sender-type,omitempty"`
	Value   string `json:"sender-value,omitempty"`
	ActorID string `json:"sender-actor-id"`
}

type taskGetThreadOutputRecipient struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	ActorID string `json:"actor-id"`
}

func (e *execution) GetThread(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := TaskGetThreadInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	var param string
	outputStruct := TaskGetThreadOutput{}
	for {
		res, err := e.client.Thread.Get(inputStruct.ThreadID, param)

		if err != nil {
			return nil, err
		}

		if len(res.Results) == 0 {
			break
		}

		// convert to output struct

		for _, value1 := range res.Results {
			// this way, the output will only contain the actual messages in the thread (ignore system message from hubspot)
			if value1.Type != "MESSAGE" {
				continue
			}

			resultOutput := taskGetThreadOutputResult{
				CreatedAt:        value1.CreatedAt,
				Text:             value1.Text,
				Subject:          value1.Subject,
				ChannelID:        value1.ChannelID,
				ChannelAccountID: value1.ChannelAccountID,
			}

			// there should only be one sender
			// sender
			if len(value1.Senders) > 0 {
				value2 := value1.Senders[0]
				userSenderOutput := taskGetThreadOutputSender{
					Name:    value2.Name,
					Type:    value2.DeliveryIdentifier.Type,
					Value:   value2.DeliveryIdentifier.Value,
					ActorID: value2.ActorID,
				}
				resultOutput.Sender = userSenderOutput
			}

			// recipient
			for _, value3 := range value1.Recipients {
				userRecipientOutput := taskGetThreadOutputRecipient{
					Name:  value3.Name,
					Type:  value3.DeliveryIdentifier.Type,
					Value: value3.DeliveryIdentifier.Value,
				}

				resultOutput.Recipients = append(resultOutput.Recipients, userRecipientOutput)

			}

			outputStruct.Results = append(outputStruct.Results, resultOutput)

		}

		// if there is no more messages/ page to be read, break
		if res.Paging == nil || len(res.Results) == 0 {
			break
		} else {
			param = fmt.Sprintf("?after=%s", res.Paging.Next.After)
		}

	}

	outputStruct.NoOfMessages = len(outputStruct.Results)
	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}

// Insert Message

// Input

type TaskInsertMessageInput struct {
	ThreadID         string   `json:"thread-id"`
	SenderActorID    string   `json:"sender-actor-id"`
	Recipients       []string `json:"recipients"`
	ChannelAccountID string   `json:"channel-account-id"`
	Subject          string   `json:"subject"`
	Text             string   `json:"text"`
}

// Request

type TaskInsertMessageReq struct {
	Type             string                          `json:"type"`
	Text             string                          `json:"text"` //content of the message
	Recipients       []taskInsertMessageReqRecipient `json:"recipients"`
	SenderActorID    string                          `json:"senderActorId"`
	ChannelID        string                          `json:"channelId"`
	ChannelAccountID string                          `json:"channelAccountId"`
	Subject          string                          `json:"subject"`
}

type taskInsertMessageReqRecipient struct {
	RecipientField     string                         `json:"recipientField"`
	DeliveryIdentifier taskInsertMessageReqIdentifier `json:"deliveryIdentifier"`
}

type taskInsertMessageReqIdentifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Response

type TaskInsertMessageResp struct {
	Status  taskInsertMessageRespStatusType `json:"status"`
	Message string                          `json:"message,omitempty"`
}

type taskInsertMessageRespStatusType struct {
	StatusType string `json:"statusType"`
}

// Output

type TaskInsertMessageOutput struct {
	Status string `json:"status"`
}

func (e *execution) InsertMessage(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := TaskInsertMessageInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %v", err)
	}

	recipients := make([]taskInsertMessageReqRecipient, len(inputStruct.Recipients))
	for index, value := range inputStruct.Recipients {
		recipients[index] = taskInsertMessageReqRecipient{
			RecipientField: "TO",
			DeliveryIdentifier: taskInsertMessageReqIdentifier{
				Type:  "HS_EMAIL_ADDRESS",
				Value: value,
			},
		}
	}

	req := TaskInsertMessageReq{
		Type:             "MESSAGE",
		Text:             inputStruct.Text,
		Recipients:       recipients,
		SenderActorID:    inputStruct.SenderActorID,
		ChannelID:        "1002", //1002 is for email
		ChannelAccountID: inputStruct.ChannelAccountID,
		Subject:          inputStruct.Subject,
	}

	res, err := e.client.Thread.Insert(inputStruct.ThreadID, &req)

	if err != nil {
		return nil, err
	}

	outputStruct := TaskInsertMessageOutput{
		Status: res.Status.StatusType,
	}

	if outputStruct.Status != "SENT" {
		return nil, fmt.Errorf("error sending message")
	}

	output, err := base.ConvertToStructpb(outputStruct)

	if err != nil {
		return nil, fmt.Errorf("failed to convert output to struct: %v", err)
	}

	return output, nil
}
