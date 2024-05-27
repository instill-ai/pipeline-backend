package definitionupdater

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/launchdarkly/go-semver"

	"github.com/instill-ai/component"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type definition interface {
	GetUid() string
	GetId() string
	GetVersion() string
	GetTombstone() bool
	GetPublic() bool
}

// UpdateComponentDefinitionIndex updates the component definitions in the
// database based on latest version of their definition.json file.
func UpdateComponentDefinitionIndex(ctx context.Context, repo repository.Repository) error {
	logger, _ := logger.GetZapLogger(ctx)

	connDefs := component.Init(logger, nil, nil).ListConnectorDefinitions(nil, true)
	for _, connDef := range connDefs {
		cd := &pb.ComponentDefinition{
			Type: service.ConnectorTypeToComponentType[connDef.Type],
			Definition: &pb.ComponentDefinition_ConnectorDefinition{
				ConnectorDefinition: connDef,
			},
		}

		if err := updateComponentDefinition(ctx, cd, repo); err != nil {
			return err
		}
	}

	opDefs := component.Init(logger, nil, nil).ListOperatorDefinitions(nil, true)
	for _, opDef := range opDefs {
		cd := &pb.ComponentDefinition{
			Type: pb.ComponentType_COMPONENT_TYPE_OPERATOR,
			Definition: &pb.ComponentDefinition_OperatorDefinition{
				OperatorDefinition: opDef,
			},
		}

		if err := updateComponentDefinition(ctx, cd, repo); err != nil {
			return err
		}
	}

	return nil
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
//   - The tombstone tag has changed.
//   - The feature score of the component (defined in the codebase as this isn't
//     a public property of the definition) has changed.
func shouldSkipUpsert(def definition, inDB *datamodel.ComponentDefinition) (bool, error) {
	if inDB == nil {
		return false, nil
	}

	if inDB.IsVisible != (def.GetPublic() && !def.GetTombstone()) {
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
