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

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/minio"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pipelineworker "github.com/instill-ai/pipeline-backend/pkg/worker"

	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"
)

const gracefulShutdownTimeout = 60 * time.Minute

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
				WorkflowExecutionRetentionPeriod: func() *time.Duration {
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
						return &t
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

func main() {
	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	// setup tracing and metrics
	if tp, err := customotel.SetupTracing(ctx, "pipeline-backend-worker"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = tp.Shutdown(ctx)
		}()
	}

	ctx, span := otel.Tracer("worker-tracer").Start(ctx,
		"main",
	)
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

	repo := repository.NewRepository(db, redisClient)

	var err error

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

	timeseries := repository.MustNewInfluxDB(ctx)
	defer timeseries.Close()

	// Initialize Minio client
	minioClient, err := minio.NewMinioClientAndInitBucket(ctx, &config.Config.Minio)
	if err != nil {
		logger.Fatal("failed to create minio client", zap.Error(err))
	}

	memory := memory.NewMemoryStore(redisClient)

	queue := uuid.New().String()
	cw := pipelineworker.NewWorker(
		repo,
		redisClient,
		timeseries.WriteAPI(),
		config.Config.Connector.Secrets,
		nil,
		minioClient,
		memory,
		queue,
	)

	w := worker.New(temporalClient, pipelineworker.TaskQueue, worker.Options{
		WorkflowPanicPolicy: worker.BlockWorkflow,
		WorkerStopTimeout:   gracefulShutdownTimeout,
	})

	lw := worker.New(temporalClient, queue, worker.Options{
		WorkflowPanicPolicy: worker.BlockWorkflow,
		WorkerStopTimeout:   gracefulShutdownTimeout,
	})

	w.RegisterWorkflow(cw.TriggerPipelineWorkflow)
	w.RegisterWorkflow(cw.SchedulePipelineWorkflow)

	lw.RegisterActivity(cw.ComponentActivity)
	lw.RegisterActivity(cw.OutputActivity)
	lw.RegisterActivity(cw.PreIteratorActivity)
	lw.RegisterActivity(cw.PostIteratorActivity)
	lw.RegisterActivity(cw.PreTriggerActivity)
	lw.RegisterActivity(cw.LoadDAGDataActivity)
	lw.RegisterActivity(cw.PostTriggerActivity)
	lw.RegisterActivity(cw.IncreasePipelineTriggerCountActivity)
	lw.RegisterActivity(cw.UpdatePipelineRunActivity)
	lw.RegisterActivity(cw.UpsertComponentRunActivity)
	lw.RegisterActivity(cw.UploadInputsToMinioActivity)
	lw.RegisterActivity(cw.UploadOutputsToMinioActivity)
	lw.RegisterActivity(cw.UploadRecipeToMinioActivity)
	lw.RegisterActivity(cw.UploadComponentInputsActivity)
	lw.RegisterActivity(cw.UploadComponentOutputsActivity)

	span.End()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer shutdownCancel()
	quitSig := make(chan os.Signal, 1)
	errSig := make(chan error)

	err = w.Start()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
	err = lw.Start()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errSig:
		logger.Error(fmt.Sprintf("Fatal error: %v\n", err))
	case <-quitSig:
		logger.Info("Shutting down worker...")
		w.Stop()
		lw.Stop()

		// If the worker is terminated while some workflow memory hasn't been
		// synced to Redis, we’ll handle syncing it back to Redis here.
		_ = memory.SyncWorkflowMemoriesToRedis(shutdownCtx)
	}
}
