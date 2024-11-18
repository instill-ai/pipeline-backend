package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/datatypes"

	fieldmaskutil "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
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
		if errors.Is(err, errIntegrationConversion) {
			return nil, errIntegrationNotFound
		}

		return nil, fmt.Errorf("converting component definition: %w", err)
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

	// TODO add HelpLink

	integration := &pb.Integration{
		Uid:         cd.GetUid(),
		Id:          cd.GetId(),
		Title:       cd.GetTitle(),
		Description: cd.GetDescription(),
		Vendor:      cd.GetVendor(),
		Icon:        cd.GetIcon(),
		View:        view,
	}

	if view != pb.View_VIEW_FULL {
		return integration, nil
	}

	integration.SetupSchema = setup.GetStructValue()
	schemaFields := integration.SetupSchema.GetFields()

	//nolint:staticcheck
	// This is deprecated and only maintained for backwards compatibility.
	// TODO jvallesm: remove when Integration Milestone 2 (OAuth) is rolled
	// out.
	integration.Schemas = []*pb.Integration_SetupSchema{
		{
			Method: pb.Connection_METHOD_DICTIONARY,
			Schema: setup.GetStructValue(),
		},
	}

	supportsOAuth, err := s.component.SupportsOAuth(uuid.FromStringOrNil(integration.Uid))
	if err != nil {
		return nil, fmt.Errorf("checking OAuth support: %w", err)
	}

	oAuthConfig, hasOAuthConfig := schemaFields["instillOAuthConfig"]
	if !(supportsOAuth && hasOAuthConfig) {
		return integration, nil
	}

	// We extract the OAuth configuration to a dedicated proto field to
	// have a structured contract with the clients.
	j, err := oAuthConfig.GetStructValue().MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling OAuth config: %w", err)
	}

	integration.OAuthConfig = new(pb.Integration_OAuthConfig)
	if err := protojson.Unmarshal(j, integration.OAuthConfig); err != nil {
		return nil, fmt.Errorf("unmarshalling OAuth config: %w", err)
	}

	// We remove the information from the setup so it isn't duplicated
	// in the response.
	delete(schemaFields, "instillOAuthConfig")

	return integration, nil

}

var outputOnlyConnectionFields = []string{
	"uid",
	"namespace_id",
	"integration_title",
	"create_time",
	"update_time",
}

// validateConnection validates the fields of a pb.Connection. In particular,it
// verifies the setup fulfills its integration's schema.
// Note that the connection input will be modified.
func (s *service) validateConnection(conn *pb.Connection, integration *pb.Integration) error {
	// https://github.com/instill-ai/protobufs/pull/475 removed
	// protoc-gen-validate because it broke the generated code used in the
	// Python SDK.
	// TODO reintroduce protoc-gen-validate and leverage struct validators.
	// if err := conn.Validate(); err != nil {
	// 	return err
	// }

	switch conn.GetMethod() {
	case pb.Connection_METHOD_DICTIONARY:
		err := fmt.Errorf("%w: invalid payload in dictionary connection", errdomain.ErrInvalidArgument)
		if integration.GetOAuthConfig() != nil {
			return errmsg.AddMessage(err, integration.GetTitle()+" connection only accepts METHOD_OAUTH.")
		}
		if len(conn.GetScopes()) > 0 {
			return errmsg.AddMessage(err, "Scopes only apply to OAuth connections.")
		}
		if conn.GetOAuthAccessDetails() != nil {
			return errmsg.AddMessage(err, "OAuth access details only apply to OAuth connections.")
		}
		if conn.Identity != nil {
			return errmsg.AddMessage(err, "Identity only applies to OAuth connections.")
		}
	case pb.Connection_METHOD_OAUTH:
		err := fmt.Errorf("%w: invalid payload in OAuth connection", errdomain.ErrInvalidArgument)
		if integration.GetOAuthConfig() == nil {
			return errmsg.AddMessage(err, integration.GetTitle()+" connection doesn't accept METHOD_OAUTH.")
		}
		if conn.GetIdentity() == "" {
			return errmsg.AddMessage(err, "Identity must be provided in OAuth connections.")
		}
	default:
		err := fmt.Errorf("%w: unsupported method", errdomain.ErrInvalidArgument)
		return errmsg.AddMessage(err, "Invalid method "+conn.GetMethod().String()+".")
	}

	// Validate setup fulfills integration schema.
	schema, err := integration.GetSetupSchema().MarshalJSON()
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

	conn.Setup, err = structpb.NewStruct(setup)
	if err != nil {
		return fmt.Errorf("filtering setup: %w", err)
	}

	return nil
}

// validateConnectionCreation checks an input connection is valid for creation.
// Note that OUTUPUT_ONLY fields and undefined setup fields will be set to
// zero.
func (s *service) validateConnectionCreation(conn *pb.Connection, integration *pb.Integration) error {
	// Check REQUIRED fields are provided in the request.
	requiredFields := []string{
		"id",
		"integration_id",
		"method",
		"setup",
	}
	if err := checkfield.CheckRequiredFields(conn, requiredFields); err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	// Set all OUTPUT_ONLY fields to zero value.
	if err := checkfield.CheckCreateOutputOnlyFields(conn, outputOnlyConnectionFields); err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	// Validate resource ID.
	if err := checkfield.CheckResourceID(conn.GetId()); err != nil {
		return fmt.Errorf("%w: %w", errdomain.ErrInvalidArgument, err)
	}

	return s.validateConnection(conn, integration)
}

