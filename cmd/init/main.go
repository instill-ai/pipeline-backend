package main

import (
	"context"
	"log"
	"time"

	_ "embed"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	connector "github.com/instill-ai/connector/pkg"
	connectorAirbyte "github.com/instill-ai/connector/pkg/airbyte"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

type PrebuiltConnector struct {
	Id                     string      `json:"id"`
	Uid                    string      `json:"uid"`
	Owner                  string      `json:"owner"`
	ConnectorDefinitionUid string      `json:"connector_definition_uid"`
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

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)

	db := database.GetConnection()
	defer database.Close(db)

	repository := repository.NewRepository(db)

	airbyte := connectorAirbyte.Init(logger, utils.GetConnectorOptions().Airbyte)

	// TODO: use pagination
	conns, _, _, err := repository.ListConnectorsAdmin(ctx, 1000, "", false, filtering.Filter{}, false)
	if err != nil {
		panic(err)
	}

	airbyteConnector := airbyte.(*connectorAirbyte.Connector)
	var uids []uuid.UUID
	for idx := range conns {
		uid := conns[idx].ConnectorDefinitionUID
		if _, err = airbyteConnector.GetConnectorDefinitionByUID(uid); err == nil {
			uids = append(uids, uid)

		}
	}

	err = airbyteConnector.PreDownloadImage(logger, uids)

	if err != nil {
		panic(err)
	}

	// Set tombstone based on definition
	connectors := connector.Init(logger, utils.GetConnectorOptions())
	definitions := connectors.ListConnectorDefinitions()
	for idx := range definitions {
		if definitions[idx].Tombstone {
			db.Unscoped().Model(&datamodel.Connector{}).Where("connector_definition_uid = ?", definitions[idx].Uid).Update("tombstone", true)
		}
	}

}
