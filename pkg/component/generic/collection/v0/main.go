//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package collection

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskAssign       = "TASK_ASSIGN"
	taskUnion        = "TASK_UNION"
	taskIntersection = "TASK_INTERSECTION"
	taskDifference   = "TASK_DIFFERENCE"
	taskAppend       = "TASK_APPEND"
	taskConcat       = "TASK_CONCAT"
	taskSplit        = "TASK_SPLIT"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
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
}

// Init returns an implementation of IOperator that processes JSON objects.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case taskAssign:
		e.execute = e.assign
	case taskUnion:
		e.execute = e.union
	case taskIntersection:
		e.execute = e.intersection
	case taskDifference:
		e.execute = e.difference
	case taskAppend:
		e.execute = e.append
	case taskConcat:
		e.execute = e.concat
	case taskSplit:
		e.execute = e.split
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}
	return e, nil
}

// Execute processes the input JSON object and returns the result.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
