package slack

import (
	"fmt"
	"strconv"
	"time"

	"github.com/instill-ai/x/errmsg"
	"github.com/slack-go/slack"
)

var types = []string{"private_channel", "public_channel"}

func loopChannelListAPI(client SlackClient, channelName string) (string, error) {
	var apiParams slack.GetConversationsParameters
	apiParams.Types = types

	var targetChannelID string
	for {

		slackChannels, nextCur, err := client.GetConversations(&apiParams)
		if err != nil {
			return "", err
		}

		targetChannelID = getChannelID(channelName, slackChannels)

		if targetChannelID != "" {
			return targetChannelID, nil
		}

		if targetChannelID == "" && nextCur == "" {
			return "", errmsg.AddMessage(
				fmt.Errorf("couldn't find channel by name"),
				fmt.Sprintf("Couldn't find channel [%s].", channelName),
			)
		}

		apiParams.Cursor = nextCur

	}
}

func getChannelID(channelName string, channels []slack.Channel) (channelID string) {
	for _, slackChannel := range channels {
		if channelName == slackChannel.Name {
			return slackChannel.ID
		}
	}
	return ""
}

func getConversationHistory(client SlackClient, channelID string, nextCur string) (*slack.GetConversationHistoryResponse, error) {
	apiHistoryParams := slack.GetConversationHistoryParameters{
		ChannelID: channelID,
		Cursor:    nextCur,
	}

	historiesResp, err := client.GetConversationHistory(&apiHistoryParams)
	if err != nil {
		return nil, err
	}
	if !historiesResp.Ok {
		err := fmt.Errorf("slack api error: %v", historiesResp.Error)
		return nil, err
	}

	return historiesResp, nil
}

func getConversationReply(client SlackClient, channelID string, ts string) ([]slack.Message, error) {
	apiParams := slack.GetConversationRepliesParameters{
		ChannelID: channelID,
		Timestamp: ts,
	}
	msgs, _, nextCur, err := client.GetConversationReplies(&apiParams)

	if err != nil {
		return nil, err
	}

	if nextCur == "" {
		return msgs, nil
	}

	allMsgs := msgs

	for nextCur != "" {
		apiParams.Cursor = nextCur
		msgs, _, nextCur, err = client.GetConversationReplies(&apiParams)
		if err != nil {
			return nil, err
		}
		allMsgs = append(allMsgs, msgs...)
	}

	return allMsgs, nil
}

func setAPIRespToReadTaskResp(apiResp []slack.Message, readTaskResp *ReadTaskResp, startReadDateString string) error {

	for _, msg := range apiResp {
		formatedDateString, err := transformTSToDate(msg.Timestamp, time.DateOnly)
		if err != nil {
			return err
		}

		startReadDate, err := time.Parse(time.DateOnly, startReadDateString)
		if err != nil {
			return err
		}

		formatedDate, err := time.Parse(time.DateOnly, formatedDateString)
		if err != nil {
			return err
		}

		if startReadDate.After(formatedDate) {
			continue
		}

		conversation := Conversation{
			UserID:     msg.User,
			Message:    msg.Text,
			StartDate:  formatedDateString,
			LastDate:   formatedDateString,
			ReplyCount: msg.ReplyCount,
			TS:         msg.Timestamp,
		}
		conversation.ThreadReplyMessage = []ThreadReplyMessage{}
		readTaskResp.Conversations = append(readTaskResp.Conversations, conversation)
	}
	return nil
}

func setRepliedToConversation(resp *ReadTaskResp, replies []slack.Message, idx int) error {
	c := resp.Conversations[idx]
	lastDay, err := time.Parse(time.DateOnly, c.LastDate)
	if err != nil {
		return err
	}
	for _, msg := range replies {

		if c.TS == msg.Timestamp {
			continue
		}

		formatedDateTime, err := transformTSToDate(msg.Timestamp, time.RFC3339)
		if err != nil {
			return err
		}
		reply := ThreadReplyMessage{
			UserID:   msg.User,
			DateTime: formatedDateTime,
			Message:  msg.Text,
		}

		foramtedDate, err := transformTSToDate(msg.Timestamp, time.DateOnly)
		if err != nil {
			return err
		}

		replyDate, err := time.Parse(time.DateOnly, foramtedDate)
		if err != nil {
			return err
		}

		if replyDate.After(lastDay) {
			replyDateString := replyDate.Format(time.DateOnly)
			resp.Conversations[idx].LastDate = replyDateString
		}
		resp.Conversations[idx].ThreadReplyMessage = append(resp.Conversations[idx].ThreadReplyMessage, reply)
	}
	return nil
}

func transformTSToDate(ts string, format string) (string, error) {

	tsFloat, err := strconv.ParseFloat(ts, 64)
	if err != nil {
		return "", err
	}

	timestamp := time.Unix(int64(tsFloat), int64((tsFloat-float64(int64(tsFloat)))*1e9))

	formatedTS := timestamp.Format(format)
	return formatedTS, nil
}

func removeDuplicateUserIDs(userIDs []string) []string {
	encountered := map[string]struct{}{}
	result := []string{}
	for _, v := range userIDs {
		if _, ok := encountered[v]; !ok {
			encountered[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func createUserIDNameMap(users []slack.User) map[string]string {
	userIDNameMap := make(map[string]string)
	for _, user := range users {
		userIDNameMap[user.ID] = user.Name
	}
	return userIDNameMap
}
