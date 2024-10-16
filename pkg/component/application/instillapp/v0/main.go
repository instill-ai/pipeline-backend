//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package instillapp

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	appPB "github.com/instill-ai/protogen-go/app/app/v1alpha"
)

const (
	TaskReadChatHistory  string = "TASK_READ_CHAT_HISTORY"
	TaskWriteChatMessage string = "TASK_WRITE_CHAT_MESSAGE"
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

	execute    func(*structpb.Struct) (*structpb.Struct, error)
	client     appPB.AppPublicServiceClient
	connection Connection
}

type Connection interface {
	Close() error
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

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	client, connection, err := initAppClient(getAppServerURL(e.SystemVariables))

	if err != nil {
		return nil, fmt.Errorf("failed to create client connection: %w", err)
	}

	e.client, e.connection = client, connection

	switch x.Task {
	case TaskReadChatHistory:
		e.execute = e.readChatHistory
	case TaskWriteChatMessage:
		e.execute = e.writeChatMessage
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
