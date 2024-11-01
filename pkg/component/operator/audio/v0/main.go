//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package audio

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskDetectActivity = "TASK_DETECT_ACTIVITY"
	taskSegment        = "TASK_SEGMENT"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	once      sync.Once
	comp      *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *base.Job) error
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil)
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
	case taskDetectActivity:
		e.execute = detectActivity
	case taskSegment:
		e.execute = segment
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
