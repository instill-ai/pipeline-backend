package main

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/go-redis/redis/v9"
	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	pipelineWorker "github.com/instill-ai/pipeline-backend/pkg/worker"
)

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	// setup tracing and metrics
	if tp, err := custom_otel.SetupTracing(ctx, "pipeline-backend-worker"); err != nil {
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
	repository := repository.NewRepository(db)

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

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	influxDBClient, influxDBWriteClient := external.InitInfluxDBServiceClient(ctx)
	defer influxDBClient.Close()

	influxErrCh := influxDBWriteClient.Errors()
	go func() {
		for err := range influxErrCh {
			logger.Error(fmt.Sprintf("write to bucket %s error: %s\n", config.Config.InfluxDB.Bucket, err.Error()))
		}
	}()

	cw := pipelineWorker.NewWorker(repository, redisClient, influxDBWriteClient)

	w := worker.New(temporalClient, pipelineWorker.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 2,
	})

	w.RegisterWorkflow(cw.TriggerPipelineWorkflow)
	w.RegisterActivity(cw.ConnectorActivity)
	w.RegisterActivity(cw.OperatorActivity)

	span.End()
	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
}
