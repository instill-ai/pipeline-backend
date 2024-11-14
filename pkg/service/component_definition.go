package service

import (
	"context"
	"errors"
	"fmt"

	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	component "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (s *service) pageSizeInRange(pageSize int32) int {
	if pageSize <= 0 {
		return repository.DefaultPageSize
	}

	if pageSize > repository.MaxPageSize {
		return repository.MaxPageSize
	}

	return int(pageSize)
}

func (s *service) pageInRange(page int32) int {
	if page <= 0 {
		return 0
	}

	return int(page)
}

// ListComponentDefinitions returns a paginated list of components.
func (s *service) ListComponentDefinitions(ctx context.Context, req *pb.ListComponentDefinitionsRequest) (*pb.ListComponentDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	page := s.pageInRange(req.GetPage())

	var compType pb.ComponentType
	var releaseStage pb.ComponentDefinition_ReleaseStage
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("qTitle", filtering.TypeString),
		filtering.DeclareEnumIdent("releaseStage", releaseStage.Type()),
		filtering.DeclareEnumIdent("componentType", compType.Type()),
	)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	p := repository.ListComponentDefinitionsParams{
		Offset: page * pageSize,
		Limit:  pageSize,
		Filter: filter,
	}

	uids, totalSize, err := s.repository.ListComponentDefinitionUIDs(ctx, p)
	if err != nil {
		return nil, err
	}

	defs := make([]*pb.ComponentDefinition, len(uids))

	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, err
	}

	for i, uid := range uids {
		def, err := s.component.GetDefinitionByUID(uid.UID, vars, nil)
		if err != nil {
			return nil, err
		}
		if req.GetView() != pb.ComponentDefinition_VIEW_FULL {
			def.Spec = nil
		}

		// Temporary solution to filter out the instill-app component for local-ce:dev edition
		// In the future the non-CE definitions shouldn't be in the CE repositories and Cloud should extend the CE component store to initialize the private components.
		if skipComponentInCE(def.GetId()) {
			continue
		}

		defs[i] = def
	}

	resp := &pb.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(page),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: defs,
	}

	return resp, nil
}

var skipComponentsInCE = map[string]bool{
	"instill-app": true,
}

func skipComponentInCE(componentID string) bool {
	if skipComponentsInCE[componentID] && config.Config.Server.Edition == "local-ce:dev" {
		return true
	}
	return false
}

func (s *service) getComponentDefinitionByID(ctx context.Context, id string) (*pb.ComponentDefinition, error) {
	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, err
	}

	cd, err := s.component.GetDefinitionByID(id, vars, nil)
	if err != nil {
		if errors.Is(err, component.ErrComponentDefinitionNotFound) {
			err = fmt.Errorf("fetching component definition: %w", errdomain.ErrNotFound)
		}
		return nil, err
	}

	return cd, nil
}
