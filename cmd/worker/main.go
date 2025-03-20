package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pipelineworker "github.com/instill-ai/pipeline-backend/pkg/worker"
)

const gracefulShutdownWaitPeriod = 15 * time.Second
const gracefulShutdownTimeout = 60 * time.Minute

// These variables might be overridden at buildtime.
var version = "dev"
var serviceName = "pipeline-backend-worker"

func main() {
	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	// setup tracing and metrics
	ctx, cancel := context.WithCancel(context.Background())

	tp, err := customotel.SetupTracing(ctx, "pipeline-backend-worker")
	if err != nil {
		panic(err)
	}

	defer func() {
		err = tp.Shutdown(ctx)
	}()

	ctx, span := otel.Tracer("worker-tracer").Start(ctx, "main")
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	db := database.GetSharedConnection()
	defer database.Close(db)

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	timeseries := repository.MustNewInfluxDB(ctx)
	defer timeseries.Close()

	minIOParams := minio.ClientParams{
		Config: config.Config.Minio,
		Logger: logger,
		AppInfo: minio.AppInfo{
			Name:    serviceName,
			Version: version,
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

	var temporalClientOptions client.Options
	if config.Config.Temporal.Ca != "" && config.Config.Temporal.Cert != "" && config.Config.Temporal.Key != "" {
		if temporalClientOptions, err = temporal.GetTLSClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger),
			config.Config.Temporal.Ca,
			config.Config.Temporal.Cert,
			config.Config.Temporal.Key,
			config.Config.Temporal.ServerName,
			true,
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	} else {
		if temporalClientOptions, err = temporal.GetClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger)); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	}

	temporalClient, err := client.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer temporalClient.Close()

	// for only local temporal cluster
	if config.Config.Temporal.Ca == "" && config.Config.Temporal.Cert == "" && config.Config.Temporal.Key == "" {
		initTemporalNamespace(ctx, temporalClient)
	}

	repo := repository.NewRepository(db, redisClient)

	pipelinePublicServiceClient, pipelinePublicServiceClientConn := external.InitPipelinePublicServiceClient(ctx)
	if pipelinePublicServiceClientConn != nil {
		defer pipelinePublicServiceClientConn.Close()
	}

	artifactPublicServiceClient, artifactPublicServiceClientConn := external.InitArtifactPublicServiceClient(ctx)
	if artifactPublicServiceClientConn != nil {
		defer artifactPublicServiceClientConn.Close()
	}

	artifactPrivateServiceClient, artifactPrivateServiceClientConn := external.InitArtifactPrivateServiceClient(ctx)
	if artifactPrivateServiceClientConn != nil {
		defer artifactPrivateServiceClientConn.Close()
	}

	binaryFetcher := external.NewArtifactBinaryFetcher(artifactPrivateServiceClient, minIOFileGetter)

	compStore := componentstore.Init(componentstore.InitParams{
		Logger:         logger,
		Secrets:        config.Config.Component.Secrets,
		BinaryFetcher:  binaryFetcher,
		TemporalClient: temporalClient,
	})

	pubsub := pubsub.NewRedisPubSub(redisClient)
	ms := memory.NewStore(pubsub, minIOClient.WithLogger(logger))

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
	})

	w.RegisterWorkflow(cw.TriggerPipelineWorkflow)
	w.RegisterWorkflow(cw.SchedulePipelineWorkflow)

	w.RegisterActivity(cw.LoadWorkflowMemoryActivity)
	w.RegisterActivity(cw.CommitWorkflowMemoryActivity)
	w.RegisterActivity(cw.ComponentActivity)
	w.RegisterActivity(cw.OutputActivity)
	w.RegisterActivity(cw.PreIteratorActivity)
	w.RegisterActivity(cw.PostIteratorActivity)
	w.RegisterActivity(cw.InitComponentsActivity)
	w.RegisterActivity(cw.SendStartedEventActivity)
	w.RegisterActivity(cw.SendCompletedEventActivity)
	w.RegisterActivity(cw.ClosePipelineActivity)
	w.RegisterActivity(cw.PurgeWorkflowMemoryActivity)
	w.RegisterActivity(cw.IncreasePipelineTriggerCountActivity)
	w.RegisterActivity(cw.UpdatePipelineRunActivity)
	w.RegisterActivity(cw.UpsertComponentRunActivity)

	w.RegisterActivity(cw.UploadOutputsToMinIOActivity)
	w.RegisterActivity(cw.UploadRecipeToMinIOActivity)
	w.RegisterActivity(cw.UploadComponentInputsActivity)
	w.RegisterActivity(cw.UploadComponentOutputsActivity)

	span.End()

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
	// in-process memory, which prevents worfklows from recovering from
	// interruptions.
	// To handle this properly, we should make activities independent of
	// the shared memory.
	<-quitSig

	time.Sleep(gracefulShutdownWaitPeriod)

	logger.Info("Shutting down worker...")
	w.Stop()
}

func initTemporalNamespace(ctx context.Context, client client.Client) {
	logger, _ := logger.GetZapLogger(ctx)

	resp, err := client.WorkflowService().ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to list namespaces: %s", err))
	}

	found := false
	for _, n := range resp.GetNamespaces() {
		if n.NamespaceInfo.Name == config.Config.Temporal.Namespace {
			found = true
		}
	}

	if !found {
		if _, err := client.WorkflowService().RegisterNamespace(ctx,
			&workflowservice.RegisterNamespaceRequest{
				Namespace: config.Config.Temporal.Namespace,
				WorkflowExecutionRetentionPeriod: func() *durationpb.Duration {
					// Check if the string ends with "d" for day.
					s := config.Config.Temporal.Retention
					if strings.HasSuffix(s, "d") {
						// Parse the number of days.
						days, err := strconv.Atoi(s[:len(s)-1])
						if err != nil {
							logger.Fatal(fmt.Sprintf("Unable to parse retention period in day: %s", err))
						}
						// Convert days to hours and then to a duration.
						t := time.Hour * 24 * time.Duration(days)
						return durationpb.New(t)
					}
					logger.Fatal(fmt.Sprintf("Unable to parse retention period in day: %s", err))
					return nil
				}(),
			},
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to register namespace: %s", err))
		}
	}
}
