package worker

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	miniox "github.com/instill-ai/x/minio"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(workflow.Context, *TriggerPipelineWorkflowParam) error
	SchedulePipelineWorkflow(workflow.Context, *SchedulePipelineWorkflowParam) error

	ComponentActivity(context.Context, *ComponentActivityParam) error
	OutputActivity(context.Context, *ComponentActivityParam) error
	PreIteratorActivity(context.Context, *PreIteratorActivityParam) (*PreIteratorActivityResult, error)
	LoadDAGDataActivity(_ context.Context, workflowID string) (*LoadDAGDataActivityResult, error)
	PostIteratorActivity(context.Context, *PostIteratorActivityParam) error
	LoadRecipeActivity(context.Context, *LoadRecipeActivityParam) error
	InitComponentsActivity(context.Context, *InitComponentsActivityParam) error
	SendStartedEventActivity(_ context.Context, workflowID string) error
	PostTriggerActivity(context.Context, *PostTriggerActivityParam) error
	ClosePipelineActivity(_ context.Context, workflowID string) error
	IncreasePipelineTriggerCountActivity(context.Context, recipe.SystemVariables) error

	UpdatePipelineRunActivity(context.Context, *UpdatePipelineRunActivityParam) error
	UpsertComponentRunActivity(context.Context, *UpsertComponentRunActivityParam) error
	UploadInputsToMinioActivity(context.Context, *UploadInputsToMinioActivityParam) error
	UploadOutputsToMinioActivity(context.Context, *UploadOutputsToMinioActivityParam) error
	UploadRecipeToMinioActivity(context.Context, *UploadRecipeToMinioActivityParam) error
	UploadComponentInputsActivity(context.Context, *ComponentActivityParam) error
	UploadComponentOutputsActivity(context.Context, *ComponentActivityParam) error
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository          repository.Repository
	redisClient         *redis.Client
	influxDBWriteClient api.WriteAPI
	component           *componentstore.Store
	minioClient         miniox.MinioI
	log                 *zap.Logger
	memoryStore         memory.MemoryStore
	workerUID           uuid.UUID
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	r repository.Repository,
	rc *redis.Client,
	i api.WriteAPI,
	cs *componentstore.Store,
	minioClient miniox.MinioI,
	m memory.MemoryStore,
	workerUID uuid.UUID,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:          r,
		redisClient:         rc,
		memoryStore:         m,
		influxDBWriteClient: i,
		component:           cs,
		minioClient:         minioClient,
		log:                 logger,
		workerUID:           workerUID,
	}
}
