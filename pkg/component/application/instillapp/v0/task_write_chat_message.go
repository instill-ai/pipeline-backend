package instillapp

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	appPB "github.com/instill-ai/protogen-go/app/app/v1alpha"
)

func (in *WriteChatMessageInput) validate() error {
	role := in.Message.Role
	if role != "" && role != "user" && role != "assistant" {
		return fmt.Errorf("role must be either 'user' or 'assistant'")
	}
	return nil
}

func (e *execution) writeChatMessage(ctx context.Context, job *base.Job) error {
	inputStruct := &WriteChatMessageInput{}

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

	apps, err := appClient.ListApps(ctx, &appPB.ListAppsRequest{
		NamespaceId: inputStruct.Namespace,
	})

	if err != nil {
		return fmt.Errorf("list apps: %w", err)
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
			return fmt.Errorf("create app: %w", err)
		}
	}

	conversations, err := appClient.ListConversations(ctx, &appPB.ListConversationsRequest{
		NamespaceId: inputStruct.Namespace,
		AppId:       inputStruct.AppID,
		IfAll:       true,
	})

	if err != nil {
		return fmt.Errorf("list conversations: %w", err)
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
			return fmt.Errorf("create conversation: %w", err)
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
		return fmt.Errorf("create message: %w", err)
	}

	messageOutput := res.Message

	output := WriteChatMessageOutput{
		MessageUID: messageOutput.Uid,
		CreateTime: messageOutput.CreateTime.AsTime().Format(time.RFC3339),
		UpdateTime: messageOutput.UpdateTime.AsTime().Format(time.RFC3339),
	}

	err = job.Output.WriteData(ctx, &output)

	if err != nil {
		return fmt.Errorf("write output data: %w", err)
	}

	return nil
}
