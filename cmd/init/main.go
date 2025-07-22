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
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/durationpb"

	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/cmd/init/definitionupdater"
	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/temporal"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	logx "github.com/instill-ai/x/log"
)

var serviceName = "pipeline-backend-init"

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db := database.GetConnection()
	defer database.Close(db)

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	// This is a workaround solution for the Instill Model connector in Instill Cloud to improve response speed.
	_ = redisClient.Del(ctx, "instill_model_connector_def")

	repo := repository.NewRepository(db, redisClient)
	if err := definitionupdater.UpdateComponentDefinitionIndex(ctx, repo); err != nil {
		log.Fatal(err)
	}

	logger, _ := logx.GetZapLogger(ctx)

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
	// for only local temporal cluster
	if config.Config.Temporal.ServerRootCA == "" && config.Config.Temporal.ClientCert == "" && config.Config.Temporal.ClientKey == "" {
		initTemporalNamespace(ctx, temporalClient)
	}

}

func initTemporalNamespace(ctx context.Context, client temporalclient.Client) {
	logger, _ := logx.GetZapLogger(ctx)

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
