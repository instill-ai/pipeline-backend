//go:generate compogen readme ./config ./README.mdx
package zilliz

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	TaskVectorSearch     = "TASK_VECTOR_SEARCH"
	TaskUpsert           = "TASK_UPSERT"
	TaskBatchUpsert      = "TASK_BATCH_UPSERT"
	TaskDelete           = "TASK_DELETE"
	TaskCreateCollection = "TASK_CREATE_COLLECTION"
	TaskDropCollection   = "TASK_DROP_COLLECTION"
	TaskCreatePartition  = "TASK_CREATE_PARTITION"
	TaskDropPartition    = "TASK_DROP_PARTITION"
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

	execute func(*structpb.Struct) (*structpb.Struct, error)
	client  *httpclient.Client
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
		client:             newClient(x.Setup, c.GetLogger()),
	}

	switch x.Task {
	case TaskVectorSearch:
		e.execute = e.search
	case TaskUpsert:
		e.execute = e.upsert
	case TaskBatchUpsert:
		e.execute = e.batchUpsert
	case TaskDelete:
		e.execute = e.delete
	case TaskCreateCollection:
		e.execute = e.createCollection
	case TaskDropCollection:
		e.execute = e.dropCollection
	case TaskCreatePartition:
		e.execute = e.createPartition
	case TaskDropPartition:
		e.execute = e.dropPartition
	default:
		return nil, errorsx.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.SequentialExecutor(ctx, jobs, e.execute)
}
