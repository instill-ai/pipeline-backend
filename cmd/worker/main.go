package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"gorm.io/gorm"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/temporal"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	pipelineworker "github.com/instill-ai/pipeline-backend/pkg/worker"
	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
	clientgrpcx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
	otelx "github.com/instill-ai/x/otel"
)

const gracefulShutdownWaitPeriod = 15 * time.Second
const gracefulShutdownTimeout = 60 * time.Minute

var (
	// These variables might be overridden at buildtime.
	serviceName    = "pipeline-backend-worker"
	serviceVersion = "dev"
)

func main() {
	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup all OpenTelemetry components
	cleanup := otelx.SetupWithCleanup(ctx,
		otelx.WithServiceName(serviceName),
		otelx.WithServiceVersion(serviceVersion),
		otelx.WithHost(config.Config.OTELCollector.Host),
		otelx.WithPort(config.Config.OTELCollector.Port),
		otelx.WithCollectorEnable(config.Config.OTELCollector.Enable),
	)
	defer cleanup()

	logx.Debug = config.Config.Server.Debug
	logger, _ := logx.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	// Set gRPC logging based on debug mode
	if config.Config.Server.Debug {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 0) // All logs
	} else {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 3) // verbosity 3 will avoid [transport] from emitting
	}

	// Initialize all clients
	pipelinePublicServiceClient, artifactPublicServiceClient, artifactPrivateServiceClient,
		redisClient, db, minIOClient, minIOFileGetter, temporalClient, timeseries, closeClients := newClients(ctx, logger)
	defer closeClients()

	// Keep NewArtifactBinaryFetcher as requested
	binaryFetcher := external.NewArtifactBinaryFetcher(artifactPrivateServiceClient, minIOFileGetter)

	compStore := componentstore.Init(componentstore.InitParams{
		Logger:         logger,
		Secrets:        config.Config.Component.Secrets,
		BinaryFetcher:  binaryFetcher,
		TemporalClient: temporalClient,
	})

	pubsub := pubsub.NewRedisPubSub(redisClient)
	ms := memory.NewStore(pubsub, minIOClient.WithLogger(logger))

	repo := repository.NewRepository(db, redisClient)

	cw := pipelineworker.NewWorker(
		pipelineworker.WorkerConfig{
			Repository:                   repo,
			RedisClient:                  redisClient,
			InfluxDBWriteClient:          timeseries.WriteAPI(),
			Component:                    compStore,
			MinioClient:                  minIOClient,
			MemoryStore:                  ms,
			ArtifactPublicServiceClient:  artifactPublicServiceClient,
			ArtifactPrivateServiceClient: artifactPrivateServiceClient,
			BinaryFetcher:                binaryFetcher,
			PipelinePublicServiceClient:  pipelinePublicServiceClient,
		},
	)

	w := worker.New(temporalClient, pipelineworker.TaskQueue, worker.Options{
		EnableSessionWorker:                    true,
		WorkflowPanicPolicy:                    worker.BlockWorkflow,
		WorkerStopTimeout:                      gracefulShutdownTimeout,
		MaxConcurrentWorkflowTaskExecutionSize: 100,
		Interceptors: func() []interceptor.WorkerInterceptor {
			if !config.Config.OTELCollector.Enable {
				return nil
			}
			workerInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
				Tracer:            otel.Tracer(serviceName),
				TextMapPropagator: otel.GetTextMapPropagator(),
			})
			if err != nil {
				logger.Fatal("Unable to create worker tracing interceptor", zap.Error(err))
			}
			return []interceptor.WorkerInterceptor{workerInterceptor}
		}(),
	})

	w.RegisterWorkflow(cw.TriggerPipelineWorkflow)
	w.RegisterWorkflow(cw.SchedulePipelineWorkflow)
	w.RegisterWorkflow(cw.CleanupMemoryWorkflow)

	w.RegisterActivity(cw.LoadWorkflowMemoryActivity)
	w.RegisterActivity(cw.CommitWorkflowMemoryActivity)
	w.RegisterActivity(cw.PurgeWorkflowMemoryActivity)
	w.RegisterActivity(cw.CleanupWorkflowMemoryActivity)

	w.RegisterActivity(cw.ProcessBatchConditionsActivity)
	w.RegisterActivity(cw.ComponentActivity)
	w.RegisterActivity(cw.OutputActivity)
	w.RegisterActivity(cw.PreIteratorActivity)
	w.RegisterActivity(cw.PostIteratorActivity)
	w.RegisterActivity(cw.InitComponentsActivity)
	w.RegisterActivity(cw.SendStartedEventActivity)
	w.RegisterActivity(cw.SendCompletedEventActivity)
	w.RegisterActivity(cw.ClosePipelineActivity)
	w.RegisterActivity(cw.IncreasePipelineTriggerCountActivity)
	w.RegisterActivity(cw.UpdatePipelineRunActivity)
	w.RegisterActivity(cw.UpsertComponentRunActivity)

	w.RegisterActivity(cw.UploadOutputsToMinIOActivity)
	w.RegisterActivity(cw.UploadRecipeToMinIOActivity)
	w.RegisterActivity(cw.UploadComponentInputsActivity)
	w.RegisterActivity(cw.UploadComponentOutputsActivity)

	if err := w.Start(); err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}

	logger.Info("worker is running.")

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	quitSig := make(chan os.Signal, 1)
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)

	// When the server receives a SIGTERM, we'll try to finish ongoing
	// workflows. This is because pipeline trigger activities use a shared
	// in-process memory, which prevents workflows from recovering from
	// interruptions.
	// To handle this properly, we should make activities independent of
	// the shared memory.
	<-quitSig

	time.Sleep(gracefulShutdownWaitPeriod)

	logger.Info("Shutting down worker...")
	w.Stop()
}

