package main

import (
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/go-redis/redis/v9"
	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	pipelineWorker "github.com/instill-ai/pipeline-backend/pkg/worker"
)

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	db := database.GetConnection()
	defer database.Close(db)

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

	modelPublicServiceClient, modelPublicServiceClientConn := external.InitModelPublicServiceClient()
	if modelPublicServiceClientConn != nil {
		defer modelPublicServiceClientConn.Close()
	}

	connectorPublicServiceClient, connectorPublicServiceClientConn := external.InitConnectorPublicServiceClient()
	if connectorPublicServiceClientConn != nil {
		defer connectorPublicServiceClientConn.Close()
	}

	cw := pipelineWorker.NewWorker(modelPublicServiceClient, connectorPublicServiceClient, redisClient)

	w := worker.New(temporalClient, pipelineWorker.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: 2,
	})

	w.RegisterWorkflow(cw.TriggerAsyncPipelineWorkflow)
	w.RegisterWorkflow(cw.TriggerAsyncPipelineByFileUploadWorkflow)
	w.RegisterActivity(cw.TriggerActivity)
	w.RegisterActivity(cw.TriggerByFileUploadActivity)
	w.RegisterActivity(cw.DestinationActivity)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
}
