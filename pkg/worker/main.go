package worker

import (
	"context"
	"github.com/instill-ai/pipeline-backend/pkg/pipelinelogger"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
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
	ComponentActivity(ctx context.Context, param *ComponentActivityParam) (*ComponentActivityParam, error)
	PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error)
	PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error
	IncreasePipelineTriggerCountActivity(context.Context, recipe.SystemVariables) error
	SchedulePipelineLoaderActivity(ctx context.Context, param *SchedulePipelineLoaderActivityParam) (*SchedulePipelineLoaderActivityResult, error)
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository          repository.Repository
	redisClient         *redis.Client
	influxDBWriteClient api.WriteAPI
	component           *componentstore.Store
	pipelineLogger      *pipelinelogger.PipelineLogger
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	r repository.Repository,
	rd *redis.Client,
	i api.WriteAPI,
	cs componentstore.ComponentSecrets,
	uh componentbase.UsageHandlerCreator,
	db *gorm.DB,
	minioClient *minio.Client,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:          r,
		redisClient:         rd,
		influxDBWriteClient: i,
		component:           componentstore.Init(logger, cs, uh),
		pipelineLogger:      pipelinelogger.NewPipelineLogger(db, minioClient),
	}
}
