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
	"github.com/redis/go-redis/v9"
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
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"

	connector "github.com/instill-ai/component/pkg/connector"
	operator "github.com/instill-ai/component/pkg/operator"
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

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	repo := repository.NewRepository(db, redisClient)

	// Update component definitions and connectors based on latest version of
	// definitions.json.
	connDefs := connector.Init(logger, utils.GetConnectorOptions()).ListConnectorDefinitions()
	for _, connDef := range connDefs {
		if connDef.Tombstone {
			db.Unscoped().Model(&datamodel.Connector{}).Where("connector_definition_uid = ?", connDef.Uid).Update("tombstone", true)
		}

		cd := &pb.ComponentDefinition{
			Type: service.ConnectorTypeToComponentType[connDef.Type],
			Definition: &pb.ComponentDefinition_ConnectorDefinition{
				ConnectorDefinition: connDef,
			},
		}

		if err := updateComponentDefinition(ctx, cd, repo); err != nil {
			log.Fatal(err)
		}
	}

	opDefs := operator.Init(logger).ListOperatorDefinitions()
	for _, opDef := range opDefs {
		cd := &pb.ComponentDefinition{
			Type: pb.ComponentType_COMPONENT_TYPE_OPERATOR,
			Definition: &pb.ComponentDefinition_OperatorDefinition{
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
			aclClient = acl.NewACLClient(fgaClient, nil, redisClient, (*models.AuthorizationModels)[0].Id)
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
			err = aclClient.SetOwner(context.Background(), "pipeline", pipeline.UID, nsType, userUID)
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
			err = aclClient.SetOwner(context.Background(), "connector", conn.UID, nsType, userUID)
			if err != nil {
				panic(err)
			}
		}
		if pageToken == "" {
			break
		}
	}

}

type definition interface {
	GetUid() string
	GetId() string
	GetVersion() string
}

func updateComponentDefinition(ctx context.Context, cd *pb.ComponentDefinition, repo repository.Repository) error {
	var def definition
	switch cd.Type {
	case pb.ComponentType_COMPONENT_TYPE_OPERATOR:
		def = cd.GetOperatorDefinition()

	case pb.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
		pb.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
		pb.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION:

		def = cd.GetConnectorDefinition()
	default:
		return fmt.Errorf("unsupported component definition type")
	}

	uid, err := uuid.FromString(def.GetUid())
	if err != nil {
		return fmt.Errorf("invalid UID in component definition %s: %w", def.GetId(), err)
	}

	inDB, err := repo.GetComponentDefinitionByUID(ctx, uid)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("error fetching component definition %s from DB: %w", def.GetId(), err)
	}

	shouldSkip, err := shouldSkipUpsert(def, inDB)
	if err != nil {
		return err
	}
	if shouldSkip {
		return nil
	}

	if err := repo.UpsertComponentDefinition(ctx, cd); err != nil {
		return fmt.Errorf("failed to upsert component definition %s: %w", def.GetId(), err)
	}

	return nil
}

// A component definition is only upserted when either of these conditions is
// satisfied:
//   - There's a version bump in the definition.
//   - The feature score of the component (defined in the codebase as this isn't
//     a public property of the definition) has changed.
func shouldSkipUpsert(def definition, inDB *datamodel.ComponentDefinition) (bool, error) {
	if inDB == nil {
		return false, nil
	}

	if inDB.FeatureScore != datamodel.FeatureScores[def.GetId()] {
		return false, nil
	}

	v, err := semver.Parse(def.GetVersion())
	if err != nil {
		return false, fmt.Errorf("failed to parse version from component definition %s: %w", def.GetId(), err)
	}

	vInDB, err := semver.Parse(inDB.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse version from DB component definition %s: %w", def.GetId(), err)
	}

	isDBVersionOutdated := v.ComparePrecedence(vInDB) > 0
	if isDBVersionOutdated {
		return false, nil
	}
	return true, nil
}