var errEmptyMask = fmt.Errorf("empty mask")

// validateConnectionUpdate checks an input connection is valid for update.
// Note that OUTUPUT_ONLY fields and undefined setup fields will be set to
// zero in the input connection.
func (s *service) validateConnectionUpdate(
	updateReq, destConn *pb.Connection,
	pbMask *fieldmaskpb.FieldMask,
	integration *pb.Integration,
) (err error) {
	// google.protobuf.Struct needs to be updated in block.
	for i, path := range pbMask.Paths {
		switch {
		case strings.Contains(path, "setup"):
			pbMask.Paths[i] = "setup"
		case strings.Contains(path, "o_auth_access_details"):
			pbMask.Paths[i] = "o_auth_access_details"
		}
	}

	if !pbMask.IsValid(updateReq) {
		return fmt.Errorf("%w: invalid input mask", errdomain.ErrInvalidArgument)
	}

	pbMask, err = checkfield.CheckUpdateOutputOnlyFields(pbMask, outputOnlyConnectionFields)
	if err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	mask, err := fieldmaskutil.MaskFromProtoFieldMask(pbMask, strcase.ToCamel)
	if err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	if mask.IsEmpty() {
		return errEmptyMask
	}

	// Return error if IMMUTABLE fields are intentionally changed.
	immutableFields := []string{"integration_id"}
	if err := checkfield.CheckUpdateImmutableFields(updateReq, destConn, immutableFields); err != nil {
		return fmt.Errorf("%w:%w", errdomain.ErrInvalidArgument, err)
	}

	// Only the fields mentioned in the field mask will be copied to the
	// destination connection, the other fields will be left intact.
	if err := fieldmaskutil.StructToStruct(mask, updateReq, destConn); err != nil {
		return fmt.Errorf("copying updates to connection object: %w", err)
	}

	return s.validateConnection(destConn, integration)
}

func (s *service) CreateNamespaceConnection(ctx context.Context, req *pb.CreateNamespaceConnectionRequest) (*pb.Connection, error) {
	ns, err := s.GetNamespaceByID(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, fmt.Errorf("checking namespace permissions: %w", err)
	}

	conn := req.GetConnection()
	integration, err := s.GetIntegration(ctx, conn.GetIntegrationId(), pb.View_VIEW_FULL)
	if err != nil {
		if errors.Is(err, errIntegrationNotFound) {
			return nil, fmt.Errorf("%w: invalid integration ID", errdomain.ErrInvalidArgument)
		}

		return nil, fmt.Errorf("fetching integration details: %w", err)
	}

	if err := s.validateConnectionCreation(conn, integration); err != nil {
		return nil, err
	}

	jsonSetup, err := conn.GetSetup().MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling setup: %w", err)
	}

	var jsonOAuth datatypes.JSON
	if conn.GetOAuthAccessDetails() != nil {
		jsonOAuth, err = conn.GetOAuthAccessDetails().MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshalling OAuth details: %w", err)
		}
	}

	identity := sql.NullString{}
	if conn.Identity != nil {
		identity.Valid = true
		identity.String = *conn.Identity
	}

	inserted, err := s.repository.CreateNamespaceConnection(ctx, &datamodel.Connection{
		ID:                 conn.GetId(),
		NamespaceUID:       ns.NsUID,
		IntegrationUID:     uuid.FromStringOrNil(integration.GetUid()),
		Method:             datamodel.ConnectionMethod(conn.GetMethod()),
		Setup:              datatypes.JSON(jsonSetup),
		Identity:           identity,
		Scopes:             conn.GetScopes(),
		OAuthAccessDetails: jsonOAuth,
	})
	if err != nil {
		return nil, fmt.Errorf("persisting connection: %w", err)
	}

	return s.connectionToPB(inserted, conn.GetNamespaceId(), pb.View_VIEW_FULL)
}

