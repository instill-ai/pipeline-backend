//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package document

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	taskConvertToMarkdown string = "TASK_CONVERT_TO_MARKDOWN"
	taskConvertToText     string = "TASK_CONVERT_TO_TEXT"
	taskConvertToImages   string = "TASK_CONVERT_TO_IMAGES"
	pythonInterpreter     string = "/opt/venv/bin/python"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte

	//go:embed execution/task_convert_to_markdown.py
	taskConvertToMarkdownExecution string
	//go:embed pdf_to_markdown/pdf_transformer.py
	pdfTransformer string
	//go:embed pdf_to_markdown/page_image_processor.py
	imageProcessor string

	//go:embed execution/task_convert_to_images.py
	taskConvertToImagesExecution string

	once sync.Once
	comp *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	execute                func(*structpb.Struct) (*structpb.Struct, error)
	getMarkdownTransformer MarkdownTransformerGetterFunc
}

type MarkdownTransformerGetterFunc func(fileExtension string, inputStruct *ConvertDocumentToMarkdownInput) (MarkdownTransformer, error)

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

func (e *execution) convertToText(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := ConvertToTextInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, err
	}
	outputStruct, err := ConvertToText(inputStruct)
	if err != nil {
		return nil, err
	}
	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// CreateExecution initializes a connector executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{
		ComponentExecution:     x,
		getMarkdownTransformer: GetMarkdownTransformer,
	}

	switch x.Task {
	case taskConvertToMarkdown:
		e.execute = e.convertDocumentToMarkdown
	case taskConvertToText:
		e.execute = e.convertToText
	case taskConvertToImages:
		e.execute = e.convertDocumentToImages
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
