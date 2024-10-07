package instillapp

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	appPB "github.com/instill-ai/protogen-go/app/app/v1alpha"
)

// ReadChatHistoryInput is the input struct for the ReadChatHistory operation
type ReadChatHistoryInput struct {
	Namespace       string `json:"namespace"`
	AppID           string `json:"app-id"`
	ConversationID  string `json:"conversation-id"`
	Role            string `json:"role"`
	MessageType     string `json:"message-type"`
	Duration        string `json:"duration"`
	MaxMessageCount int    `json:"max-message-count"`
}

// ReadChatHistoryOutput is the output struct for the ReadChatHistory operation
type ReadChatHistoryOutput struct {
	Messages []Message `json:"messages"`
}

// Message is the struct for a message
type Message struct {
	Content []Content `json:"content"`
	// Role can be either 'user' or 'assistant'
	Role string `json:"role"`
	Name string `json:"name,omitempty"`
}

// Content is the struct for the content of a message.
// It can be either text, image_url, or image_base64.
// Only one of the fields should be set, and Type should be set to the type of the content.
type Content struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	ImageURL    string `json:"image-url,omitempty"`
	ImageBase64 string `json:"image-base64,omitempty"`
}

func (in *ReadChatHistoryInput) validate() error {
	if in.Role != "" && in.Role != "user" && in.Role != "assistant" {
		return fmt.Errorf("role must be either 'user' or 'assistant'")
	}

	if in.MessageType != "" && in.MessageType != "MESSAGE_TYPE_TEXT" {
		return fmt.Errorf("message-type must be 'MESSAGE_TYPE_TEXT'")
	}

	if in.Duration != "" {
		_, err := time.ParseDuration(in.Duration)
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
	}
	return nil
}

func (out *ReadChatHistoryOutput) filter(inputStruct ReadChatHistoryInput, messages []*appPB.Message) {
	for _, message := range messages {

		if inputStruct.Role != "" && inputStruct.Role != message.Role {
			continue
		}

		if inputStruct.MessageType != "" && inputStruct.MessageType != message.Type.String() {
			continue
		}

		if inputStruct.Duration != "" {
			duration, _ := time.ParseDuration(inputStruct.Duration)
			if time.Since(message.CreateTime.AsTime()) > duration {
				continue
			}
		}
		if inputStruct.MaxMessageCount > 0 && len(out.Messages) >= inputStruct.MaxMessageCount {
			break
		}

		content := []Content{
			{
				Type: convertPBTypeToJSONType(message.Type.String()),
				Text: message.Content,
			},
		}

		out.Messages = append(out.Messages, Message{
			Content: content,
			Role:    message.Role,
		})
	}
}

func convertPBTypeToJSONType(pbType string) string {
	switch pbType {
	case "MESSAGE_TYPE_TEXT":
		return "text"
	default:
		return "unknown"
	}
}

func (e *execution) readChatHistory(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := ReadChatHistoryInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input struct: %w", err)
	}

	err = inputStruct.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	appClient, connection := e.client, e.connection
	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	res, err := appClient.ListMessages(ctx, &appPB.ListMessagesRequest{
		NamespaceId:           inputStruct.Namespace,
		AppId:                 inputStruct.AppID,
		ConversationId:        inputStruct.ConversationID,
		IncludeSystemMessages: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}

	output := ReadChatHistoryOutput{
		Messages: make([]Message, 0),
	}

	output.filter(inputStruct, res.Messages)

	for res.NextPageToken != "" || (len(output.Messages) < inputStruct.MaxMessageCount && inputStruct.MaxMessageCount > 0) {
		res, err = appClient.ListMessages(ctx, &appPB.ListMessagesRequest{
			NamespaceId:           inputStruct.Namespace,
			AppId:                 inputStruct.AppID,
			ConversationId:        inputStruct.ConversationID,
			IncludeSystemMessages: true,
			PageToken:             res.NextPageToken,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to list messages: %w", err)
		}

		output.filter(inputStruct, res.Messages)
	}

	return base.ConvertToStructpb(output)

}

type WriteChatMessageInput struct {
	Namespace      string       `json:"namespace"`
	AppID          string       `json:"app-id"`
	ConversationID string       `json:"conversation-id"`
	Message        WriteMessage `json:"message"`
}

type WriteMessage struct {
	Content string `json:"content"`
	Role    string `json:"role,omitempty"`
}

type WriteChatMessageOutput struct {
	MessageUID string `json:"message-uid"`
	CreateTime string `json:"create-time"`
	UpdateTime string `json:"update-time"`
}

func (in *WriteChatMessageInput) validate() error {
	role := in.Message.Role
	if role != "" && role != "user" && role != "assistant" {
		return fmt.Errorf("role must be either 'user' or 'assistant'")
	}
	return nil
}

func (e *execution) writeChatMessage(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := WriteChatMessageInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input struct: %w", err)
	}

	err = inputStruct.validate()

	if err != nil {
		return nil, fmt.Errorf("invalid input: %w", err)
	}

	appClient, connection := e.client, e.connection
	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	apps, err := appClient.ListApps(ctx, &appPB.ListAppsRequest{
		NamespaceId: inputStruct.Namespace,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list apps: %w", err)
	}

	found := false

	for _, app := range apps.Apps {
		if app.AppId == inputStruct.AppID {
			found = true
			break
		}
	}

	if !found {
		_, err = appClient.CreateApp(ctx, &appPB.CreateAppRequest{
			NamespaceId: inputStruct.Namespace,
			Id:          inputStruct.AppID,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create app: %w", err)
		}
	}

	conversations, err := appClient.ListConversations(ctx, &appPB.ListConversationsRequest{
		NamespaceId: inputStruct.Namespace,
		AppId:       inputStruct.AppID,
		IfAll:       true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}

	found = false

	for _, conversation := range conversations.Conversations {
		if conversation.Id == inputStruct.ConversationID {
			found = true
			break
		}
	}

	if !found {
		_, err = appClient.CreateConversation(ctx, &appPB.CreateConversationRequest{
			NamespaceId:    inputStruct.Namespace,
			AppId:          inputStruct.AppID,
			ConversationId: inputStruct.ConversationID,
		})

		if err != nil {
			return nil, fmt.Errorf("failed to create conversation: %w", err)
		}
	}

	res, err := appClient.CreateMessage(ctx, &appPB.CreateMessageRequest{
		NamespaceId:    inputStruct.Namespace,
		AppId:          inputStruct.AppID,
		ConversationId: inputStruct.ConversationID,
		Role:           inputStruct.Message.Role,
		Type:           appPB.Message_MessageType(appPB.Message_MessageType_value["MESSAGE_TYPE_TEXT"]),
		Content:        inputStruct.Message.Content,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	messageOutput := res.Message

	output := WriteChatMessageOutput{
		MessageUID: messageOutput.Uid,
		CreateTime: messageOutput.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime: messageOutput.UpdateTime.AsTime().Format(time.RFC3339),
	}

	return base.ConvertToStructpb(output)

}
