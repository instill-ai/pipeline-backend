//go:generate compogen readme ./config ./README.mdx --extraContents intro=.compogen/extra-intro.mdx --extraContents bottom=.compogen/bottom.mdx
package instillartifact

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

const (
	taskUploadFile        string = "TASK_UPLOAD_FILE"
	taskUploadFiles       string = "TASK_UPLOAD_FILES"
	taskGetFilesMetadata  string = "TASK_GET_FILES_METADATA"
	taskGetChunksMetadata string = "TASK_GET_CHUNKS_METADATA"
	taskGetFileInMarkdown string = "TASK_GET_FILE_IN_MARKDOWN"
	taskMatchFileStatus   string = "TASK_MATCH_FILE_STATUS"
	taskSearchChunks      string = "TASK_RETRIEVE"
	taskQuery             string = "TASK_ASK"
	taskSyncFiles         string = "TASK_SYNC_FILES"
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
	client     artifactPB.ArtifactPublicServiceClient
	connection Connection
}

// Init initializes the component.
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

// CreateExecution creates a new execution instance.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	client, connection, err := initArtifactClient(getArtifactServerURL(e.SystemVariables))

	if err != nil {
		return nil, fmt.Errorf("failed to create client connection: %w", err)
	}

	e.client, e.connection = client, connection

	switch x.Task {
	case taskUploadFile:
		e.execute = e.uploadFile
	case taskUploadFiles:
		e.execute = e.uploadFiles
	case taskGetFilesMetadata:
		e.execute = e.getFilesMetadata
	case taskGetChunksMetadata:
		e.execute = e.getChunksMetadata
	case taskGetFileInMarkdown:
		e.execute = e.getFileInMarkdown
	case taskMatchFileStatus:
		e.execute = e.matchFileStatus
	case taskSearchChunks:
		e.execute = e.searchChunks
	case taskQuery:
		e.execute = e.query
	case taskSyncFiles:
		e.execute = e.syncFiles
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

// Execute executes the given jobs sequentially.
func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	defer e.connection.Close()
	return base.SequentialExecutor(ctx, jobs, e.execute)
}

// Connection is the interface that wraps the basic Close method.
type Connection interface {
	Close() error
}
