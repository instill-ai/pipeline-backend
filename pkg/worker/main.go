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
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	miniox "github.com/instill-ai/x/minio"
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
	ClosePipelineActivity(ctx context.Context, workflowID string) error
	IncreasePipelineTriggerCountActivity(context.Context, recipe.SystemVariables) error

	UpdatePipelineRunActivity(ctx context.Context, param *UpdatePipelineRunActivityParam) error
	UpsertComponentRunActivity(ctx context.Context, param *UpsertComponentRunActivityParam) error
	UploadInputsToMinioActivity(ctx context.Context, param *UploadInputsToMinioActivityParam) error
	UploadOutputsToMinioActivity(ctx context.Context, param *UploadOutputsToMinioActivityParam) error
	UploadRecipeToMinioActivity(ctx context.Context, param *UploadRecipeToMinioActivityParam) error
	UploadComponentInputsActivity(ctx context.Context, param *ComponentActivityParam) error
	UploadComponentOutputsActivity(ctx context.Context, param *ComponentActivityParam) error
}

// WorkerConfig is the configuration for the worker
type WorkerConfig struct {
	Repository                   repository.Repository
	RedisClient                  *redis.Client
	InfluxDBWriteClient          api.WriteAPI
	Component                    *componentstore.Store
	MinioClient                  miniox.MinioI
	MemoryStore                  memory.MemoryStore
	WorkerUID                    uuid.UUID
	ArtifactPublicServiceClient  artifactpb.ArtifactPublicServiceClient
	ArtifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
}

// worker represents resources required to run Temporal workflow and activity
type worker struct {
	repository                   repository.Repository
	redisClient                  *redis.Client
	influxDBWriteClient          api.WriteAPI
	component                    *componentstore.Store
	minioClient                  miniox.MinioI
	log                          *zap.Logger
	memoryStore                  memory.MemoryStore
	workerUID                    uuid.UUID
	artifactPublicServiceClient  artifactpb.ArtifactPublicServiceClient
	artifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
}

// NewWorker initiates a temporal worker for workflow and activity definition
func NewWorker(
	workerConfig WorkerConfig,
) Worker {
	logger, _ := logger.GetZapLogger(context.Background())
	return &worker{
		repository:                   workerConfig.Repository,
		redisClient:                  workerConfig.RedisClient,
		memoryStore:                  workerConfig.MemoryStore,
		influxDBWriteClient:          workerConfig.InfluxDBWriteClient,
		component:                    workerConfig.Component,
		minioClient:                  workerConfig.MinioClient,
		log:                          logger,
		workerUID:                    workerConfig.WorkerUID,
		artifactPublicServiceClient:  workerConfig.ArtifactPublicServiceClient,
		artifactPrivateServiceClient: workerConfig.ArtifactPrivateServiceClient,
	}
}
