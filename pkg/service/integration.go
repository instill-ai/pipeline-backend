package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/x/errmsg"
	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var errIntegrationNotFound = errmsg.AddMessage(errdomain.ErrNotFound, "Integration does not exist.")

func (s *service) GetIntegration(ctx context.Context, id string, view pb.View) (*pb.Integration, error) {
	cd, err := s.getComponentDefinitionByID(ctx, id)
	if err != nil {
		if errors.Is(err, errdomain.ErrNotFound) {
			err = errIntegrationNotFound
		}

		return nil, fmt.Errorf("fetching component information: %w", err)
	}

	cdIdx, err := s.repository.GetDefinitionByUID(ctx, uuid.FromStringOrNil(cd.GetUid()))
	if err != nil {
		return nil, fmt.Errorf("fetching definition index: %w", err)
	}

	integration, err := s.componentDefinitionToIntegration(cd, cdIdx, view)
	if err != nil {
		return nil, errIntegrationNotFound
	}

	return integration, nil
}

func (s *service) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("qIntegration", filtering.TypeString),
		filtering.DeclareIdent("featured", filtering.TypeBool),
	)
	if err != nil {
		return nil, fmt.Errorf("building filter declarations: %w", err)
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}

	p := repository.ListIntegrationsParams{
		PageToken: req.GetPageToken(),
		Limit:     s.pageSizeInRange(req.GetPageSize()),
		Filter:    filter,
	}

	integrationsPage, err := s.repository.ListIntegrations(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("fetching integration UIDs: %w", err)
	}

	vars, err := recipe.GenerateSystemVariables(ctx, recipe.SystemVariables{})
	if err != nil {
		return nil, fmt.Errorf("generating system variables: %w", err)
	}

	cdIndices := integrationsPage.ComponentDefinitions
	integrations := make([]*pb.Integration, len(cdIndices))
	for i, cdIdx := range cdIndices {
		cd, err := s.component.GetDefinitionByUID(cdIdx.UID, vars, nil)
		if err != nil {
			return nil, fmt.Errorf("fetching component definition: %w", err)
		}

		integrations[i], err = s.componentDefinitionToIntegration(cd, cdIdx, pb.View_VIEW_BASIC)
		if err != nil {
			return nil, fmt.Errorf("converting component definition: %w", err)
		}
	}

	return &pb.ListIntegrationsResponse{
		Integrations:  integrations,
		NextPageToken: integrationsPage.NextPageToken,
		TotalSize:     integrationsPage.TotalSize,
	}, nil
}

var errIntegrationConversion = fmt.Errorf("component definition has no integration configuration")

func (s *service) componentDefinitionToIntegration(
	cd *pb.ComponentDefinition,
	cdIdx *datamodel.ComponentDefinition,
	view pb.View,
) (*pb.Integration, error) {

	props, hasIntegration := cd.GetSpec().GetComponentSpecification().GetFields()["properties"]
	if !hasIntegration {
		return nil, errIntegrationConversion
	}

	setup, hasIntegration := props.GetStructValue().GetFields()["setup"]
	if !hasIntegration {
		return nil, errIntegrationConversion
	}

	var schemas []*pb.Integration_SetupSchema
	if view == pb.View_VIEW_FULL {
		// Integration Milestone 1 supports only key-value integrations.
		schemas = []*pb.Integration_SetupSchema{
			{
				Method: pb.Connection_METHOD_DICTIONARY,
				Schema: setup.GetStructValue(),
			},
		}
	}

	return &pipelinepb.Integration{
		Uid:         cd.GetUid(),
		Id:          cd.GetId(),
		Title:       cd.GetTitle(),
		Description: cd.GetDescription(),
		Vendor:      cd.GetVendor(),
		Icon:        cd.GetIcon(),
		// TODO jvallesm: we'll probably want different "featured" lists for
		// the component defintion list (showcase components in the marketing
		// website) and for the integrations (shortlist on the integrations
		// page or pipeline builder).
		Featured: cdIdx.FeatureScore > 0,
		Schemas:  schemas,
		View:     view,
	}, nil
}
