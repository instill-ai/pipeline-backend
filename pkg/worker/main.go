package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
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

	ComponentActivity(ctx context.Context, param *ComponentActivityParam) error
	OutputActivity(ctx context.Context, param *ComponentActivityParam) error
	PreIteratorActivity(ctx context.Context, param *PreIteratorActivityParam) (*PreIteratorActivityResult, error)
	LoadDAGDataActivity(ctx context.Context, param *LoadDAGDataActivityParam) (*LoadDAGDataActivityResult, error)
	PostIteratorActivity(ctx context.Context, param *PostIteratorActivityParam) error
	PreTriggerActivity(ctx context.Context, param *PreTriggerActivityParam) error
	PostTriggerActivity(ctx context.Context, param *PostTriggerActivityParam) error
	ClosePipelineActivity(ctx context.Context, workflowID string, isStreaming bool) error
	IncreasePipelineTriggerCountActivity(context.Context, recipe.SystemVariables) error

	UpdatePipelineRunActivity(ctx context.Context, param *UpdatePipelineRunActivityParam) error
	UpsertComponentRunActivity(ctx context.Context, param *UpsertComponentRunActivityParam) error
	UploadInputsToMinioActivity(ctx context.Context, param *UploadInputsToMinioActivityParam) error
	UploadOutputsToMinioActivity(ctx context.Context, param *UploadOutputsToMinioActivityParam) error
	UploadRecipeToMinioActivity(ctx context.Context, param *UploadRecipeToMinioActivityParam) error
	UploadComponentInputsActivity(ctx context.Context, param *ComponentActivityParam) error
	UploadComponentOutputsActivity(ctx context.Context, param *ComponentActivityParam) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository          repository.Repository
	redisClient         *redis.Client
	influxDBWriteClient api.WriteAPI
	component           *componentstore.Store
	minioClient         minio.MinioI
	log                 *zap.Logger
	memoryStore         memory.MemoryStore
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	r repository.Repository,
	rc *redis.Client,
	i api.WriteAPI,
	cs componentstore.ComponentSecrets,
	uh componentbase.UsageHandlerCreator,
	minioClient minio.MinioI,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	m := memory.NewMemoryStore(rc)
	return &worker{
		repository:          r,
		redisClient:         rc,
		memoryStore:         m,
		influxDBWriteClient: i,
		component:           componentstore.Init(logger, cs, uh),
		minioClient:         minioClient,
		log:                 logger,
	}
}
