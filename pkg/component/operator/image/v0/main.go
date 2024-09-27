//go:generate compogen readme ./config ./README.mdx
package image

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"sync"

	_ "embed"
	_ "image/gif"
	_ "image/jpeg"
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

	var base64ByteImg []byte
	for _, job := range jobs {

		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		b, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(input.Fields["image"].GetStringValue()))
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		img, _, err := image.Decode(bytes.NewReader(b))
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}

		switch e.Task {
		case "TASK_DRAW_CLASSIFICATION":
			base64ByteImg, err = drawClassification(img, input.Fields["category"].GetStringValue(), input.Fields["score"].GetNumberValue())
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case "TASK_DRAW_DETECTION":
			base64ByteImg, err = drawDetection(img, input.Fields["objects"].GetListValue().Values)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case "TASK_DRAW_KEYPOINT":
			base64ByteImg, err = drawKeypoint(img, input.Fields["objects"].GetListValue().Values)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case "TASK_DRAW_OCR":
			base64ByteImg, err = drawOCR(img, input.Fields["objects"].GetListValue().Values)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case "TASK_DRAW_INSTANCE_SEGMENTATION":
			base64ByteImg, err = drawInstanceSegmentation(img, input.Fields["objects"].GetListValue().Values)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case "TASK_DRAW_SEMANTIC_SEGMENTATION":
			base64ByteImg, err = drawSemanticSegmentation(img, input.Fields["stuffs"].GetListValue().Values)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		default:
			job.Error.Error(ctx, fmt.Errorf("not supported task: %s", e.Task))
			continue
		}

		output := &structpb.Struct{Fields: make(map[string]*structpb.Value)}

		output.Fields["image"] = &structpb.Value{
			Kind: &structpb.Value_StringValue{
				StringValue: fmt.Sprintf("data:image/jpeg;base64,%s", string(base64ByteImg)),
			},
		}

		err = job.Output.Write(ctx, output)
		if err != nil {
			return err
		}
	}
	return nil
}
