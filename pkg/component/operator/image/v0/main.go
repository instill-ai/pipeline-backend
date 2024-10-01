//go:generate compogen readme ./config ./README.mdx
package image

import (
	"context"
	"fmt"
	"sync"

	_ "embed"
	_ "image/gif"
	_ "image/png"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
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
	execute func(*structpb.Struct, *base.Job, context.Context) (*structpb.Struct, error)
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

// CreateExecution initializes a component execution that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case "TASK_CONCAT":
		e.execute = concat
	case "TASK_CROP":
		e.execute = crop
	case "TASK_RESIZE":
		e.execute = resize
	case "TASK_DRAW_CLASSIFICATION":
		e.execute = drawClassification
	case "TASK_DRAW_DETECTION":
		e.execute = drawDetection
	case "TASK_DRAW_KEYPOINT":
		e.execute = drawKeypoint
	case "TASK_DRAW_SEMANTIC_SEGMENTATION":
		e.execute = drawSemanticSegmentation
	case "TASK_DRAW_INSTANCE_SEGMENTATION":
		e.execute = drawInstanceSegmentation
	case "TASK_DRAW_OCR":
		e.execute = drawOCR
	default:
		return nil, fmt.Errorf("not supported task: %s", x.Task)
	}

	return e, nil
}

// Execute executes the derived execution
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
