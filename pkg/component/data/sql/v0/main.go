//go:generate compogen readme ./config ./README.mdx
package sql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "embed"

	"github.com/jmoiron/sqlx"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	errorsx "github.com/instill-ai/x/errors"
)

const (
	TaskInsert      = "TASK_INSERT"
	TaskInsertMany  = "TASK_INSERT_MANY"
	TaskUpdate      = "TASK_UPDATE"
	TaskSelect      = "TASK_SELECT"
	TaskDelete      = "TASK_DELETE"
	TaskCreateTable = "TASK_CREATE_TABLE"
	TaskDropTable   = "TASK_DROP_TABLE"
)

//go:embed config/definition.yaml
var definitionYAML []byte

//go:embed config/setup.yaml
var setupYAML []byte

//go:embed config/tasks.yaml
var tasksYAML []byte

var once sync.Once
var comp *component

type SQLClient interface {
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Queryx(query string, args ...interface{}) (*sqlx.Rows, error)
}

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution

	execute func(*structpb.Struct) (*structpb.Struct, error)
	client  SQLClient
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
	client, err := newClient(x.Setup)
	if err != nil {
		return nil, err
	}
	e := &execution{
		ComponentExecution: x,
		client:             client,
	}

	switch x.Task {
	case TaskInsert:
		e.execute = e.insert
	case TaskUpdate:
		e.execute = e.update
	case TaskSelect:
		e.execute = e.selects
	case TaskDelete:
		e.execute = e.delete
	case TaskCreateTable:
		e.execute = e.createTable
	case TaskDropTable:
		e.execute = e.dropTable
	case TaskInsertMany:
		e.execute = e.insertMany
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
