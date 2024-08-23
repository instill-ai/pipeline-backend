package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/x/errmsg"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
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

	props, hasIntegration := cd.GetSpec().GetComponentSpecification().GetFields()["properties"]
	if !hasIntegration {
		return nil, errIntegrationNotFound
	}

	setup, hasIntegration := props.GetStructValue().GetFields()["setup"]
	if !hasIntegration {
		return nil, errIntegrationNotFound
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

	cdIdx, err := s.repository.GetDefinitionByUID(ctx, uuid.FromStringOrNil(cd.GetUid()))
	if err != nil {
		return nil, fmt.Errorf("fetching definition index: %w", err)
	}

	return &pb.Integration{
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
