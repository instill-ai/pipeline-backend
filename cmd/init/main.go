package main

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"

	"github.com/instill-ai/pipeline-backend/cmd/init/definitionupdater"
	"github.com/instill-ai/pipeline-backend/config"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
)

func main() {
	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
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
}