func newClients(ctx context.Context, logger *zap.Logger) (
	pipelinepb.PipelinePublicServiceClient,
	artifactpb.ArtifactPublicServiceClient,
	artifactpb.ArtifactPrivateServiceClient,
	*redis.Client,
	*gorm.DB,
	minio.Client,
	*minio.FileGetter,
	temporalclient.Client,
	*repository.InfluxDB,
	func(),
) {
	closeFuncs := map[string]func() error{}

	// Initialize external service clients
	pipelinePublicServiceClient, pipelinePublicClose, err := clientgrpcx.NewClient[pipelinepb.PipelinePublicServiceClient](
		clientgrpcx.WithServiceConfig(client.ServiceConfig{
			Host:       config.Config.Server.InstanceID,
			PublicPort: config.Config.Server.PublicPort,
			HTTPS: client.HTTPSConfig{
				Cert: config.Config.Server.HTTPS.Cert,
				Key:  config.Config.Server.HTTPS.Key,
			},
		}),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create pipeline public service client", zap.Error(err))
	}
	closeFuncs["pipelinePublic"] = pipelinePublicClose

	artifactPublicServiceClient, artifactPublicClose, err := clientgrpcx.NewClient[artifactpb.ArtifactPublicServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.ArtifactBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create artifact public service client", zap.Error(err))
	}
	closeFuncs["artifactPublic"] = artifactPublicClose

	artifactPrivateServiceClient, artifactPrivateClose, err := clientgrpcx.NewClient[artifactpb.ArtifactPrivateServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.ArtifactBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create artifact private service client", zap.Error(err))
	}
	closeFuncs["artifactPrivate"] = artifactPrivateClose

	// Initialize database
	db := database.GetSharedConnection()
	closeFuncs["database"] = func() error {
		database.Close(db)
		return nil
	}

	// Initialize redis client
	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	closeFuncs["redis"] = redisClient.Close

	// Initialize InfluxDB
	timeseries := repository.MustNewInfluxDB(ctx)
	closeFuncs["influxDB"] = func() error {
		timeseries.Close()
		return nil
	}

	// Initialize Temporal client
	temporalClientOptions, err := temporal.ClientOptions(config.Config.Temporal, logger)
	if err != nil {
		logger.Fatal("Unable to build Temporal client options", zap.Error(err))
	}

	// Only add interceptor if tracing is enabled
	if config.Config.OTELCollector.Enable {
		temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
			Tracer:            otel.Tracer(serviceName),
			TextMapPropagator: otel.GetTextMapPropagator(),
		})
		if err != nil {
			logger.Fatal("Unable to create temporal tracing interceptor", zap.Error(err))
		}
		temporalClientOptions.Interceptors = []interceptor.ClientInterceptor{temporalTracingInterceptor}
	}

	temporalClient, err := temporalclient.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal("Unable to create Temporal client", zap.Error(err))
	}
	closeFuncs["temporal"] = func() error {
		temporalClient.Close()
		return nil
	}

	// Initialize MinIO client
	minIOParams := minio.ClientParams{
		Config: config.Config.Minio,
		Logger: logger,
		AppInfo: minio.AppInfo{
			Name:    serviceName,
			Version: serviceVersion,
		},
	}
	minIOFileGetter, err := minio.NewFileGetter(minIOParams)
	if err != nil {
		logger.Fatal("Failed to create MinIO file getter", zap.Error(err))
	}

	retentionHandler := service.NewRetentionHandler()
	minIOParams.ExpiryRules = retentionHandler.ListExpiryRules()
	minIOClient, err := minio.NewMinIOClientAndInitBucket(ctx, minIOParams)
	if err != nil {
		logger.Fatal("failed to create MinIO client", zap.Error(err))
	}

	closer := func() {
		for conn, fn := range closeFuncs {
			if err := fn(); err != nil {
				logger.Error("Failed to close conn", zap.Error(err), zap.String("conn", conn))
			}
		}
	}

	return pipelinePublicServiceClient, artifactPublicServiceClient, artifactPrivateServiceClient,
		redisClient, db, minIOClient, minIOFileGetter, temporalClient, timeseries, closer
}
