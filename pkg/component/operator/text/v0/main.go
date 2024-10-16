//go:generate compogen readme ./config ./README.mdx
package text

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"google.golang.org/protobuf/types/known/structpb"
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
	execute func(*structpb.Struct) (*structpb.Struct, error)
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

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case taskChunkText:
		e.execute = e.chunkTextHandler
	default:
		return nil, fmt.Errorf("unsupported task: %s", x.Task)
	}

	return e, nil
}

// Execute executes the derived execution
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		output, err := e.execute(input)
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

// chunkTextHandler handles the chunking text task
func (e *execution) chunkTextHandler(input *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct ChunkTextInput
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input: %w", err)
	}

	var outputStruct ChunkTextOutput
	if inputStruct.Strategy.Setting.ChunkMethod == "Markdown" {
		outputStruct, err = chunkMarkdown(inputStruct)
	} else {
		outputStruct, err = chunkText(inputStruct)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to chunk text: %w", err)
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output: %w", err)
	}

	return output, nil
}
