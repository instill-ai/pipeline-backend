//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package github

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskListPRs             = "TASK_LIST_PULL_REQUESTS"
	taskGetPR               = "TASK_GET_PULL_REQUEST"
	taskGetCommit           = "TASK_GET_COMMIT"
	taskGetReviewComments   = "TASK_LIST_REVIEW_COMMENTS"
	taskCreateReviewComment = "TASK_CREATE_REVIEW_COMMENT"
	taskListIssues          = "TASK_LIST_ISSUES"
	taskGetIssue            = "TASK_GET_ISSUE"
	taskCreateIssue         = "TASK_CREATE_ISSUE"
	taskCreateWebhook       = "TASK_CREATE_WEBHOOK"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/setup.json
	setupJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/event.json
	eventJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
	base.OAuthConnector
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *structpb.Struct) (*structpb.Struct, error)
	client  Client
}

// Init returns an implementation of IComponent that interacts with Slack.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, eventJSON, nil)
		if err != nil {
			panic(err)
		}
	})

	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	ctx := context.Background()
	githubClient := newClient(ctx, x.Setup)
	e := &execution{
		ComponentExecution: x,
		client:             githubClient,
	}
	switch x.Task {
	case taskListPRs:
		e.execute = e.client.listPullRequestsTask
	case taskGetPR:
		e.execute = e.client.getPullRequestTask
	case taskGetReviewComments:
		e.execute = e.client.listReviewCommentsTask
	case taskCreateReviewComment:
		e.execute = e.client.createReviewCommentTask
	case taskGetCommit:
		e.execute = e.client.getCommitTask
	case taskListIssues:
		e.execute = e.client.listIssuesTask
	case taskGetIssue:
		e.execute = e.client.getIssueTask
	case taskCreateIssue:
		e.execute = e.client.createIssueTask
	case taskCreateWebhook:
		e.execute = e.client.createWebhookTask
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		// TODO: use FillInDefaultValues for all components
		if _, err := e.FillInDefaultValues(input); err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		output, err := e.execute(ctx, input)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		err = job.Output.Write(ctx, output)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
	}

	return nil
}

func (c *component) HandleVerificationEvent(header map[string][]string, req *structpb.Struct, setup map[string]any) (isVerification bool, resp *structpb.Struct, err error) {
	if len(header["x-github-event"]) > 0 && header["x-github-event"][0] == "ping" {
		return true, nil, nil
	}
	return false, nil, nil
}

func (c *component) ParseEvent(ctx context.Context, req *structpb.Struct, setup map[string]any) (parsed *structpb.Struct, err error) {
	// TODO: parse and validate event
	return req, nil
}

// SupportsOAuth checks whether the component is configured to support OAuth.
func (c *component) SupportsOAuth() bool {
	return c.OAuthConnector.SupportsOAuth()
}
