package slack

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type UserInputReadTask struct {
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
}

type WriteTaskResp struct {
	Result string `json:"result"`
}

func (e *execution) readMessage(in *structpb.Struct) (*structpb.Struct, error) {

	params := UserInputReadTask{}

	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, err
	}

	targetChannelID, err := loopChannelListAPI(e, params.ChannelName)

	if err != nil {
		return nil, err
	}

	resp, err := getConversationHistory(e, targetChannelID, "")
	if err != nil {
		return nil, err
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
				replies, _ := getConversationReply(e, targetChannelID, readTaskResp.Conversations[idx].TS)
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

	userIDs = removeDuplicateUserIDs(userIDs)
	users, err := e.client.GetUsersInfo(userIDs...)

	if err != nil {
		return nil, err
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
		return nil, err
	}

	return out, nil
}

func (e *execution) sendMessage(in *structpb.Struct) (*structpb.Struct, error) {
	params := UserInputWriteTask{}

	if err := base.ConvertFromStructpb(in, &params); err != nil {
		return nil, err
	}

	targetChannelID, err := loopChannelListAPI(e, params.ChannelName)
	if err != nil {
		return nil, err
	}

	message := strings.Replace(params.Message, "\\n", "\n", -1)
	_, _, err = e.client.PostMessage(targetChannelID, slack.MsgOptionText(message, false))

	if err != nil {
		return nil, err
	}

	out, err := base.ConvertToStructpb(WriteTaskResp{
		Result: "succeed",
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}
