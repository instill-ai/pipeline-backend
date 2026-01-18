package service

import (
	"context"
	"errors"
	"fmt"

	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	component "github.com/instill-ai/pipeline-backend/pkg/component/store"
	errorsx "github.com/instill-ai/x/errors"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
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
func (s *service) ListComponentDefinitions(ctx context.Context, req *pipelinepb.ListComponentDefinitionsRequest) (*pipelinepb.ListComponentDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	page := s.pageInRange(req.GetPage())

	var compType pipelinepb.ComponentType
	var releaseStage pipelinepb.ComponentDefinition_ReleaseStage
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

	defs := make([]*pipelinepb.ComponentDefinition, len(uids))

	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, err
	}

	for i, uid := range uids {
		def, err := s.component.GetDefinitionByUID(uid.UID, vars, nil)
		if err != nil {
			return nil, err
		}
		if req.GetView() != pipelinepb.ComponentDefinition_VIEW_FULL {
			def.Spec = nil
		}

		defs[i] = def
	}

	resp := &pipelinepb.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(page),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: defs,
	}

	return resp, nil
}

func (s *service) getComponentDefinitionByID(ctx context.Context, id string) (*pipelinepb.ComponentDefinition, error) {
	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, err
	}

	cd, err := s.component.GetDefinitionByID(id, vars, nil)
	if err != nil {
		if errors.Is(err, component.ErrComponentDefinitionNotFound) {
			err = fmt.Errorf("fetching component definition: %w", errorsx.ErrNotFound)
		}
		return nil, err
	}

	return cd, nil
}
