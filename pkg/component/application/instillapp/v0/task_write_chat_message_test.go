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
	fakeWriteInput = WriteChatMessageInput{
		Namespace:      "namespace",
		AppID:          "app-id",
		ConversationID: "conversation-id",
		Message: WriteMessage{
			Role:    "user",
			Content: "hello",
		},
	}

	now       = time.Now()
	nowString = now.Format(time.RFC3339)
)

func Test_ExecuteWriteChatMEssage(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	mc := minimock.NewController(c)

	grpcServer := grpc.NewServer()
	testServer := mock.NewAppPublicServiceServerMock(mc)

	setListAppsMock(testServer)
	setCreateAppMock(testServer)
	setListConversationMock(testServer)
	setCreateConversationMock(testServer)
	setCreateMessageMock(testServer)

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
		Task: TaskWriteChatMessage,
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
		case *WriteChatMessageInput:
			*v = fakeWriteInput
		default:
			panic("unexpected type")
		}
		return nil
	})

	ow.WriteDataMock.Set(func(ctx context.Context, v interface{}) error {
		switch v := v.(type) {
		case *WriteChatMessageOutput:
			mock.Equal(v.MessageUID, "message-uid")
			mock.Equal(v.CreateTime, nowString)
			mock.Equal(v.UpdateTime, nowString)
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

func setListAppsMock(testServer *mock.AppPublicServiceServerMock) {
	testServer.ListAppsMock.Times(1).Set(func(ctx context.Context, in *appPB.ListAppsRequest) (*appPB.ListAppsResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		return &appPB.ListAppsResponse{
			Apps: []*appPB.App{
				{
					AppId: "app-id-not-match",
				},
			},
		}, nil
	})
}

func setCreateAppMock(testServer *mock.AppPublicServiceServerMock) {
	testServer.CreateAppMock.Times(1).Set(func(ctx context.Context, in *appPB.CreateAppRequest) (*appPB.CreateAppResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		mock.Equal(in.Id, "app-id")
		return &appPB.CreateAppResponse{}, nil
	})
}

func setListConversationMock(testServer *mock.AppPublicServiceServerMock) {
	testServer.ListConversationsMock.Times(1).Set(func(ctx context.Context, in *appPB.ListConversationsRequest) (*appPB.ListConversationsResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		mock.Equal(in.AppId, "app-id")
		mock.Equal(in.IfAll, true)
		return &appPB.ListConversationsResponse{
			Conversations: []*appPB.Conversation{
				{
					Id: "conversation-id-not-match",
				},
			},
		}, nil
	})
}

func setCreateConversationMock(testServer *mock.AppPublicServiceServerMock) {
	testServer.CreateConversationMock.Times(1).Set(func(ctx context.Context, in *appPB.CreateConversationRequest) (*appPB.CreateConversationResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		mock.Equal(in.AppId, "app-id")
		mock.Equal(in.ConversationId, "conversation-id")
		return &appPB.CreateConversationResponse{}, nil
	})
}

func setCreateMessageMock(testServer *mock.AppPublicServiceServerMock) {
	testServer.CreateMessageMock.Times(1).Set(func(ctx context.Context, in *appPB.CreateMessageRequest) (*appPB.CreateMessageResponse, error) {
		mock.Equal(in.NamespaceId, "namespace")
		mock.Equal(in.AppId, "app-id")
		mock.Equal(in.ConversationId, "conversation-id")
		mock.Equal(in.Role, "user")
		mock.Equal(in.Type, appPB.Message_MessageType(appPB.Message_MessageType_value["MESSAGE_TYPE_TEXT"]))
		mock.Equal(in.Content, "hello")
		return &appPB.CreateMessageResponse{
			Message: &appPB.Message{
				Uid:        "message-uid",
				CreateTime: timestampPB.New(now),
				UpdateTime: timestampPB.New(now),
			},
		}, nil
	})
}
