package worker

import (
	"context"

	"github.com/go-redis/redis/v9"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	component "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/connector/pkg"
	operator "github.com/instill-ai/operator/pkg"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowRequest) (*TriggerPipelineWorkflowResponse, error)
	ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteConnectorActivityResponse, error)
	OperatorActivity(ctx context.Context, param *ExecuteOperatorActivityRequest) (*ExecuteOperatorActivityResponse, error)
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
func NewWorker(r repository.Repository, rd *redis.Client, i api.WriteAPI) Worker {

	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:          r,
		redisClient:         rd,
		influxDBWriteClient: i,
		operator:            operator.Init(logger),
		connector:           connector.Init(logger, utils.GetConnectorOptions()),
	}
}
