package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	_ "embed"

	"github.com/gofrs/uuid"
	"github.com/launchdarkly/go-semver"
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
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"

	connector "github.com/instill-ai/connector/pkg"
	operator "github.com/instill-ai/operator/pkg"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
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

	repo := repository.NewRepository(db)

	// Update component definitions and connectors based on latest version of
	// definitions.json.
	connDefs := connector.Init(logger, utils.GetConnectorOptions()).ListConnectorDefinitions()
	for _, connDef := range connDefs {
		if connDef.Tombstone {
			db.Unscoped().Model(&datamodel.Connector{}).Where("connector_definition_uid = ?", connDef.Uid).Update("tombstone", true)
		}

		cd := &pipelinePB.ComponentDefinition{
			Type: service.ConnectorTypeToComponentType[connDef.Type],
			Definition: &pipelinePB.ComponentDefinition_ConnectorDefinition{
				ConnectorDefinition: connDef,
			},
		}

		if err := updateComponentDefinition(ctx, cd, repo); err != nil {
			log.Fatal(err)
		}
	}

	opDefs := operator.Init(logger).ListOperatorDefinitions()
	for _, opDef := range opDefs {
		cd := &pipelinePB.ComponentDefinition{
			Type: pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR,
			Definition: &pipelinePB.ComponentDefinition_OperatorDefinition{
				OperatorDefinition: opDef,
			},
		}

		if err := updateComponentDefinition(ctx, cd, repo); err != nil {
			log.Fatal(err)
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
		pipelines, _, pageToken, err = repo.ListPipelinesAdmin(context.Background(), 100, pageToken, true, filtering.Filter{}, false)
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
		connectors, _, pageToken, err = repo.ListConnectorsAdmin(context.Background(), 100, pageToken, true, filtering.Filter{}, false)
		if err != nil {
			panic(err)
		}
		for _, conn := range connectors {
			nsType := strings.Split(conn.Owner, "/")[0]
			nsType = nsType[0 : len(nsType)-1]
			userUID, err := uuid.FromString(strings.Split(conn.Owner, "/")[1])
			if err != nil {
				panic(err)
			}
			err = aclClient.SetOwner("connector", conn.UID, nsType, userUID)
			if err != nil {
				panic(err)
			}
		}
		if pageToken == "" {
			break
		}
	}

}

func updateComponentDefinition(ctx context.Context, cd *pipelinePB.ComponentDefinition, repo repository.Repository) error {
	var id, uniqueID, version string
	switch cd.Type {
	case pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR:
		d := cd.GetOperatorDefinition()
		id, uniqueID, version = d.GetId(), d.GetUid(), d.GetVersion()

	case pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
		pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
		pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION:

		d := cd.GetConnectorDefinition()
		id, uniqueID, version = d.GetId(), d.GetUid(), d.GetVersion()
	default:
		return fmt.Errorf("unsupported component definition type")
	}

	uid, err := uuid.FromString(uniqueID)
	if err != nil {
		return fmt.Errorf("invalid UID in component definition %s: %w", id, err)
	}

	v, err := semver.Parse(version)
	if err != nil {
		return fmt.Errorf("failed to parse version from component definition %s: %w", id, err)
	}

	inDB, err := repo.GetComponentDefinitionByUID(ctx, uid)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("error fetching component definition %s from DB: %w", id, err)
	}

	// Component definitions are only updated when there's a version bump.
	if inDB != nil {
		vInDB, err := semver.Parse(inDB.Version)
		if err != nil {
			return fmt.Errorf("failed to parse version from DB component definition %s: %w", id, err)
		}

		if v.ComparePrecedence(vInDB) <= 0 {
			return nil
		}
	}

	if err := repo.UpsertComponentDefinition(ctx, cd); err != nil {
		return fmt.Errorf("failed to upsert component definition %s: %w", id, err)
	}

	return nil
}
