package slack

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/x/errmsg"
)

type MockSlackClient struct{}

func (m *MockSlackClient) GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error) {
	var channels []slack.Channel
	nextCursor := ""
	fakeChannel := slack.Channel{
		GroupConversation: slack.GroupConversation{
			Conversation: slack.Conversation{
				ID: "G0AKFJBEU",
			},
			Name: "test_channel",
		},
	}
	channels = append(channels, fakeChannel)

	return channels, nextCursor, nil
}

func (m *MockSlackClient) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return "", "", nil
}

func (m *MockSlackClient) GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	fakeResp := slack.GetConversationHistoryResponse{
		SlackResponse: slack.SlackResponse{
			Ok: true,
		},
		Messages: []slack.Message{
			{
				Msg: slack.Msg{
					Timestamp:  "1715159446.644219",
					User:       "user123",
					Text:       "Hello, world!",
					ReplyCount: 1,
				},
			},
		},
	}

	return &fakeResp, nil
}

func (m *MockSlackClient) GetConversationReplies(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error) {
	fakeMessages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1715159446.644219",
				User:      "user123",
				Text:      "Hello, world!",
			},
		},
		{
			Msg: slack.Msg{
				Timestamp: "1715159449.399879",
				User:      "user456",
				Text:      "Hello, how are you",
			},
		},
	}
	hasMore := false
	nextCursor := ""
	return fakeMessages, hasMore, nextCursor, nil
}

func (m *MockSlackClient) GetUsersInfo(users ...string) (*[]slack.User, error) {
	resp := &[]slack.User{
		{
			ID:   "user123",
			Name: "Penguin",
		},
		{
			ID:   "user456",
			Name: "Giraffe",
		},
	}

	return resp, nil
}

func TestComponent_ExecuteWriteTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{}
	component := Init(bc)

	testcases := []struct {
		name       string
		botClient  SlackClient
		userClient SlackClient
		input      UserInputWriteTask
		wantResp   WriteTaskResp
		wantErr    string
		wantErrMsg string
	}{
		{
			name:      "ok - as bot",
			botClient: new(MockSlackClient),
			input: UserInputWriteTask{
				ChannelName: "test_channel",
				Message:     "I am unit test",
			},
			wantResp: WriteTaskResp{
				Result: "succeed",
			},
		},
		{
			name:       "ok - as user",
			userClient: new(MockSlackClient),
			input: UserInputWriteTask{
				ChannelName: "test_channel",
				Message:     "I am unit test",
				AsUser:      true,
			},
			wantResp: WriteTaskResp{
				Result: "succeed",
			},
		},
		{
			name:      "nok - missing user token",
			botClient: new(MockSlackClient),
			input: UserInputWriteTask{
				ChannelName: "test_channel",
				Message:     "I am unit test",
				AsUser:      true,
			},
			wantErr:    "empty user token",
			wantErrMsg: "To send messages on behalf of the user, fill the user-token field in the component setup.",
		},
		{
			name:      "nok - missing channel",
			botClient: new(MockSlackClient),
			input: UserInputWriteTask{
				ChannelName: "test_channel_1",
				Message:     "I am unit test",
			},
			wantErr:    "fetching channel ID: couldn't find channel by name",
			wantErrMsg: "Couldn't find channel [test_channel_1].",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup := new(structpb.Struct)

			// It will increase the modification range if we change the input
			// of CreateExecution. So, we replaced it with the code below to
			// cover the test for taskFunctions.go
			x := base.ComponentExecution{
				Component:       component,
				SystemVariables: nil,
				Setup:           setup,
				Task:            taskWriteMessage,
			}
			e := &execution{
				ComponentExecution: x,
				botClient:          tc.botClient,
				userClient:         tc.userClient,
			}
			e.execute = e.sendMessage

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
				if tc.wantErrMsg != "" {
					c.Assert(errmsg.Message(err), qt.Equals, tc.wantErrMsg)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}

func TestComponent_ExecuteReadTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{}
	component := Init(bc)

	mockDateTime, _ := transformTSToDate("1715159449.399879", time.RFC3339)
	testcases := []struct {
		name       string
		input      userInputReadTask
		wantResp   ReadTaskResp
		wantErr    string
		wantErrMsg string
	}{
		{
			name: "ok to read",
			input: userInputReadTask{
				ChannelName:     "test_channel",
				StartToReadDate: "2024-05-05",
			},
			wantResp: ReadTaskResp{
				Conversations: []Conversation{
					{
						UserID:     "user123",
						UserName:   "Penguin",
						Message:    "Hello, world!",
						StartDate:  "2024-05-08",
						LastDate:   "2024-05-08",
						TS:         "1715159446.644219",
						ReplyCount: 1,
						ThreadReplyMessage: []ThreadReplyMessage{
							{
								UserID:   "user456",
								UserName: "Giraffe",
								Message:  "Hello, how are you",
								DateTime: mockDateTime,
							},
						},
					},
				},
			},
		},
		{
			name: "fail to read",
			input: userInputReadTask{
				ChannelName: "test_channel_1",
			},
			wantErr:    `fetching channel ID: couldn't find channel by name`,
			wantErrMsg: "Couldn't find channel [test_channel_1].",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup := new(structpb.Struct)

			// It will increase the modification range if we change the input
			// of CreateExecution. So, we replaced it with the code below to
			// cover the test for taskFunctions.go
			x := base.ComponentExecution{
				Component:       component,
				SystemVariables: nil,
				Setup:           setup,
				Task:            taskReadMessage,
			}
			e := &execution{
				ComponentExecution: x,
				botClient:          new(MockSlackClient),
			}
			e.execute = e.readMessage

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				wantJSON, err := json.Marshal(tc.wantResp)
				c.Assert(err, qt.IsNil)
				c.Check(wantJSON, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Assert(err, qt.ErrorMatches, tc.wantErr)
				}
				if tc.wantErrMsg != "" {
					c.Assert(errmsg.Message(err), qt.Equals, tc.wantErrMsg)
				}
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}
}
