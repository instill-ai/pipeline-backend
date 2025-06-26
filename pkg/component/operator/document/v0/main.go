//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package document

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/operator/document/v0/transformer"
)

const (
	taskConvertToMarkdown string = "TASK_CONVERT_TO_MARKDOWN"
	taskConvertToText     string = "TASK_CONVERT_TO_TEXT"
	taskConvertToImages   string = "TASK_CONVERT_TO_IMAGES"
	taskSplitInPages      string = "TASK_SPLIT_IN_PAGES"

	pythonInterpreter string = "/opt/venv/bin/python"
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
	logger  *zap.Logger
	execute func(ctx context.Context, job *base.Job) error
}

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

func (e *execution) convertToText(ctx context.Context, job *base.Job) error {
	inputStruct := ConvertToTextInput{}
	err := job.Input.ReadData(ctx, &inputStruct)
	if err != nil {
		return err
	}

	dataURI, err := inputStruct.Document.DataURI()
	if err != nil {
		return err
	}
	transformerInputStruct := transformer.ConvertToTextTransformerInput{
		Document: dataURI.String(),
		Filename: inputStruct.Filename,
	}

	transformerOutputStruct, err := transformer.ConvertToText(transformerInputStruct)
	if err != nil {
		return err
	}

	outputStruct := ConvertToTextOutput{
		Body:     transformerOutputStruct.Body,
		Filename: transformerOutputStruct.Filename,
		Meta:     transformerOutputStruct.Meta,
		MSecs:    transformerOutputStruct.MSecs,
		Error:    transformerOutputStruct.Error,
	}

	err = job.Output.WriteData(ctx, outputStruct)
	return err
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	executionID, _ := uuid.NewV4()
	e := &execution{
		ComponentExecution: x,
		logger: x.GetLogger().With(
			zap.Any("pipelineTriggerID", x.GetSystemVariables()["__PIPELINE_TRIGGER_ID"]),
			zap.String("executionID", executionID.String()),
		),
	}

	switch x.Task {
	case taskConvertToMarkdown:
		e.execute = e.convertDocumentToMarkdown
	case taskConvertToText:
		e.execute = e.convertToText
	case taskConvertToImages:
		e.execute = e.convertDocumentToImages
	case taskSplitInPages:
		e.execute = e.splitInPages
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
