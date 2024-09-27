//go:generate compogen readme ./config ./README.mdx
package text

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskChunkText string = "TASK_CHUNK_TEXT"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	once      sync.Once
	comp      *component
)

// Operator is the derived operator
type component struct {
	base.Component
}

// Execution is the derived execution
type execution struct {
	base.ComponentExecution
}

// Init initializes the operator
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

// CreateExecution initializes a connector executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{ComponentExecution: x}, nil
}

// Execute executes the derived execution
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		switch e.Task {
		case taskChunkText:
			inputStruct := ChunkTextInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}

			var outputStruct ChunkTextOutput
			if inputStruct.Strategy.Setting.ChunkMethod == "Markdown" {
				outputStruct, err = chunkMarkdown(inputStruct)
			} else {
				outputStruct, err = chunkText(inputStruct)
			}

			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			output, err := base.ConvertToStructpb(outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			err = job.Output.Write(ctx, output)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		default:
			job.Error.Error(ctx, fmt.Errorf("not supported task: %s", e.Task))
			continue
		}
	}
	return nil
}
