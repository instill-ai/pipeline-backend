package instillapp

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	appPB "github.com/instill-ai/protogen-go/app/app/v1alpha"
)

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

func (out *ReadChatHistoryOutput) filter(inputStruct *ReadChatHistoryInput, messages []*appPB.Message) {
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

func (e *execution) readChatHistory(ctx context.Context, job *base.Job) error {

	inputStruct := &ReadChatHistoryInput{}

	err := job.Input.ReadData(ctx, inputStruct)

	if err != nil {
		return fmt.Errorf("read input data: %w", err)
	}

	err = inputStruct.validate()

	if err != nil {
		return fmt.Errorf("validate input: %w", err)
	}

	appClient, connection := e.client, e.connection
	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	output := &ReadChatHistoryOutput{
		Messages: make([]Message, 0),
	}

	conversationRes, err := appClient.ListConversations(ctx, &appPB.ListConversationsRequest{
		NamespaceId:    inputStruct.Namespace,
		AppId:          inputStruct.AppID,
		ConversationId: inputStruct.ConversationID,
	})

	if len(conversationRes.Conversations) == 0 || err != nil {
		err = job.Output.WriteData(ctx, output)

		if err != nil {
			return fmt.Errorf("write output data: %w", err)
		}

		return nil
	}

	var nextPageToken string
	for {
		res, err := appClient.ListMessages(ctx, &appPB.ListMessagesRequest{
			NamespaceId:           inputStruct.Namespace,
			AppId:                 inputStruct.AppID,
			ConversationId:        inputStruct.ConversationID,
			IncludeSystemMessages: true,
			PageToken:             nextPageToken,
		})

		if err != nil {
			return fmt.Errorf("list messages: %w", err)
		}

		output.filter(inputStruct, res.Messages)

		if res.NextPageToken == "" || (len(output.Messages) >= inputStruct.MaxMessageCount && inputStruct.MaxMessageCount > 0) {
			break
		}

		nextPageToken = res.NextPageToken
	}

	err = job.Output.WriteData(ctx, output)

	if err != nil {
		return fmt.Errorf("write output data: %w", err)
	}

	return nil
}
