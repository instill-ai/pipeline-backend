package slack

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/slack-go/slack"
)

func (c *component) identifyEventMessage(ctx context.Context, rawEvent *base.RawEvent) (identifierResult *base.IdentifierResult, err error) {
	var raw rawSlackMessage
	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	err = unmarshaler.Unmarshal(ctx, rawEvent.Message, &raw)
	if err != nil {
		return nil, err
	}
	var rawMessage rawSlackMessageEvent
	err = unmarshaler.Unmarshal(ctx, raw.Event, &rawMessage)
	if err != nil {
		return nil, err
	}
	identifiers := []base.Identifier{}
	for _, authorization := range raw.Authorizations {
		identifier := map[string]any{
			"user-id":    authorization.UserID,
			"channel-id": rawMessage.Channel,
		}
		identifiers = append(identifiers, identifier)
	}
	return &base.IdentifierResult{
		Identifiers: identifiers,
	}, nil
}

func (c *component) handleEventMessage(ctx context.Context, client slackClient, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {
	var raw rawSlackMessage
	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	err = unmarshaler.Unmarshal(ctx, rawEvent.Message, &raw)
	if err != nil {
		return nil, err
	}
	var rawMessage rawSlackMessageEvent
	err = unmarshaler.Unmarshal(ctx, raw.Event, &rawMessage)
	if err != nil {
		return nil, err
	}

	userInfo, err := client.GetUsersInfo(rawMessage.User)
	if err != nil {
		return nil, err
	}
	channelInfo := map[string]slack.Channel{}
	cursor := ""
	for {
		channels, nextCursor, err := client.GetConversations(&slack.GetConversationsParameters{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		cursor = nextCursor
		for _, channel := range channels {
			channelInfo[channel.ID] = channel
		}
		if nextCursor == "" {
			break
		}
	}

	slackEventMessageStruct := slackEventNewMessage{
		Timestamp: rawMessage.TS,
		Channel: channel{
			ID:   rawMessage.Channel,
			Name: channelInfo[rawMessage.Channel].Name,
		},
		Text: rawMessage.Text,
	}
	if rawMessage.User != "" && len(*userInfo) > 0 {
		slackEventMessageStruct.User = user{
			ID:       rawMessage.User,
			Name:     (*userInfo)[0].Name,
			RealName: (*userInfo)[0].RealName,
			Profile: userProfile{
				DisplayName: (*userInfo)[0].Profile.DisplayName,
			},
		}
	}

	config := slackEventNewMessageConfig{}
	err = unmarshaler.Unmarshal(ctx, rawEvent.Config, &config)
	if err != nil {
		return nil, err
	}

	marshaler := data.NewMarshaler()
	parsed, err := marshaler.Marshal(slackEventMessageStruct)
	if err != nil {
		return nil, err
	}

	return &base.ParsedEvent{
		Response:      data.Map{},
		ParsedMessage: parsed,
	}, nil
}
