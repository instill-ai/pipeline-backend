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

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/component/base"
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

	integration, err := s.componentDefinitionToIntegration(cd, view)
	if err != nil {
		return nil, errIntegrationNotFound
	}

	return integration, nil
}

func (s *service) ListIntegrations(ctx context.Context, req *pb.ListIntegrationsRequest) (*pb.ListIntegrationsResponse, error) {
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("qIntegration", filtering.TypeString),
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

		integrations[i], err = s.componentDefinitionToIntegration(cd, pb.View_VIEW_BASIC)
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

	return &pb.Integration{
		Uid:         cd.GetUid(),
		Id:          cd.GetId(),
		Title:       cd.GetTitle(),
		Description: cd.GetDescription(),
		Vendor:      cd.GetVendor(),
		Icon:        cd.GetIcon(),
		Schemas:     schemas,
		View:        view,
	}, nil
}

// validateConnection checks an input connection (for creation) is valid. Note
// that OUTUPUT_ONLY fields and undefined setup fields will be set to zero.
func (s *service) validateConnection(conn *pb.Connection, integration *pb.Integration) error {
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
	if conn.GetMethod() != pb.Connection_METHOD_DICTIONARY {
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

func (s *service) CreateNamespaceConnection(ctx context.Context, conn *pb.Connection) (*pb.Connection, error) {
	ns, err := s.GetRscNamespace(ctx, conn.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, fmt.Errorf("checking namespace permissions: %w", err)
	}

	integration, err := s.GetIntegration(ctx, conn.GetIntegrationId(), pb.View_VIEW_FULL)
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

	return s.connectionToPB(inserted, conn.GetNamespaceId(), pb.View_VIEW_FULL), nil
}

func (s *service) GetNamespaceConnection(ctx context.Context, req *pb.GetNamespaceConnectionRequest) (*pb.Connection, error) {
	view := req.GetView()
	if view == pb.View_VIEW_UNSPECIFIED {
		view = pb.View_VIEW_BASIC
	}

	ns, err := s.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, fmt.Errorf("checking namespace permissions: %w", err)
	}

	inDB, err := s.repository.GetNamespaceConnectionByID(ctx, ns.NsUID, req.GetConnectionId())
	if err != nil {
		return nil, fmt.Errorf("fetching connection: %w", err)
	}

	conn := s.connectionToPB(inDB, req.GetNamespaceId(), view)
	if view != pb.View_VIEW_FULL {
		return conn, nil
	}

	conn.Setup = new(structpb.Struct)
	if err := conn.Setup.UnmarshalJSON(inDB.Setup); err != nil {
		return nil, fmt.Errorf("unmarshalling setup: %w", err)
	}

	// TODO jvallesm: INS-5963 addresses redacting these values.

	return conn, nil
}

func (s *service) connectionToPB(conn *datamodel.Connection, nsID string, view pb.View) *pb.Connection {
	return &pb.Connection{
		Uid:              conn.UID.String(),
		Id:               conn.ID,
		NamespaceId:      nsID,
		IntegrationId:    conn.Integration.ID,
		IntegrationTitle: conn.Integration.Title,
		Method:           pb.Connection_Method(conn.Method),
		View:             view,
		CreateTime:       timestamppb.New(conn.CreateTime),
		UpdateTime:       timestamppb.New(conn.UpdateTime),
	}
}

func (s *service) ListNamespaceConnections(ctx context.Context, req *pb.ListNamespaceConnectionsRequest) (*pb.ListNamespaceConnectionsResponse, error) {
	ns, err := s.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, fmt.Errorf("checking namespace permissions: %w", err)
	}

	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("integrationId", filtering.TypeString),
		filtering.DeclareIdent("qConnection", filtering.TypeString),
	)
	if err != nil {
		return nil, fmt.Errorf("building filter declarations: %w", err)
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}
	p := repository.ListNamespaceConnectionsParams{
		NamespaceUID: ns.NsUID,
		PageToken:    req.GetPageToken(),
		Limit:        s.pageSizeInRange(req.GetPageSize()),
		Filter:       filter,
	}

	dbConns, err := s.repository.ListNamespaceConnections(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("fetching connections: %w", err)
	}

	resp := &pb.ListNamespaceConnectionsResponse{
		Connections:   make([]*pb.Connection, len(dbConns.Connections)),
		NextPageToken: dbConns.NextPageToken,
		TotalSize:     dbConns.TotalSize,
	}

	for i, inDB := range dbConns.Connections {
		resp.Connections[i] = s.connectionToPB(inDB, req.GetNamespaceId(), pb.View_VIEW_BASIC)
	}

	return resp, nil
}

func (s *service) DeleteNamespaceConnection(ctx context.Context, namespaceID, id string) error {
	ns, err := s.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		return fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return fmt.Errorf("checking namespace permissions: %w", err)
	}

	return s.repository.DeleteNamespaceConnectionByID(ctx, ns.NsUID, id)

}
