package slack

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

type userInputReadTask struct {
	ChannelName     string `json:"channel-name"`
	StartToReadDate string `json:"start-to-read-date"`
}

type ReadTaskResp struct {
	Conversations []Conversation `json:"conversations"`
}

type Conversation struct {
	UserID             string               `json:"user-id"`
	UserName           string               `json:"user-name"`
	Message            string               `json:"message"`
	StartDate          string               `json:"start-date"`
	LastDate           string               `json:"last-date"`
	TS                 string               `json:"ts"`
	ReplyCount         int                  `json:"reply-count"`
	ThreadReplyMessage []ThreadReplyMessage `json:"thread-reply-messages"`
}

type ThreadReplyMessage struct {
	UserID   string `json:"user-id"`
	UserName string `json:"user-name"`
	DateTime string `json:"datetime"`
	Message  string `json:"message"`
}

type UserInputWriteTask struct {
	ChannelName string `json:"channel-name"`
	Message     string `json:"message"`
	AsUser      bool   `json:"as-user"`
}

type WriteTaskResp struct {
	Result string `json:"result"`
}

func (e *execution) readMessage(in *structpb.Struct) (*structpb.Struct, error) {
	client := e.botClient

	params := userInputReadTask{}
	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, fmt.Errorf("converting task input: %w", err)
	}

	targetChannelID, err := loopChannelListAPI(client, params.ChannelName)

	if err != nil {
		return nil, fmt.Errorf("fetching channel ID: %w", err)
	}

	resp, err := getConversationHistory(client, targetChannelID, "")
	if err != nil {
		return nil, fmt.Errorf("fetching channel history: %w", err)
	}

	if params.StartToReadDate == "" {
		currentTime := time.Now()
		sevenDaysAgo := currentTime.AddDate(0, 0, -7)
		sevenDaysAgoString := sevenDaysAgo.Format(time.DateOnly)
		params.StartToReadDate = sevenDaysAgoString
	}

	var readTaskResp ReadTaskResp
	err = setAPIRespToReadTaskResp(resp.Messages, &readTaskResp, params.StartToReadDate)
	if err != nil {
		return nil, err
	}

	// TODO: fetch historyAPI first if there are more conversations.
	// if resp.ResponseMetaData.NextCursor != "" {

	// }

	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, conversation := range readTaskResp.Conversations {
		if conversation.ReplyCount > 0 {
			wg.Add(1)
			go func(readTaskResp *ReadTaskResp, idx int) {
				defer wg.Done()
				replies, _ := getConversationReply(client, targetChannelID, readTaskResp.Conversations[idx].TS)
				// TODO: to be discussed about this error handdling
				// fail? or not fail?
				// if err != nil {
				// }

				// TODO: fetch further replies if there are

				mu.Lock()
				err := setRepliedToConversation(readTaskResp, replies, idx)
				// TODO: think a better way to pass lint, maybe use channel
				if err != nil {
					fmt.Println("error when set the output: ", err)
				}
				mu.Unlock()

			}(&readTaskResp, i)
		}
	}
	wg.Wait()

	if readTaskResp.Conversations == nil {
		readTaskResp.Conversations = []Conversation{}
	}

	// To reduce API calls, we get all user information in one call.
	var userIDs []string
	for _, conversation := range readTaskResp.Conversations {
		userIDs = append(userIDs, conversation.UserID)
		if len(conversation.ThreadReplyMessage) > 0 {
			for _, threadReply := range conversation.ThreadReplyMessage {
				userIDs = append(userIDs, threadReply.UserID)
			}
		}
	}

	users, err := client.GetUsersInfo(removeDuplicateUserIDs(userIDs)...)
	if err != nil {
		return nil, fmt.Errorf("fetching user information: %w", err)
	}

	userIDNameMap := createUserIDNameMap(*users)

	for i, conversation := range readTaskResp.Conversations {
		readTaskResp.Conversations[i].UserName = userIDNameMap[conversation.UserID]

		if len(conversation.ThreadReplyMessage) > 0 {
			for j, threadReply := range conversation.ThreadReplyMessage {
				readTaskResp.Conversations[i].ThreadReplyMessage[j].UserName = userIDNameMap[threadReply.UserID]
			}
		}
	}

	out, err := base.ConvertToStructpb(readTaskResp)
	if err != nil {
		return nil, fmt.Errorf("converting task output: %w", err)
	}

	return out, nil
}

func (e *execution) sendMessage(in *structpb.Struct) (*structpb.Struct, error) {
	params := UserInputWriteTask{}
	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, fmt.Errorf("converting task input: %w", err)
	}

	var client SlackClient
	switch {
	case params.AsUser:
		client = e.userClient
		if client == nil {
			return nil, errmsg.AddMessage(
				fmt.Errorf("empty user token"),
				"To send messages on behalf of the user, fill the user-token field in the component setup.",
			)
		}
	default:
		client = e.botClient
	}

	targetChannelID, err := loopChannelListAPI(client, params.ChannelName)
	if err != nil {
		return nil, fmt.Errorf("fetching channel ID: %w", err)
	}

	message := strings.Replace(params.Message, "\\n", "\n", -1)
	_, _, err = client.PostMessage(targetChannelID, slack.MsgOptionText(message, false))
	if err != nil {
		return nil, fmt.Errorf("posting message: %w", err)
	}

	out, err := base.ConvertToStructpb(WriteTaskResp{
		Result: "succeed",
	})
	if err != nil {
		return nil, fmt.Errorf("converting task output: %w", err)
	}

	return out, nil
}
