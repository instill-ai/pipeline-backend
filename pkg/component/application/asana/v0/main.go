//go:generate compogen readme ./config ./README.mdx
package asana

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/setup.yaml
	setupYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte

	once sync.Once
	comp *component
)

const (
	apiBaseURL         = "https://app.asana.com/api/1.0"
	TaskAsanaGoal      = "TASK_CRUD_GOAL"
	TaskAsanaTask      = "TASK_CRUD_TASK"
	TaskAsanaPortfolio = "TASK_CRUD_PORTFOLIO"
	TaskAsanaProject   = "TASK_CRUD_PROJECT"
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *structpb.Struct) (*structpb.Struct, error)
	client  Client
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	ctx := context.Background()
	asanaClient, err := newClient(ctx, x.Setup, c.Logger)
	if err != nil {
		return nil, err
	}
	e := &execution{
		ComponentExecution: x,
		client:             *asanaClient,
	}
	switch x.Task {
	case TaskAsanaGoal:
		e.execute = e.client.GoalRelatedTask
	case TaskAsanaTask:
		e.execute = e.client.TaskRelatedTask
	case TaskAsanaPortfolio:
		e.execute = e.client.PortfolioRelatedTask
	case TaskAsanaProject:
		e.execute = e.client.ProjectRelatedTask
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
		// TODO: migrate to new interface with default value

		action := input
		if input.GetFields()["action"].GetStringValue() == "" {
			action = input.GetFields()["action"].GetStructValue()
		}
		output, err := e.execute(ctx, action)
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
