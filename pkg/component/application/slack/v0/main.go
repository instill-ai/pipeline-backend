//go:generate compogen readme ./config ./README.mdx --extraContents setup=.compogen/extra-setup.mdx
package slack

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/slack-go/slack"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskWriteMessage = "TASK_WRITE_MESSAGE"
	taskReadMessage  = "TASK_READ_MESSAGE"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/setup.json
	setupJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/events.json
	eventsJSON []byte

	once sync.Once
	comp *component
)

// slackClient implements the methods we'll need to interact with Slack.
// TODO jvallesm: instead of using an interface and mocking it in the tests,
// create a client with the Slack SDK and use OptionAPIURL to test the
// component.
type slackClient interface {
	GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationReplies(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error)
	GetUsersInfo(users ...string) (*[]slack.User, error)
	AuthTest() (*slack.AuthTestResponse, error)
}

type component struct {
	base.Component
	base.OAuthConnector
}

type execution struct {
	base.ComponentExecution

	botClient  slackClient
	userClient slackClient
	execute    func(*structpb.Struct) (*structpb.Struct, error)
}

func (e *execution) botToken() string {
	return e.Setup.GetFields()["bot-token"].GetStringValue()
}

func (e *execution) userToken() string {
	return e.Setup.GetFields()["user-token"].GetStringValue()
}

// Init returns an implementation of IComponent that interacts with Slack.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, eventsJSON, nil)
		if err != nil {
			panic(err)
		}
	})

	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	// TODO jvallesm: this should be replaced by a validation at the recipe
	// level. The recipe and the setup schema have enough information so the
	// trigger can be aborted earlier.
	if e.botToken() == "" {
		return nil, errmsg.AddMessage(
			fmt.Errorf("missing bot token"),
			"Bot token is a required setup field.",
		)
	}

	e.botClient = newClient(e.botToken())
	if e.userToken() != "" {
		e.userClient = newClient(e.userToken())
	}

	switch x.Task {
	case taskWriteMessage:
		e.execute = e.sendMessage
	case taskReadMessage:
		e.execute = e.readMessage
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {

	return nil
}

func (c *component) ParseEvent(ctx context.Context, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {

	var slackEvent rawSlackMessage
	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	err = unmarshaler.Unmarshal(ctx, rawEvent.Message, &slackEvent)
	if err != nil {
		return nil, err
	}
	var setupStruct slackComponentSetup

	err = unmarshaler.Unmarshal(ctx, rawEvent.Setup, &setupStruct)
	if err != nil {
		return nil, err
	}

	client := newClient(setupStruct.BotToken)

	switch event := slackEvent.Type; event {
	case "url_verification":
		resp := data.Map{
			"challenge": data.NewString(rawEvent.Message.(data.Map)["challenge"].String()),
		}
		return &base.ParsedEvent{
			SkipTrigger:   true,
			ParsedMessage: rawEvent.Message,
			Response:      resp,
		}, nil
	case "event_callback":
		var slackEventType rawSlackEventType
		unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
		err = unmarshaler.Unmarshal(ctx, slackEvent.Event, &slackEventType)
		if err != nil {
			return nil, err
		}

		switch slackEventType.Type {
		case "message":
			return c.handleEventMessage(ctx, client, rawEvent)
		}
	}
	return nil, nil
}

func (c *component) RegisterEvent(ctx context.Context, settings *base.RegisterEventSettings) ([]base.Identifier, error) {

	var setupStruct slackComponentSetup
	var config slackEventNewMessageConfig

	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	err := unmarshaler.Unmarshal(ctx, settings.Setup, &setupStruct)
	if err != nil {
		return nil, err
	}
	err = unmarshaler.Unmarshal(ctx, settings.Config, &config)
	if err != nil {
		return nil, err
	}

	client := newClient(setupStruct.BotToken)
	authTestResp, err := client.AuthTest()
	if err != nil {
		return nil, err
	}
	identifiers := make([]base.Identifier, 0, len(config.ChannelNames))

	for _, channelName := range config.ChannelNames {
		targetChannelID, err := loopChannelListAPI(client, channelName)
		if err != nil {
			return nil, fmt.Errorf("fetching channel ID: %w", err)
		}
		identifier := base.Identifier{
			"user-id":    authTestResp.UserID,
			"channel-id": targetChannelID,
		}
		identifiers = append(identifiers, identifier)
	}

	return identifiers, nil
}

func (c *component) UnregisterEvent(ctx context.Context, settings *base.UnregisterEventSettings, identifiers []base.Identifier) error {
	// We don't register dedciated webhook url for each pipeline in Slack. So we don't need to unregister event here.
	return nil
}

// SupportsOAuth checks whether the component is configured to support OAuth.
func (c *component) SupportsOAuth() bool {
	return c.OAuthConnector.SupportsOAuth()
}
