//go:generate compogen readme ./config ./README.mdx
package slack

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/slack-go/slack"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
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

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute func(*structpb.Struct) (*structpb.Struct, error)
	client  SlackClient
}

type SlackClient interface {
	GetConversations(params *slack.GetConversationsParameters) ([]slack.Channel, string, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationReplies(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error)
	GetUsersInfo(users ...string) (*[]slack.User, error)
}

// Init returns an implementation of IComponent that interacts with Slack.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, nil)
		if err != nil {
			panic(err)
		}
	})

	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{
		ComponentExecution: x,
		client:             newClient(x.Setup),
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

func (c *component) HandleVerificationEvent(header map[string][]string, req *structpb.Struct, setup map[string]any) (isVerification bool, resp *structpb.Struct, err error) {

	switch event := req.GetFields()["type"].GetStringValue(); event {
	case "url_verification":
		resp, _ := structpb.NewStruct(map[string]any{
			"challenge": req.GetFields()["challenge"].GetStringValue(),
		})
		return true, resp, nil
	default:
		return false, nil, nil

	}

}

func (c *component) ParseEvent(ctx context.Context, req *structpb.Struct, setup map[string]any) (parsed *structpb.Struct, err error) {
	// TODO: parse and validate event
	return req, nil
}
