//go:generate compogen readme ./config ./README.mdx
package redis

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/resources/schemas"
)

const (
	taskWriteChatMessage           = "TASK_WRITE_CHAT_MESSAGE"
	taskWriteMultiModalChatMessage = "TASK_WRITE_MULTI_MODAL_CHAT_MESSAGE"
	taskRetrieveChatHistory        = "TASK_RETRIEVE_CHAT_HISTORY"
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

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		additionalJSONBytes := map[string][]byte{
			"schema.yaml": schemas.SchemaYAML,
		}
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, additionalJSONBytes)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	return &execution{
		ComponentExecution: x,
	}, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	client, err := NewClient(e.Setup)
	if err != nil {
		return err
	}
	defer client.Close()

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		var output *structpb.Struct
		switch e.Task {
		case taskWriteChatMessage:
			inputStruct := ChatMessageWriteInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			outputStruct := WriteMessage(client, inputStruct)
			output, err = base.ConvertToStructpb(outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case taskWriteMultiModalChatMessage:
			inputStruct := ChatMultiModalMessageWriteInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			outputStruct := WriteMultiModelMessage(client, inputStruct)
			output, err = base.ConvertToStructpb(outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		case taskRetrieveChatHistory:
			inputStruct := ChatHistoryRetrieveInput{}
			err := base.ConvertFromStructpb(input, &inputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
			outputStruct := RetrieveSessionMessages(client, inputStruct)
			output, err = base.ConvertToStructpb(outputStruct)
			if err != nil {
				job.Error.Error(ctx, err)
				continue
			}
		default:
			job.Error.Error(ctx, fmt.Errorf("unsupported task: %s", e.Task))
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

func (c *component) Test(sysVars map[string]any, setup *structpb.Struct) error {
	client, err := NewClient(setup)
	if err != nil {
		return err
	}
	defer client.Close()

	// Ping the Redis server to check the setup
	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	return nil
}
