package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/minio"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	componentbase "github.com/instill-ai/component/base"
	componentstore "github.com/instill-ai/component/store"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(ctx workflow.Context, param *TriggerPipelineWorkflowParam) error
	SchedulePipelineWorkflow(ctx workflow.Context, param *SchedulePipelineWorkflowParam) error
	UploadOutputsToMinioWorkflow(ctx workflow.Context, param *UploadOutputsWorkflowParam) error

	ComponentActivity(ctx context.Context, param *ComponentActivityParam) (*ComponentActivityParam, error)
	PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error)
	PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error
	IncreasePipelineTriggerCountActivity(context.Context, recipe.SystemVariables) error
	SchedulePipelineLoaderActivity(ctx context.Context, param *SchedulePipelineLoaderActivityParam) (*SchedulePipelineLoaderActivityResult, error)
	UploadToMinioActivity(ctx context.Context, param *UploadToMinioActivityParam) (string, error)
	UploadInputsToMinioActivity(ctx context.Context, param *UploadInputsToMinioActivityParam) error
	UploadReceiptToMinioActivity(ctx context.Context, param *UploadReceiptToMinioActivityParam) error
	UploadComponentInputsActivity(ctx context.Context, param *ComponentActivityParam) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository          repository.Repository
	redisClient         *redis.Client
	influxDBWriteClient api.WriteAPI
	component           *componentstore.Store
	minioClient         minio.MinioI
	log                 *zap.Logger
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	r repository.Repository,
	rd *redis.Client,
	i api.WriteAPI,
	cs componentstore.ComponentSecrets,
	uh componentbase.UsageHandlerCreator,
	minioClient minio.MinioI,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:          r,
		redisClient:         rd,
		influxDBWriteClient: i,
		component:           componentstore.Init(logger, cs, uh),
		minioClient:         minioClient,
		log:                 logger,
	}
}
