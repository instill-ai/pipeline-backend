package worker

import (
	"context"

	"github.com/go-redis/redis/v9"
	"go.temporal.io/sdk/workflow"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerAsyncPipelineWorkflow(ctx workflow.Context, param *TriggerAsyncPipelineWorkflowRequest) error
	ConnectorActivity(ctx context.Context, param *ExecuteConnectorActivityRequest) (*ExecuteConnectorActivityResponse, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	connectorPublicServiceClient connectorPB.ConnectorPublicServiceClient
	redisClient                  *redis.Client
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(c connectorPB.ConnectorPublicServiceClient, r *redis.Client) Worker {

	return &worker{
		connectorPublicServiceClient: c,
		redisClient:                  r,
	}
}
