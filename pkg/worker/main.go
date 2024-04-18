package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	component "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/component/pkg/connector"
	operator "github.com/instill-ai/component/pkg/operator"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowParam) error
	ConnectorActivity(ctx context.Context, param *ConnectorActivityParam) error
	OperatorActivity(ctx context.Context, param *OperatorActivityParam) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository          repository.Repository
	redisClient         *redis.Client
	influxDBWriteClient api.WriteAPI
	operator            component.IOperator
	connector           component.IConnector
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(r repository.Repository, rd *redis.Client, i api.WriteAPI, u component.UsageHandler) Worker {

	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:          r,
		redisClient:         rd,
		influxDBWriteClient: i,
		operator:            operator.Init(logger, u),
		connector:           connector.Init(logger, u, utils.GetConnectorOptions()),
	}
}
