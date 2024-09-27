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

const (
	apiKey = "testkey"
)

func TestComponent_ExecuteWriteTask(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	bc := base.Component{}
	connector := Init(bc)

	testcases := []struct {
		name     string
		input    UserInputWriteTask
		wantResp WriteTaskResp
		wantErr  string
	}{
		{
			name: "ok to write",
			input: UserInputWriteTask{
				ChannelName: "test_channel",
				Message:     "I am unit test",
			},
			wantResp: WriteTaskResp{
				Result: "succeed",
			},
		},
		{
			name: "fail to write",
			input: UserInputWriteTask{
				ChannelName: "test_channel_1",
				Message:     "I am unit test",
			},
			wantErr: `there is no match name in slack channel \[test_channel_1\]`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {

			setup, err := structpb.NewStruct(map[string]any{
				"api-key": apiKey,
			})
			c.Assert(err, qt.IsNil)

			// It will increase the modification range if we change the input of CreateExecution.
			// So, we replaced it with the code below to cover the test for taskFunctions.go
			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskWriteMessage},
				client:             &MockSlackClient{},
			}
			e.execute = e.sendMessage

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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
	connector := Init(bc)

	mockDateTime, _ := transformTSToDate("1715159449.399879", time.RFC3339)
	testcases := []struct {
		name     string
		input    UserInputReadTask
		wantResp ReadTaskResp
		wantErr  string
	}{
		{
			name: "ok to read",
			input: UserInputReadTask{
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
			input: UserInputReadTask{
				ChannelName: "test_channel_1",
			},
			wantErr: `there is no match name in slack channel \[test_channel_1\]`,
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"api-key": apiKey,
			})
			c.Assert(err, qt.IsNil)

			// It will increase the modification range if we change the input of CreateExecution.
			// So, we replaced it with the code below to cover the test for taskFunctions.go
			e := &execution{
				ComponentExecution: base.ComponentExecution{Component: connector, SystemVariables: nil, Setup: setup, Task: taskReadMessage},
				client:             &MockSlackClient{},
			}
			e.execute = e.readMessage

			pbIn, err := base.ConvertToStructpb(tc.input)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := base.GenerateMockJob(c)
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
			})

			err = e.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})

	}

}
