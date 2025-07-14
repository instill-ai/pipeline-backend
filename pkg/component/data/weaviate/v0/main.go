//go:generate compogen readme ./config ./README.mdx
package weaviate

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	TaskVectorSearch     = "TASK_VECTOR_SEARCH"
	TaskInsert           = "TASK_INSERT"
	TaskUpdate           = "TASK_UPDATE"
	TaskDelete           = "TASK_DELETE"
	TaskBatchInsert      = "TASK_BATCH_INSERT"
	TaskDeleteCollection = "TASK_DELETE_COLLECTION"
)

//go:embed config/definition.yaml
var definitionYAML []byte

//go:embed config/setup.yaml
var setupYAML []byte

//go:embed config/tasks.yaml
var tasksYAML []byte

var once sync.Once
var comp *component

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute    func(context.Context, *structpb.Struct) (*structpb.Struct, error)
	client     *weaviate.Client
	mockClient *MockWeaviateClient
}

type MockWeaviateClient struct {
	Successful   int
	VectorSearch Result
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionYAML, setupYAML, tasksYAML, nil, nil)
		if err != nil {
			panic(err)
		}
	})

	return comp
}

func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{
		ComponentExecution: x,
		client:             newClient(x.Setup),
	}

	switch x.Task {
	case TaskVectorSearch:
		e.execute = e.vectorSearch
	case TaskInsert:
		e.execute = e.insert
	case TaskUpdate:
		e.execute = e.update
	case TaskDelete:
		e.execute = e.delete
	case TaskBatchInsert:
		e.execute = e.batchInsert
	case TaskDeleteCollection:
		e.execute = e.deleteCollection
	default:
		return nil, errorsx.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {

	for _, job := range jobs {
		input, err := job.Input.Read(ctx)
		if err != nil {
			job.Error.Error(ctx, err)
			continue
		}
		output, err := e.execute(ctx, input)
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
