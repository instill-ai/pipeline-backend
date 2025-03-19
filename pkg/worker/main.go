package worker

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/minio"

	"github.com/instill-ai/pipeline-backend/pkg/component/generic/scheduler/v0"
	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

// TaskQueue is the Temporal task queue name for pipeline-backend
const TaskQueue = "pipeline-backend"

// Worker interface
type Worker interface {
	TriggerPipelineWorkflow(workflow.Context, *TriggerPipelineWorkflowParam) error
	SchedulePipelineWorkflow(workflow.Context, *scheduler.SchedulePipelineWorkflowParam) error

	LoadWorkflowMemory(context.Context, LoadWorkflowMemoryActivityParam) error
	ComponentActivity(context.Context, *ComponentActivityParam) error
	OutputActivity(context.Context, *ComponentActivityParam) error
	PreIteratorActivity(context.Context, *PreIteratorActivityParam) ([]ChildPipelineTriggerParams, error)
	PostIteratorActivity(context.Context, *PostIteratorActivityParam) error
	InitComponentsActivity(context.Context, *InitComponentsActivityParam) error
	SendStartedEventActivity(_ context.Context, workflowID string) error
	PostTriggerActivity(context.Context, *PostTriggerActivityParam) error
	ClosePipelineActivity(_ context.Context, workflowID string) error
	IncreasePipelineTriggerCountActivity(context.Context, recipe.SystemVariables) error

	UpdatePipelineRunActivity(context.Context, *UpdatePipelineRunActivityParam) error
	UpsertComponentRunActivity(context.Context, *UpsertComponentRunActivityParam) error
	UploadOutputsToMinIOActivity(context.Context, *MinIOUploadMetadata) error
	UploadRecipeToMinIOActivity(context.Context, UploadRecipeToMinIOParam) error
	UploadComponentInputsActivity(context.Context, *ComponentActivityParam) error
	UploadComponentOutputsActivity(context.Context, *ComponentActivityParam) error
}

// WorkerConfig is the configuration for the worker
type WorkerConfig struct {
	Repository                   repository.Repository
	RedisClient                  *redis.Client
	InfluxDBWriteClient          api.WriteAPI
	Component                    *componentstore.Store
	MinioClient                  minio.Client
	MemoryStore                  *memory.Store
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
	minioClient                  minio.Client
	log                          *zap.Logger
	memoryStore                  *memory.Store
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
		artifactPublicServiceClient:  workerConfig.ArtifactPublicServiceClient,
		artifactPrivateServiceClient: workerConfig.ArtifactPrivateServiceClient,
		binaryFetcher:                workerConfig.BinaryFetcher,
		pipelinePublicServiceClient:  workerConfig.PipelinePublicServiceClient,
	}
}