func (s *service) UpdateNamespaceConnection(ctx context.Context, req *pb.UpdateNamespaceConnectionRequest) (*pb.Connection, error) {
	ns, err := s.GetNamespaceByID(ctx, req.GetNamespaceId())
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

	destConn, err := s.connectionToPB(inDB, ns.NsID, pb.View_VIEW_FULL)
	if err != nil {
		return nil, fmt.Errorf("converting database connection to proto: %w", err)
	}

	integration, err := s.GetIntegration(ctx, inDB.Integration.ID, pb.View_VIEW_FULL)
	if err != nil {
		if errors.Is(err, errIntegrationNotFound) {
			return nil, fmt.Errorf("%w: invalid integration ID", errdomain.ErrInvalidArgument)
		}

		return nil, fmt.Errorf("fetching integration details: %w", err)
	}

	err = s.validateConnectionUpdate(req.GetConnection(), destConn, req.GetUpdateMask(), integration)
	if err != nil {
		if !errors.Is(err, errEmptyMask) {
			return nil, err
		}

		return s.connectionToPB(inDB, ns.NsID, pb.View_VIEW_FULL)
	}

	jsonSetup, err := destConn.GetSetup().MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshalling setup: %w", err)
	}

	var jsonOAuth datatypes.JSON
	if destConn.GetOAuthAccessDetails() != nil {
		jsonOAuth, err = destConn.GetOAuthAccessDetails().MarshalJSON()
		if err != nil {
			return nil, fmt.Errorf("marshalling OAuth details: %w", err)
		}
	}

	identity := sql.NullString{}
	if destConn.Identity != nil {
		identity.Valid = true
		identity.String = *destConn.Identity
	}

	updated, err := s.repository.UpdateNamespaceConnectionByUID(ctx, inDB.UID, &datamodel.Connection{
		ID:                 destConn.GetId(),
		Method:             datamodel.ConnectionMethod(destConn.GetMethod()),
		Setup:              datatypes.JSON(jsonSetup),
		Scopes:             destConn.GetScopes(),
		Identity:           identity,
		OAuthAccessDetails: jsonOAuth,
	})
	if err != nil {
		return nil, fmt.Errorf("persisting connection: %w", err)
	}

	return s.connectionToPB(updated, destConn.GetNamespaceId(), pb.View_VIEW_FULL)
}

func (s *service) GetNamespaceConnection(ctx context.Context, req *pb.GetNamespaceConnectionRequest) (*pb.Connection, error) {
	view := req.GetView()
	if view == pb.View_VIEW_UNSPECIFIED {
		view = pb.View_VIEW_BASIC
	}

	ns, err := s.GetNamespaceByID(ctx, req.GetNamespaceId())
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

	return s.connectionToPB(inDB, req.GetNamespaceId(), view)
}

func (s *service) connectionToPB(conn *datamodel.Connection, nsID string, view pb.View) (*pb.Connection, error) {
	pbConn := &pb.Connection{
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

	if conn.Identity.Valid {
		pbConn.Identity = &conn.Identity.String
	}

	if view != pb.View_VIEW_FULL {
		return pbConn, nil
	}

	// TODO jvallesm: INS-5963 addresses redacting these values.
	pbConn.Setup = new(structpb.Struct)
	if err := pbConn.Setup.UnmarshalJSON(conn.Setup); err != nil {
		return nil, fmt.Errorf("unmarshalling setup: %w", err)
	}

	pbConn.Scopes = conn.Scopes

	if len(conn.OAuthAccessDetails) > 0 {
		pbConn.OAuthAccessDetails = new(structpb.Struct)
		if err := pbConn.OAuthAccessDetails.UnmarshalJSON(conn.OAuthAccessDetails); err != nil {
			return nil, fmt.Errorf("unmarshalling OAuth config: %w", err)
		}
	}

	return pbConn, nil
}

func (s *service) ListNamespaceConnections(ctx context.Context, req *pb.ListNamespaceConnectionsRequest) (*pb.ListNamespaceConnectionsResponse, error) {
	ns, err := s.GetNamespaceByID(ctx, req.GetNamespaceId())
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
		resp.Connections[i], err = s.connectionToPB(inDB, req.GetNamespaceId(), pb.View_VIEW_BASIC)
		if err != nil {
			return nil, fmt.Errorf("building proto connection: %w", err)
		}
	}

	return resp, nil
}

func (s *service) DeleteNamespaceConnection(ctx context.Context, namespaceID, id string) error {
	ns, err := s.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return fmt.Errorf("checking namespace permissions: %w", err)
	}

	return s.repository.DeleteNamespaceConnectionByID(ctx, ns.NsUID, id)

}

func (s *service) ListPipelineIDsByConnectionID(ctx context.Context, req *pb.ListPipelineIDsByConnectionIDRequest) (*pb.ListPipelineIDsByConnectionIDResponse, error) {
	ns, err := s.GetNamespaceByID(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("fetching namespace: %w", err)
	}

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, fmt.Errorf("checking namespace permissions: %w", err)
	}

	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("q", filtering.TypeString),
	)
	if err != nil {
		return nil, fmt.Errorf("building filter declarations: %w", err)
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, fmt.Errorf("parsing filter: %w", err)
	}

	p := repository.ListPipelineIDsByConnectionIDParams{
		Owner:        ns,
		ConnectionID: req.GetConnectionId(),
		PageToken:    req.GetPageToken(),
		Limit:        s.pageSizeInRange(req.GetPageSize()),
		Filter:       filter,
	}

	page, err := s.repository.ListPipelineIDsByConnectionID(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("fetching connections: %w", err)
	}

	return &pb.ListPipelineIDsByConnectionIDResponse{
		PipelineIds:   page.PipelineIDs,
		NextPageToken: page.NextPageToken,
		TotalSize:     page.TotalSize,
	}, nil
}
