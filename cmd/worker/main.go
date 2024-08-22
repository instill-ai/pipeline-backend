package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/minio"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pipelineworker "github.com/instill-ai/pipeline-backend/pkg/worker"
)

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

	cw := pipelineworker.NewWorker(
		repo,
		redisClient,
		timeseries.WriteAPI(),
		config.Config.Connector.Secrets,
		nil,
		minioClient,
	)

	w := worker.New(temporalClient, pipelineworker.TaskQueue, worker.Options{
		EnableSessionWorker:               true,
		WorkflowPanicPolicy:               worker.FailWorkflow,
		MaxConcurrentSessionExecutionSize: 1000,
	})

	w.RegisterWorkflow(cw.TriggerPipelineWorkflow)
	w.RegisterWorkflow(cw.SchedulePipelineWorkflow)

	w.RegisterActivity(cw.ComponentActivity)
	w.RegisterActivity(cw.OutputActivity)
	w.RegisterActivity(cw.PreIteratorActivity)
	w.RegisterActivity(cw.PostIteratorActivity)
	w.RegisterActivity(cw.CloneToMemoryStoreActivity)
	w.RegisterActivity(cw.CloneToRedisActivity)
	w.RegisterActivity(cw.IncreasePipelineTriggerCountActivity)
	w.RegisterActivity(cw.UploadToMinioActivity)
	w.RegisterActivity(cw.UploadInputsToMinioActivity)
	w.RegisterActivity(cw.UploadOutputsToMinioActivity)
	w.RegisterActivity(cw.UploadRecipeToMinioActivity)
	w.RegisterActivity(cw.UploadComponentInputsActivity)
	w.RegisterActivity(cw.UploadComponentOutputsActivity)

	span.End()
	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
}
