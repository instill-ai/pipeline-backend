package worker

import (
	"context"

	"github.com/go-redis/redis/v9"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	"go.temporal.io/sdk/workflow"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

type exitCode int64

const (
	exitCodeOK exitCode = iota
	exitCodeError
)

// Worker interface
type Worker interface {
	TriggerAsyncPipelineWorkflow(ctx workflow.Context, param *TriggerAsyncPipelineWorkflowParam) ([][]byte, error)
	TriggerAsyncPipelineByFileUploadWorkflow(ctx workflow.Context, param *TriggerAsyncPipelineByFileUploadWorkflowParam) ([][]byte, error)
	TriggerActivity(ctx context.Context, param *TriggerActivityParam) ([]byte, error)
	TriggerByFileUploadActivity(ctx context.Context, param *TriggerByFileUploadActivityParam) ([]byte, error)
	DestinationActivity(ctx context.Context, param *DestinationActivityParam) ([]byte, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	modelPublicServiceClient     modelPB.ModelPublicServiceClient
	connectorPublicServiceClient connectorPB.ConnectorPublicServiceClient
	redisClient                  *redis.Client
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(m modelPB.ModelPublicServiceClient, c connectorPB.ConnectorPublicServiceClient, r *redis.Client) Worker {

	return &worker{
		modelPublicServiceClient:     m,
		connectorPublicServiceClient: c,
		redisClient:                  r,
	}
}
