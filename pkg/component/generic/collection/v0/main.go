//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package collection

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	taskAppend              = "TASK_APPEND"
	taskAssign              = "TASK_ASSIGN"
	taskConcat              = "TASK_CONCAT"
	taskDifference          = "TASK_DIFFERENCE"
	taskIntersection        = "TASK_INTERSECTION"
	taskSplit               = "TASK_SPLIT"
	taskSymmetricDifference = "TASK_SYMMETRIC_DIFFERENCE"
	taskUnion               = "TASK_UNION"
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte

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
		err := comp.LoadDefinition(definitionYAML, nil, tasksYAML, nil, nil)
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
	case taskAppend:
		e.execute = e.append
	case taskAssign:
		e.execute = e.assign
	case taskConcat:
		e.execute = e.concat
	case taskDifference:
		e.execute = e.difference
	case taskIntersection:
		e.execute = e.intersection
	case taskSplit:
		e.execute = e.split
	case taskSymmetricDifference:
		e.execute = e.symmetricDifference
	case taskUnion:
		e.execute = e.union
	default:
		return nil, errorsx.AddMessage(
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
