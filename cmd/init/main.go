package main

import (
	"context"
	"log"
	"time"

	_ "embed"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/cmd/init/definitionupdater"
	"github.com/instill-ai/pipeline-backend/config"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
)

type PrebuiltConnector struct {
	ID                     string      `json:"id"`
	UID                    string      `json:"uid"`
	Owner                  string      `json:"owner"`
	ConnectorDefinitionUID string      `json:"connector_definition_uid"`
	Configuration          interface{} `json:"configuration"`
	Task                   string      `json:"task"`
}

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// Connector is the data model of the connector table
type Connector struct {
	BaseDynamic
	ID                     string
	Owner                  string
	ConnectorDefinitionUID uuid.UUID
	Description            string
	Tombstone              bool
	Configuration          datatypes.JSON `gorm:"type:jsonb"`
	ConnectorType          string         `sql:"type:string"`
	State                  string         `sql:"type:string"`
	Visibility             string         `sql:"type:string"`
	Task                   string         `sql:"type:string"`
}

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
