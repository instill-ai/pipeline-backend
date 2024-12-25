//go:generate compogen readme ./config ./README.mdx
package mongodb

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/x/errmsg"
)

const (
	TaskInsert            = "TASK_INSERT"
	TaskInsertMany        = "TASK_INSERT_MANY"
	TaskFind              = "TASK_FIND"
	TaskUpdate            = "TASK_UPDATE"
	TaskDelete            = "TASK_DELETE"
	TaskDropCollection    = "TASK_DROP_COLLECTION"
	TaskDropDatabase      = "TASK_DROP_DATABASE"
	TaskCreateSearchIndex = "TASK_CREATE_SEARCH_INDEX"
	TaskDropSearchIndex   = "TASK_DROP_SEARCH_INDEX"
	TaskVectorSearch      = "TASK_VECTOR_SEARCH"
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

type MongoCollectionClient interface {
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (cur *mongo.Cursor, err error)
	InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	InsertMany(ctx context.Context, documents []interface{}, opts ...*options.InsertManyOptions) (*mongo.InsertManyResult, error)
	UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	Drop(ctx context.Context) error

	SearchIndexes() mongo.SearchIndexView
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error)
}

type MongoDatabaseClient interface {
	Drop(ctx context.Context) error
}

type MongoSearchIndexClient interface {
	CreateOne(ctx context.Context, model mongo.SearchIndexModel, opts ...*options.CreateSearchIndexesOptions) (string, error)
	DropOne(ctx context.Context, name string, _ ...*options.DropSearchIndexOptions) error
}

type MongoClient struct {
	collectionClient  MongoCollectionClient
	databaseClient    MongoDatabaseClient
	searchIndexClient MongoSearchIndexClient
}

// dbClient for task DropDatabase
// collectionClient for other than task DropDatabase
type execution struct {
	base.ComponentExecution

	execute func(context.Context, *structpb.Struct) (*structpb.Struct, error)
	client  *MongoClient
}

// Init returns an implementation of IComponent that interacts with MongoDB.
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
		client:             &MongoClient{},
	}

	switch x.Task {
	case TaskInsert:
		e.execute = e.insert
	case TaskInsertMany:
		e.execute = e.insertMany
	case TaskFind:
		e.execute = e.find
	case TaskUpdate:
		e.execute = e.update
	case TaskDelete:
		e.execute = e.delete
	case TaskDropCollection:
		e.execute = e.dropCollection
	case TaskDropDatabase:
		e.execute = e.dropDatabase
	case TaskCreateSearchIndex:
		e.execute = e.createSearchIndex
	case TaskDropSearchIndex:
		e.execute = e.dropSearchIndex
	case TaskVectorSearch:
		e.execute = e.vectorSearch
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

// dbClient wont be nil on component test (use mock dbClient)
// collectionClient wont be nil on component test (use mock collectionClient)
// collectionClient will be nil on task DropDatabase
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
