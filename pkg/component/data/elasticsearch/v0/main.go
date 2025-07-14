//go:generate compogen readme ./config ./README.mdx
package elasticsearch

import (
	"context"
	"fmt"
	"io"
	"sync"

	_ "embed"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	TaskSearch       = "TASK_SEARCH"
	TaskVectorSearch = "TASK_VECTOR_SEARCH"
	TaskIndex        = "TASK_INDEX"
	TaskMultiIndex   = "TASK_MULTI_INDEX"
	TaskUpdate       = "TASK_UPDATE"
	TaskDelete       = "TASK_DELETE"
	TaskCreateIndex  = "TASK_CREATE_INDEX"
	TaskDeleteIndex  = "TASK_DELETE_INDEX"
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

	execute func(*structpb.Struct) (*structpb.Struct, error)
	client  ESClient
}

type ESClient struct {
	indexClient        esapi.Index
	searchClient       esapi.Search
	updateClient       esapi.UpdateByQuery
	deleteClient       esapi.DeleteByQuery
	createIndexClient  esapi.IndicesCreate
	deleteIndexClient  esapi.IndicesDelete
	sqlTranslateClient esapi.SQLTranslate
	bulkClient         esapi.Bulk
}

type ESSearch func(o ...func(*esapi.SearchRequest)) (*esapi.Response, error)

type ESIndex func(index string, body io.Reader, o ...func(*esapi.IndexRequest)) (*esapi.Response, error)

type ESUpdate func(index []string, o ...func(*esapi.UpdateByQueryRequest)) (*esapi.Response, error)

type ESDelete func(index []string, body io.Reader, o ...func(*esapi.DeleteByQueryRequest)) (*esapi.Response, error)

type ESCreateIndex func(index string, o ...func(*esapi.IndicesCreateRequest)) (*esapi.Response, error)

type ESDeleteIndex func(index []string, o ...func(*esapi.IndicesDeleteRequest)) (*esapi.Response, error)

type ESCount func(o ...func(*esapi.CountRequest)) (*esapi.Response, error)

type ESSQLTranslate func(body io.Reader, o ...func(*esapi.SQLTranslateRequest)) (*esapi.Response, error)

type ESBulk func(body io.Reader, o ...func(*esapi.BulkRequest)) (*esapi.Response, error)

// Init returns an implementation of IComponent that interacts with Elasticsearch.
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
		client:             *newClient(x.Setup),
	}

	switch x.Task {
	case TaskVectorSearch:
		e.execute = e.vectorSearch
	case TaskSearch:
		e.execute = e.search
	case TaskIndex:
		e.execute = e.index
	case TaskUpdate:
		e.execute = e.update
	case TaskDelete:
		e.execute = e.delete
	case TaskCreateIndex:
		e.execute = e.createIndex
	case TaskDeleteIndex:
		e.execute = e.deleteIndex
	case TaskMultiIndex:
		e.execute = e.multiIndex
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
