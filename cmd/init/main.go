package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	_ "embed"

	"github.com/gofrs/uuid"
	"github.com/launchdarkly/go-semver"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/config"
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
	if err := config.Init(config.ParseConfigFlag()); err != nil {
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

	// This is a workaround solution for the Instill Model connector in Instill Cloud to improve response speed.
	_ = redisClient.Del(ctx, "instill_model_connector_def")

	repo := repository.NewRepository(db, redisClient)

	// Update component definitions and connectors based on latest version of
	// definitions.json.
	connDefs := connector.Init(logger, nil, utils.GetConnectorOptions()).ListConnectorDefinitions()
	for _, connDef := range connDefs {

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

	opDefs := operator.Init(logger, nil).ListOperatorDefinitions()
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
