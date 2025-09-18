//go:generate compogen readme ./config ./README.mdx
package video

import (
	"context"
	"fmt"
	"sync"

	_ "embed"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// ffmpegMutex protects concurrent access to ffmpeg library functions
// to prevent data races in the library's internal global state initialization
var ffmpegMutex sync.Mutex

const (
	taskSegment       = "TASK_SEGMENT"
	taskSubsample     = "TASK_SUBSAMPLE"
	taskExtractAudio  = "TASK_EXTRACT_AUDIO"
	taskExtractFrames = "TASK_EXTRACT_FRAMES"
	taskEmbedAudio    = "TASK_EMBED_AUDIO"
)

var (
	//go:embed config/definition.yaml
	definitionYAML []byte
	//go:embed config/tasks.yaml
	tasksYAML []byte
	once      sync.Once
	comp      *component
)

type component struct {
	base.Component
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *base.Job) error
}

func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionYAML, nil, tasksYAML, nil, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline trigger.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {
	e := &execution{ComponentExecution: x}

	switch x.Task {
	case taskSegment:
		e.execute = segment
	case taskSubsample:
		e.execute = subsample
	case taskExtractAudio:
		e.execute = extractAudio
	case taskExtractFrames:
		e.execute = extractFrames
	case taskEmbedAudio:
		e.execute = embedAudio
	default:
		return nil, fmt.Errorf("%s task is not supported", x.Task)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}
