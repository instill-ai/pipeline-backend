package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"

	component "github.com/instill-ai/component/pkg/base"
	"github.com/instill-ai/component/pkg/connector"
	"github.com/instill-ai/component/pkg/operator"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowParam) error
	ConnectorActivity(ctx context.Context, param *ConnectorActivityParam) error
	OperatorActivity(ctx context.Context, param *OperatorActivityParam) error
	PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error)
	PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository          repository.Repository
	redisClient         *redis.Client
	influxDBWriteClient api.WriteAPI
	operator            *operator.Store
	connector           *connector.Store
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	r repository.Repository,
	rd *redis.Client,
	i api.WriteAPI,
	cs connector.ConnectionSecrets,
	uh map[string]component.UsageHandlerCreator,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:          r,
		redisClient:         rd,
		influxDBWriteClient: i,
		operator:            operator.Init(logger),
		connector:           connector.Init(logger, cs, uh),
	}
}
