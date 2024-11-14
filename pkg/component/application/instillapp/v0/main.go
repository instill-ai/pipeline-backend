//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package instillapp

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	appPB "github.com/instill-ai/protogen-go/app/app/v1alpha"
)

const (
	// TaskReadChatHistory is the task name for reading chat history
	TaskReadChatHistory string = "TASK_READ_CHAT_HISTORY"
	// TaskWriteChatMessage is the task name for writing chat message
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

	execute    func(context.Context, *base.Job) error
	client     appPB.AppPublicServiceClient
	connection Connection
}

// Connection is the interface for the connection to the application server
type Connection interface {
	Close() error
}

// Init initializes the component
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, nil, tasksJSON, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution creates an execution for the component
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

// Execute executes the jobs concurrently
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
