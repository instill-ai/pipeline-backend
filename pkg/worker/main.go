package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/usage"

	componentbase "github.com/instill-ai/component/base"
	componentstore "github.com/instill-ai/component/store"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowParam) error
	ComponentActivity(ctx context.Context, param *ComponentActivityParam) (*ComponentActivityParam, error)
	PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error)
	PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error
	UsageCollectActivity(ctx context.Context, param *UsageCollectActivityParam) error
	UsageCheckActivity(ctx context.Context, param *UsageCheckActivityParam) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository           repository.Repository
	redisClient          *redis.Client
	influxDBWriteClient  api.WriteAPI
	component            *componentstore.Store
	pipelineUsageHandler usage.PipelineUsageHandler
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	r repository.Repository,
	rd *redis.Client,
	i api.WriteAPI,
	cs componentstore.ComponentSecrets,
	uh map[string]componentbase.UsageHandlerCreator,
	pipelineUsageHandler usage.PipelineUsageHandler,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	if pipelineUsageHandler == nil {
		pipelineUsageHandler = usage.NewNoopPipelineUsageHandler()
	}
	return &worker{
		repository:           r,
		redisClient:          rd,
		influxDBWriteClient:  i,
		component:            componentstore.Init(logger, cs, uh),
		pipelineUsageHandler: pipelineUsageHandler,
	}
}
