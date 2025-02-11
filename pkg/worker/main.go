package worker

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	"github.com/instill-ai/pipeline-backend/pkg/component/generic/scheduler/v0"
	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	miniox "github.com/instill-ai/x/minio"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(workflow.Context, *TriggerPipelineWorkflowParam) error
	SchedulePipelineWorkflow(workflow.Context, *scheduler.SchedulePipelineWorkflowParam) error

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
	UploadOutputsToMinIOActivity(context.Context, *MinIOUploadMetadata) error
	UploadRecipeToMinIOActivity(context.Context, *MinIOUploadMetadata) error
	UploadComponentInputsActivity(context.Context, *ComponentActivityParam) error
	UploadComponentOutputsActivity(context.Context, *ComponentActivityParam) error
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
	BinaryFetcher                external.BinaryFetcher
	PipelinePublicServiceClient  pb.PipelinePublicServiceClient
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
	pipelinePublicServiceClient  pb.PipelinePublicServiceClient
	binaryFetcher                external.BinaryFetcher
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
		binaryFetcher:                workerConfig.BinaryFetcher,
		pipelinePublicServiceClient:  workerConfig.PipelinePublicServiceClient,
	}
}
