package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"

	componentbase "github.com/instill-ai/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/errmsg"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var errIntegrationNotFound = errmsg.AddMessage(errdomain.ErrNotFound, "Integration does not exist.")

func (s *service) GetIntegration(ctx context.Context, id string, view pipelinepb.View) (*pipelinepb.Integration, error) {
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

func (s *service) ListIntegrations(ctx context.Context, req *pipelinepb.ListIntegrationsRequest) (*pipelinepb.ListIntegrationsResponse, error) {
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
	integrations := make([]*pipelinepb.Integration, len(cdIndices))
	for i, cdIdx := range cdIndices {
		cd, err := s.component.GetDefinitionByUID(cdIdx.UID, vars, nil)
		if err != nil {
			return nil, fmt.Errorf("fetching component definition: %w", err)
		}

		integrations[i], err = s.componentDefinitionToIntegration(cd, cdIdx, pipelinepb.View_VIEW_BASIC)
		if err != nil {
			return nil, fmt.Errorf("converting component definition: %w", err)
		}
	}

	return &pipelinepb.ListIntegrationsResponse{
		Integrations:  integrations,
		NextPageToken: integrationsPage.NextPageToken,
		TotalSize:     integrationsPage.TotalSize,
	}, nil
}

var errIntegrationConversion = fmt.Errorf("component definition has no integration configuration")

func (s *service) componentDefinitionToIntegration(
	cd *pipelinepb.ComponentDefinition,
	cdIdx *datamodel.ComponentDefinition,
	view pipelinepb.View,
) (*pipelinepb.Integration, error) {

	props, hasIntegration := cd.GetSpec().GetComponentSpecification().GetFields()["properties"]
	if !hasIntegration {
		return nil, errIntegrationConversion
	}

	setup, hasIntegration := props.GetStructValue().GetFields()["setup"]
	if !hasIntegration {
		return nil, errIntegrationConversion
	}

	var schemas []*pipelinepb.Integration_SetupSchema
	if view == pipelinepb.View_VIEW_FULL {
		// Integration Milestone 1 supports only key-value integrations.
		schemas = []*pipelinepb.Integration_SetupSchema{
			{
				Method: pipelinepb.Connection_METHOD_DICTIONARY,
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

// validateConnection checks an input connection (for creation) is valid. Note
// that OUTUPUT_ONLY fields and undefined setup fields will be set to zero.
func (s *service) validateConnection(conn *pipelinepb.Connection, integration *pipelinepb.Integration) error {
	// Check REQUIRED fields are provided in the request.
	requiredFields := []string{
		"id",
		"namespace_id",
		"integration_id",
		"method",
		"setup",
	}
	if err := checkfield.CheckRequiredFields(conn, requiredFields); err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	// Set all OUTPUT_ONLY fields to zero value.
	outputOnlyFields := []string{
		"uid",
		"integration_title",
		"create_time",
		"update_time",
	}
	if err := checkfield.CheckCreateOutputOnlyFields(conn, outputOnlyFields); err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	// Validate resource ID.
	if err := checkfield.CheckResourceID(conn.GetId()); err != nil {
		return fmt.Errorf("%w: %w", errdomain.ErrInvalidArgument, err)
	}

	// Validate method is METHOD_DICTIONARY.
	// TODO jvallesm: support OAuth in v0.39.0
	if conn.GetMethod() != pipelinepb.Connection_METHOD_DICTIONARY {
		err := fmt.Errorf("%w: unsupported method", errdomain.ErrInvalidArgument)
		return errmsg.AddMessage(err, "Only METHOD_DICTIONARY is supported at the moment.")
	}

	var pbSchema *structpb.Struct
	for _, s := range integration.GetSchemas() {
		if s.GetMethod() == conn.GetMethod() {
			pbSchema = s.GetSchema()
		}
	}

	if pbSchema == nil {
		return fmt.Errorf(
			"%w: integration doesn't support method %s",
			errdomain.ErrInvalidArgument,
			conn.GetMethod().String(),
		)
	}

	// Validate setup fulfills integration schema.
	schema, err := pbSchema.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshalling integration schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.RegisterExtension(
		"instillAcceptFormats",
		componentbase.InstillAcceptFormatsMeta,
		componentbase.InstillAcceptFormatsCompiler{},
	)

	schemaID := fmt.Sprintf("%s/config/setup.json", conn.GetIntegrationId())
	if err := compiler.AddResource(schemaID, bytes.NewReader(schema)); err != nil {
		return fmt.Errorf("adding schema to compiler: %w", err)
	}

	validator, err := compiler.Compile(schemaID)
	if err != nil {
		return fmt.Errorf("compiling integration schema: %w", err)
	}

	setup := conn.GetSetup().AsMap()
	if err := validator.Validate(setup); err != nil {
		return fmt.Errorf("%w: %w", errdomain.ErrInvalidArgument, err)
	}

	// Remove setup fields that aren't defined in the schema.
	filteredSetup := map[string]any{}
	properties := pbSchema.GetFields()["properties"].GetStructValue()
	for k := range properties.GetFields() {
		filteredSetup[k] = setup[k]
	}

	conn.Setup, err = structpb.NewStruct(filteredSetup)
	if err != nil {
		return fmt.Errorf("filtering setup: %w", err)
	}

	return nil
}

func (s *service) CreateNamespaceConnection(ctx context.Context, conn *pipelinepb.Connection) (*pipelinepb.Connection, error) {
	ns, err := s.GetRscNamespace(ctx, conn.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, fmt.Errorf("checking namespace permissions: %w", err)
	}

	integration, err := s.GetIntegration(ctx, conn.GetIntegrationId(), pipelinepb.View_VIEW_FULL)
	if err != nil {
		if errors.Is(err, errIntegrationNotFound) {
			return nil, fmt.Errorf("%w: invalid integration ID", errdomain.ErrInvalidArgument)
		}

		return nil, fmt.Errorf("fetching integration details: %w", err)
	}

	if err := s.validateConnection(conn, integration); err != nil {
		return nil, err
	}

	j, err := conn.GetSetup().MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling setup: %w", err)
	}

	inserted, err := s.repository.CreateNamespaceConnection(ctx, &datamodel.Connection{
		ID:             conn.GetId(),
		NamespaceUID:   ns.NsUID,
		IntegrationUID: uuid.FromStringOrNil(integration.GetUid()),
		Method:         datamodel.ConnectionMethod(conn.GetMethod()),
		Setup:          datatypes.JSON(j),
	})
	if err != nil {
		return nil, fmt.Errorf("persisting connection: %w", err)
	}

	conn.Uid = inserted.UID.String()
	conn.CreateTime = timestamppb.New(inserted.CreateTime)
	conn.UpdateTime = timestamppb.New(inserted.UpdateTime)
	conn.View = pipelinepb.View_VIEW_FULL

	return conn, nil
}
