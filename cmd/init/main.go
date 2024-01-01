package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	_ "embed"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	openfgaClient "github.com/openfga/go-sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	connector "github.com/instill-ai/connector/pkg"
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

	// Set tombstone based on definition
	connector := connector.Init(logger, utils.GetConnectorOptions())
	definitions := connector.ListConnectorDefinitions()
	for idx := range definitions {
		if definitions[idx].Tombstone {
			db.Unscoped().Model(&datamodel.Connector{}).Where("connector_definition_uid = ?", definitions[idx].Uid).Update("tombstone", true)
		}
	}

	fgaClient, err := openfgaClient.NewSdkClient(&openfgaClient.ClientConfiguration{
		ApiScheme: "http",
		ApiHost:   fmt.Sprintf("%s:%d", config.Config.OpenFGA.Host, config.Config.OpenFGA.Port),
	})

	if err != nil {
		panic(err)
	}

	var aclClient acl.ACLClient
	if stores, err := fgaClient.ListStores(context.Background()).Execute(); err == nil {
		fgaClient.SetStoreId(*(*stores.Stores)[0].Id)
		if models, err := fgaClient.ReadAuthorizationModels(context.Background()).Execute(); err == nil {
			aclClient = acl.NewACLClient(fgaClient, (*models.AuthorizationModels)[0].Id)
		}
		if err != nil {
			panic(err)
		}

	} else {
		panic(err)
	}

	var pipelines []*datamodel.Pipeline
	pageToken := ""
	for {
		pipelines, _, pageToken, err = repository.ListPipelinesAdmin(context.Background(), 100, pageToken, true, filtering.Filter{}, false)
		if err != nil {
			panic(err)
		}
		for _, pipeline := range pipelines {
			nsType := strings.Split(pipeline.Owner, "/")[0]
			nsType = nsType[0 : len(nsType)-1]
			userUID, err := uuid.FromString(strings.Split(pipeline.Owner, "/")[1])
			if err != nil {
				panic(err)
			}
			err = aclClient.SetOwner("pipeline", pipeline.UID, nsType, userUID)
			if err != nil {
				panic(err)
			}
		}
		if pageToken == "" {
			break
		}
	}

	var connectors []*datamodel.Connector
	pageToken = ""
	for {
		connectors, _, pageToken, err = repository.ListConnectorsAdmin(context.Background(), 100, pageToken, true, filtering.Filter{}, false)
		if err != nil {
			panic(err)
		}
		for _, connector := range connectors {
			nsType := strings.Split(connector.Owner, "/")[0]
			nsType = nsType[0 : len(nsType)-1]
			userUID, err := uuid.FromString(strings.Split(connector.Owner, "/")[1])
			if err != nil {
				panic(err)
			}
			err = aclClient.SetOwner("connector", connector.UID, nsType, userUID)
			if err != nil {
				panic(err)
			}
		}
		if pageToken == "" {
			break
		}
	}

}
