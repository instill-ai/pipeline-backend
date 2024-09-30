package definitionupdater

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/launchdarkly/go-semver"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// UpdateComponentDefinitionIndex updates the component definitions in the
// database based on latest version of their definition.json file.
func UpdateComponentDefinitionIndex(ctx context.Context, repo repository.Repository) error {
	logger, _ := logger.GetZapLogger(ctx)

	defs := componentstore.Init(logger, nil, nil).ListDefinitions(nil, true)
	for _, def := range defs {

		if err := updateComponentDefinition(ctx, def, repo); err != nil {
			return err
		}
	}

	return nil
}

func updateComponentDefinition(ctx context.Context, cd *pb.ComponentDefinition, repo repository.Repository) error {
	uid, err := uuid.FromString(cd.GetUid())
	if err != nil {
		return fmt.Errorf("invalid UID in component definition %s: %w", cd.GetId(), err)
	}

	inDB, err := repo.GetDefinitionByUID(ctx, uid)
	if err != nil && !errors.Is(err, errdomain.ErrNotFound) {
		return fmt.Errorf("error fetching component definition %s from DB: %w", cd.GetId(), err)
	}

	inDef := datamodel.ComponentDefinitionFromProto(cd)
	if shouldSkip, err := shouldSkipUpsert(inDef, inDB); err != nil {
		return err
	} else if shouldSkip {
		return nil
	}

	if err := repo.UpsertComponentDefinition(ctx, cd); err != nil {
		return fmt.Errorf("failed to upsert component definition %s: %w", cd.GetId(), err)
	}

	return nil
}

// A component definition is only upserted when any of these conditions is
// satisfied:
//   - There's a version bump in the definition.
//   - The visibility has changed.
//   - An integration configuration is introduced or removed.
//   - The component version is bumped.
//   - The feature score of the component (defined in the codebase as this isn't
//     a public property of the definition) has changed.
//   - The title or vendor name have changed (fuzzy search is performed against
//     these).
func shouldSkipUpsert(def, inDB *datamodel.ComponentDefinition) (bool, error) {
	if inDB == nil {
		return false, nil
	}

	if inDB.Title != def.Title {
		return false, nil
	}

	if inDB.Vendor != def.Vendor {
		return false, nil
	}

	if inDB.IsVisible != def.IsVisible {
		return false, nil
	}

	if inDB.HasIntegration != def.HasIntegration {
		return false, nil
	}

	if inDB.FeatureScore != def.FeatureScore {
		return false, nil
	}

	v, err := semver.Parse(def.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse version from component definition %s: %w", def.ID, err)
	}

	vInDB, err := semver.Parse(inDB.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse version from DB component definition %s: %w", def.ID, err)
	}

	isDBVersionOutdated := v.ComparePrecedence(vInDB) > 0
	if isDBVersionOutdated {
		return false, nil
	}

	return true, nil
}
