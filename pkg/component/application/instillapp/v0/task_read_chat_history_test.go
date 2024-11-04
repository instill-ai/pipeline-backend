package instillapp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"

	appPB "github.com/instill-ai/protogen-go/app/app/v1alpha"
	timestampPB "google.golang.org/protobuf/types/known/timestamppb"
)

var (
	fakeInput = ReadChatHistoryInput{
		Namespace:       "namespace",
		AppID:           "app-id",
		ConversationID:  "conversation-id",
		Role:            "user",
		MessageType:     "MESSAGE_TYPE_TEXT",
		Duration:        "2h",
		MaxMessageCount: 10,
	}

	wantOutput = ReadChatHistoryOutput{
		Messages: []Message{
			{
				Content: []Content{
					{
						Type: "text",
						Text: "content",
					},
				},
				Role: "user",
				Name: "name",
			},
		},
	}
)

func Test_ExecuteTaskReadChatHistory(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	mc := minimock.NewController(c)

	grpcServer := grpc.NewServer()
	testServer := mock.NewAppPublicServiceServerMock(mc)

	setListConversationsMock(testServer)
	setListMessages(testServer)

	appPB.RegisterAppPublicServiceServer(grpcServer, testServer)
	lis, err := net.Listen("tcp", ":0")
	c.Assert(err, qt.IsNil)

	go func() {
		err := grpcServer.Serve(lis)
		c.Assert(err, qt.IsNil)
	}()
	defer grpcServer.Stop()
	mockAddress := lis.Addr().String()

	bc := base.Component{}

	comp := Init(bc)

	x, err := comp.CreateExecution(base.ComponentExecution{
		Task: TaskReadChatHistory,
		SystemVariables: map[string]any{
			"__APP_BACKEND":                   mockAddress,
			"__PIPELINE_HEADER_AUTHORIZATION": "Bearer inst token",
			"__PIPELINE_USER_UID":             "user1",
			"__PIPELINE_REQUESTER_UID":        "requester1",
		},
	})

	c.Assert(err, qt.IsNil)

	ir, ow, eh, job := mock.GenerateMockJob(c)

	ir.ReadDataMock.Set(func(ctx context.Context, v interface{}) error {
		switch v := v.(type) {
		case *ReadChatHistoryInput:
			*v = fakeInput
		default:
			panic("unexpected type")
		}
		return nil
	})

	ow.WriteDataMock.Optional().Set(func(ctx context.Context, v interface{}) error {
		switch v := v.(type) {
		case *ReadChatHistoryOutput:
			mock.Equal(len(v.Messages), 1)
		default:
			panic("unexpected type")
		}
		return nil
	})

	eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
		mock.Nil(err)
	})

	err = x.Execute(ctx, []*base.Job{job})
	c.Assert(err, qt.IsNil)
}

func setListConversationsMock(s *mock.AppPublicServiceServerMock) {
	s.ListConversationsMock.Times(1).Set(func(ctx context.Context, in *appPB.ListConversationsRequest) (*appPB.ListConversationsResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		mock.Equal(in.AppId, "app-id")
		mock.Equal(in.ConversationId, "conversation-id")
		return &appPB.ListConversationsResponse{
			Conversations: []*appPB.Conversation{
				{
					Id: "conversation-id",
				},
			},
		}, nil
	})
}

func setListMessages(s *mock.AppPublicServiceServerMock) {
	s.ListMessagesMock.Times(1).Set(func(ctx context.Context, in *appPB.ListMessagesRequest) (*appPB.ListMessagesResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		mock.Equal(in.AppId, "app-id")
		mock.Equal(in.ConversationId, "conversation-id")
		mock.Equal(in.IncludeSystemMessages, true)
		now := time.Now()

		return &appPB.ListMessagesResponse{
			Messages: []*appPB.Message{
				{
					Content:    "fake content",
					Role:       "user",
					Type:       appPB.Message_MessageType(appPB.Message_MessageType_value["MESSAGE_TYPE_TEXT"]),
					CreateTime: timestampPB.New(now),
					UpdateTime: timestampPB.New(now),
				},
			},
			NextPageToken: "",
		}, nil
	})
}
