//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package jira

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskCreateIssue  = "TASK_CREATE_ISSUE"
	taskCreateSprint = "TASK_CREATE_SPRINT"
	taskGetIssue     = "TASK_GET_ISSUE"
	taskGetSprint    = "TASK_GET_SPRINT"
	taskListBoards   = "TASK_LIST_BOARDS"
	taskListIssues   = "TASK_LIST_ISSUES"
	taskListSprints  = "TASK_LIST_SPRINTS"
	taskUpdateIssue  = "TASK_UPDATE_ISSUE"
	taskUpdateSprint = "TASK_UPDATE_SPRINT"
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
	execute func(context.Context, *base.Job) error
	client  client
}

// Init returns an implementation of IComponent that interacts with Slack.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, nil, nil)
		if err != nil {
			panic(err)
		}
	})

	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	ctx := context.Background()
	client, err := newClient(ctx, x.Setup, c.Logger)
	if err != nil {
		return nil, err
	}
	e := &execution{
		ComponentExecution: x,
		client:             *client,
	}
	// docs: https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/#about
	switch x.Task {
	case taskListBoards:
		e.execute = e.client.listBoards
	case taskListIssues:
		e.execute = e.client.listIssues
	case taskListSprints:
		e.execute = e.client.listSprints
	case taskGetIssue:
		e.execute = e.client.getIssue
	case taskGetSprint:
		e.execute = e.client.getSprint
	case taskCreateIssue:
		e.execute = e.client.createIssue
	case taskUpdateIssue:
		e.execute = e.client.updateIssue
	case taskCreateSprint:
		e.execute = e.client.createSprint
	case taskUpdateSprint:
		e.execute = e.client.updateSprint
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
