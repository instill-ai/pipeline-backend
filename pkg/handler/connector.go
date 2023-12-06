package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PrivateHandler) ListConnectorsAdmin(ctx context.Context, req *pipelinePB.ListConnectorsAdminRequest) (resp *pipelinePB.ListConnectorsAdminResponse, err error) {

	var pageSize int32
	var pageToken string

	resp = &pipelinePB.ListConnectorsAdminResponse{}
	pageSize = req.GetPageSize()
	pageToken = req.GetPageToken()

	var connType pipelinePB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		return nil, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	connectors, totalSize, nextPageToken, err := h.service.ListConnectorsAdmin(ctx, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		return nil, err
	}

	resp.Connectors = connectors
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(totalSize)

	return resp, nil

}

func (h *PrivateHandler) LookUpConnectorAdmin(ctx context.Context, req *pipelinePB.LookUpConnectorAdminRequest) (resp *pipelinePB.LookUpConnectorAdminResponse, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.LookUpConnectorAdminResponse{}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, lookUpConnectorRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] lookup connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "REQUIRED fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return nil, err
	}

	connector, err := h.service.GetConnectorByUIDAdmin(ctx, connUID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		return nil, err
	}

	resp.Connector = connector

	return resp, nil
}

func (h *PrivateHandler) CheckConnector(ctx context.Context, req *pipelinePB.CheckConnectorRequest) (resp *pipelinePB.CheckConnectorResponse, err error) {

	resp = &pipelinePB.CheckConnectorResponse{}
	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}

	connector, err := h.service.GetConnectorByUIDAdmin(ctx, connUID, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	if connector.Tombstone {
		resp.State = pipelinePB.Connector_STATE_ERROR
		return resp, nil
	}

	if connector.State == pipelinePB.Connector_STATE_CONNECTED {
		state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connector.Uid))
		if err != nil {
			return resp, err
		}

		resp.State = *state
		return resp, nil

	} else {
		resp.State = pipelinePB.Connector_STATE_DISCONNECTED
		return resp, nil
	}

}

func (h *PublicHandler) ListConnectors(ctx context.Context, req *pipelinePB.ListConnectorsRequest) (resp *pipelinePB.ListConnectorsResponse, err error) {

	eventName := "ListConnectors"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var pageSize int32
	var pageToken string

	resp = &pipelinePB.ListConnectorsResponse{}
	pageSize = req.GetPageSize()
	pageToken = req.GetPageToken()

	var connType pipelinePB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectors, totalSize, nextPageToken, err := h.service.ListConnectors(ctx, authUser, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
	)))

	resp.Connectors = connectors
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(totalSize)

	return resp, nil

}

func (h *PublicHandler) LookUpConnector(ctx context.Context, req *pipelinePB.LookUpConnectorRequest) (resp *pipelinePB.LookUpConnectorResponse, err error) {

	eventName := "LookUpConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.LookUpConnectorResponse{}

	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return nil, err
	}

	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, lookUpConnectorRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] lookup connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "REQUIRED fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return resp, st.Err()
	}

	connector, err := h.service.GetConnectorByUID(ctx, authUser, connUID, parseView(int32(*req.GetView().Enum())), true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	resp.Connector = connector

	return resp, nil
}

type CreateNamespaceConnectorRequestInterface interface {
	GetParent() string
}

func (h *PublicHandler) CreateUserConnector(ctx context.Context, req *pipelinePB.CreateUserConnectorRequest) (resp *pipelinePB.CreateUserConnectorResponse, err error) {
	resp = &pipelinePB.CreateUserConnectorResponse{}
	resp.Connector, err = h.createNamespaceConnector(ctx, req.Connector, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationConnector(ctx context.Context, req *pipelinePB.CreateOrganizationConnectorRequest) (resp *pipelinePB.CreateOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.CreateOrganizationConnectorResponse{}
	resp.Connector, err = h.createNamespaceConnector(ctx, req.Connector, req)
	return resp, err
}

func (h *PublicHandler) createNamespaceConnector(ctx context.Context, connector *pipelinePB.Connector, req CreateNamespaceConnectorRequestInterface) (connectorCreated *pipelinePB.Connector, err error) {

	eventName := "CreateNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload
	if err := checkfield.CheckCreateOutputOnlyFields(connector, outputOnlyConnectorFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "OUTPUT_ONLY fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(connector, append(createConnectorRequiredFields, immutableConnectorFields...)); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "REQUIRED fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	connID = connector.GetId()
	if len(connID) > 8 && connID[:8] == "instill-" {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "connector",
					Description: "the id can not start with instill-",
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(connID); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "id",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	connector.OwnerName = req.GetParent()

	connectorCreated, err = h.service.CreateNamespaceConnector(ctx, ns, authUser, connector)

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return connectorCreated, nil
}

type ListNamespaceConnectorsRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() pipelinePB.Connector_View
	GetFilter() string
	GetParent() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserConnectors(ctx context.Context, req *pipelinePB.ListUserConnectorsRequest) (resp *pipelinePB.ListUserConnectorsResponse, err error) {
	resp = &pipelinePB.ListUserConnectorsResponse{}
	resp.Connectors, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceConnectors(ctx, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationConnectors(ctx context.Context, req *pipelinePB.ListOrganizationConnectorsRequest) (resp *pipelinePB.ListOrganizationConnectorsResponse, err error) {
	resp = &pipelinePB.ListOrganizationConnectorsResponse{}
	resp.Connectors, resp.NextPageToken, resp.TotalSize, err = h.listNamespaceConnectors(ctx, req)
	return resp, err
}

func (h *PublicHandler) listNamespaceConnectors(ctx context.Context, req ListNamespaceConnectorsRequestInterface) (connectors []*pipelinePB.Connector, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespaceConnectors"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()
	logger, _ := logger.GetZapLogger(ctx)
	pageSize := req.GetPageSize()
	pageToken := req.GetPageToken()

	var connType pipelinePB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	connectors, totalSize, nextPageToken, err = h.service.ListNamespaceConnectors(ctx, ns, authUser, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
	)))

	return connectors, nextPageToken, int32(totalSize), nil

}

type GetNamespaceConnectorRequestInterface interface {
	GetName() string
	GetView() pipelinePB.Connector_View
}

func (h *PublicHandler) GetUserConnector(ctx context.Context, req *pipelinePB.GetUserConnectorRequest) (resp *pipelinePB.GetUserConnectorResponse, err error) {
	resp = &pipelinePB.GetUserConnectorResponse{}
	resp.Connector, err = h.getNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationConnector(ctx context.Context, req *pipelinePB.GetOrganizationConnectorRequest) (resp *pipelinePB.GetOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.GetOrganizationConnectorResponse{}
	resp.Connector, err = h.getNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespaceConnector(ctx context.Context, req GetNamespaceConnectorRequestInterface) (connector *pipelinePB.Connector, err error) {
	eventName := "GetNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	connector, err = h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, parseView(int32(*req.GetView().Enum())), true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	return connector, nil
}

type UpdateNamespaceConnectorRequestInterface interface {
	GetConnector() *pipelinePB.Connector
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserConnector(ctx context.Context, req *pipelinePB.UpdateUserConnectorRequest) (resp *pipelinePB.UpdateUserConnectorResponse, err error) {
	resp = &pipelinePB.UpdateUserConnectorResponse{}
	resp.Connector, err = h.updateNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationConnector(ctx context.Context, req *pipelinePB.UpdateOrganizationConnectorRequest) (resp *pipelinePB.UpdateOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.UpdateOrganizationConnectorResponse{}
	resp.Connector, err = h.updateNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespaceConnector(ctx context.Context, req UpdateNamespaceConnectorRequestInterface) (connector *pipelinePB.Connector, err error) {

	eventName := "UpdateNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var mask fieldmask_utils.Mask

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetConnector().Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbConnectorReq := req.GetConnector()
	pbUpdateMask := req.GetUpdateMask()

	// configuration filed is type google.protobuf.Struct, which needs to be updated as a whole
	for idx, path := range pbUpdateMask.Paths {
		if strings.Contains(path, "configuration") {
			pbUpdateMask.Paths[idx] = "configuration"
		}
	}

	if !pbUpdateMask.IsValid(req.GetConnector()) {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "update_mask",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	// Remove all OUTPUT_ONLY fields in the requested payload
	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyConnectorFields)
	if err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "OUTPUT_ONLY fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	existedConnector, err := h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_FULL, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(req.GetConnector(), existedConnector, immutableConnectorFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "IMMUTABLE fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	mask, err = fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] update connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "update_mask",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	if mask.IsEmpty() {
		existedConnector, err := h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_FULL, true)
		if err != nil {
			span.SetStatus(1, err.Error())
			return nil, err
		}
		return existedConnector, nil
	}

	pbConnectorToUpdate := existedConnector
	if pbConnectorToUpdate.State == pipelinePB.Connector_STATE_CONNECTED {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s", req.GetConnector().Id),
					Description: fmt.Sprintf("Cannot update a connected %s connector", req.GetConnector().Id),
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	dbConnDefID, err := resource.GetRscNameID(existedConnector.GetConnectorDefinitionName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	configuration := &structpb.Struct{}
	h.service.KeepCredentialFieldsWithMaskString(dbConnDefID, pbConnectorToUpdate.Configuration)
	proto.Merge(configuration, pbConnectorToUpdate.Configuration)

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbConnectorReq, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	h.service.RemoveCredentialFieldsWithMaskString(dbConnDefID, req.GetConnector().Configuration)
	proto.Merge(configuration, req.GetConnector().Configuration)
	pbConnectorToUpdate.Configuration = configuration

	connector, err = h.service.UpdateNamespaceConnectorByID(ctx, ns, authUser, connID, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))
	return connector, nil
}

type DeleteNamespaceConnectorRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserConnector(ctx context.Context, req *pipelinePB.DeleteUserConnectorRequest) (resp *pipelinePB.DeleteUserConnectorResponse, err error) {
	resp = &pipelinePB.DeleteUserConnectorResponse{}
	err = h.deleteNamespaceConnector(ctx, req)
	return resp, err
}
func (h *PublicHandler) DeleteOrganizationConnector(ctx context.Context, req *pipelinePB.DeleteOrganizationConnectorRequest) (resp *pipelinePB.DeleteOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.DeleteOrganizationConnectorResponse{}
	err = h.deleteNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) deleteNamespaceConnector(ctx context.Context, req DeleteNamespaceConnectorRequestInterface) (err error) {

	eventName := "DeleteNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	dbConnector, err := h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	if err := h.service.DeleteNamespaceConnectorByID(ctx, ns, authUser, connID); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return err
	}
	return nil
}

type ConnectNamespaceConnectorRequest interface {
	GetName() string
}

func (h *PublicHandler) ConnectUserConnector(ctx context.Context, req *pipelinePB.ConnectUserConnectorRequest) (resp *pipelinePB.ConnectUserConnectorResponse, err error) {
	resp = &pipelinePB.ConnectUserConnectorResponse{}
	resp.Connector, err = h.connectNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) ConnectOrganizationConnector(ctx context.Context, req *pipelinePB.ConnectOrganizationConnectorRequest) (resp *pipelinePB.ConnectOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.ConnectOrganizationConnectorResponse{}
	resp.Connector, err = h.connectNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) connectNamespaceConnector(ctx context.Context, req ConnectNamespaceConnectorRequest) (connector *pipelinePB.Connector, err error) {

	eventName := "ConnectNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, connectConnectorRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] connect connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "REQUIRED fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	connector, err = h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connector.Uid))

	if err != nil {
		st, _ := sterr.CreateErrorBadRequest(
			fmt.Sprintf("[handler] connect connector error %v", err),
			[]*errdetails.BadRequest_FieldViolation{},
		)
		span.SetStatus(1, fmt.Sprintf("connect connector error %v", err))
		return nil, st.Err()
	}
	if *state != pipelinePB.Connector_STATE_CONNECTED {
		st, _ := sterr.CreateErrorBadRequest(
			"[handler] connect connector error not Connector_STATE_CONNECTED",
			[]*errdetails.BadRequest_FieldViolation{},
		)
		span.SetStatus(1, "connect connector error not Connector_STATE_CONNECTED")
		return nil, st.Err()
	}

	connector, err = h.service.UpdateNamespaceConnectorStateByID(ctx, ns, authUser, connID, *state)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	return connector, nil
}

type DisconnectNamespaceConnectorRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DisconnectUserConnector(ctx context.Context, req *pipelinePB.DisconnectUserConnectorRequest) (resp *pipelinePB.DisconnectUserConnectorResponse, err error) {
	resp = &pipelinePB.DisconnectUserConnectorResponse{}
	resp.Connector, err = h.disconnectNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) DisconnectOrganizationConnector(ctx context.Context, req *pipelinePB.DisconnectOrganizationConnectorRequest) (resp *pipelinePB.DisconnectOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.DisconnectOrganizationConnectorResponse{}
	resp.Connector, err = h.disconnectNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) disconnectNamespaceConnector(ctx context.Context, req DisconnectNamespaceConnectorRequestInterface) (connector *pipelinePB.Connector, err error) {

	eventName := "DisconnectNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, disconnectConnectorRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] disconnect connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "REQUIRED fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	connector, err = h.service.UpdateNamespaceConnectorStateByID(ctx, ns, authUser, connID, pipelinePB.Connector_STATE_DISCONNECTED)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	return connector, nil
}

type RenameNamespaceConnectorRequestInterface interface {
	GetName() string
	GetNewConnectorId() string
}

func (h *PublicHandler) RenameUserConnector(ctx context.Context, req *pipelinePB.RenameUserConnectorRequest) (resp *pipelinePB.RenameUserConnectorResponse, err error) {
	resp = &pipelinePB.RenameUserConnectorResponse{}
	resp.Connector, err = h.renameNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) RenameOrganizationConnector(ctx context.Context, req *pipelinePB.RenameOrganizationConnectorRequest) (resp *pipelinePB.RenameOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.RenameOrganizationConnectorResponse{}
	resp.Connector, err = h.renameNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) renameNamespaceConnector(ctx context.Context, req RenameNamespaceConnectorRequestInterface) (connector *pipelinePB.Connector, err error) {

	eventName := "RenameNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connNewID string

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req, renameConnectorRequiredFields); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] rename connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "REQUIRED fields",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	connNewID = req.GetNewConnectorId()
	if len(connNewID) > 8 && connNewID[:8] == "instill-" {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "connector",
					Description: "the id can not start with instill-",
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(connNewID); err != nil {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] rename connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "id",
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
	}

	connector, err = h.service.UpdateNamespaceConnectorIDByID(ctx, ns, authUser, connID, connNewID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	return connector, nil
}

type WatchNamespaceConnectorRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) WatchUserConnector(ctx context.Context, req *pipelinePB.WatchUserConnectorRequest) (resp *pipelinePB.WatchUserConnectorResponse, err error) {
	resp = &pipelinePB.WatchUserConnectorResponse{}
	resp.State, err = h.watchNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) WatchOrganizationConnector(ctx context.Context, req *pipelinePB.WatchOrganizationConnectorRequest) (resp *pipelinePB.WatchOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.WatchOrganizationConnectorResponse{}
	resp.State, err = h.watchNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) watchNamespaceConnector(ctx context.Context, req WatchNamespaceConnectorRequestInterface) (state pipelinePB.Connector_State, err error) {

	eventName := "WatchNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			authUser.UID,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}

	statePtr, err := h.service.GetConnectorState(uuid.FromStringOrNil(connector.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			authUser.UID,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(connector),
		)))
		return pipelinePB.Connector_STATE_ERROR, nil
	}

	return *statePtr, nil
}

type TestNamespaceConnectorRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) TestUserConnector(ctx context.Context, req *pipelinePB.TestUserConnectorRequest) (resp *pipelinePB.TestUserConnectorResponse, err error) {
	resp = &pipelinePB.TestUserConnectorResponse{}
	resp.State, err = h.testNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) TestOrganizationConnector(ctx context.Context, req *pipelinePB.TestOrganizationConnectorRequest) (resp *pipelinePB.TestOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.TestOrganizationConnectorResponse{}
	resp.State, err = h.testNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) testNamespaceConnector(ctx context.Context, req TestNamespaceConnectorRequestInterface) (state pipelinePB.Connector_State, err error) {

	eventName := "TestNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}

	statePtr, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connector.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		return pipelinePB.Connector_STATE_UNSPECIFIED, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		authUser.UID,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	return *statePtr, nil
}

type ExecuteNamespaceConnectorRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
	GetTask() string
}

func (h *PublicHandler) ExecuteUserConnector(ctx context.Context, req *pipelinePB.ExecuteUserConnectorRequest) (resp *pipelinePB.ExecuteUserConnectorResponse, err error) {
	resp = &pipelinePB.ExecuteUserConnectorResponse{}
	resp.Outputs, err = h.executeNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) ExecuteOrganizationConnector(ctx context.Context, req *pipelinePB.ExecuteOrganizationConnectorRequest) (resp *pipelinePB.ExecuteOrganizationConnectorResponse, err error) {
	resp = &pipelinePB.ExecuteOrganizationConnectorResponse{}
	resp.Outputs, err = h.executeNamespaceConnector(ctx, req)
	return resp, err
}

func (h *PublicHandler) executeNamespaceConnector(ctx context.Context, req ExecuteNamespaceConnectorRequestInterface) (outputs []*structpb.Struct, err error) {

	startTime := time.Now()
	eventName := "ExecuteNamespaceConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	authUser, err := h.service.AuthenticateUser(ctx, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, authUser, connID, service.VIEW_FULL, true)
	if err != nil {
		return nil, err
	}
	if connector.Tombstone {
		st, _ := sterr.CreateErrorPreconditionFailure(
			"ExecuteConnector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s", connID),
					Description: "the connector definition is deprecated, you can not use it anymore",
				},
			})
		return nil, st.Err()
	}

	var ownerType mgmtPB.OwnerType
	switch ns.NsType {
	case resource.Organization:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION
	case resource.User:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_USER
	default:
		ownerType = mgmtPB.OwnerType_OWNER_TYPE_UNSPECIFIED
	}

	dataPoint := utils.ConnectorUsageMetricData{
		OwnerUID:               ns.NsUid.String(),
		OwnerType:              ownerType,
		UserUID:                authUser.UID.String(),
		UserType:               mgmtPB.OwnerType_OWNER_TYPE_USER, // TODO: currently only support /users type, will change after beta
		ConnectorID:            connector.Id,
		ConnectorUID:           connector.Uid,
		ConnectorExecuteUID:    logUUID.String(),
		ConnectorDefinitionUid: connector.ConnectorDefinition.Uid,
		ExecuteTime:            startTime.Format(time.RFC3339Nano),
	}

	md, _ := metadata.FromIncomingContext(ctx)

	pipelineVal := &structpb.Value{}
	if len(md.Get("id")) > 0 &&
		len(md.Get("uid")) > 0 &&
		len(md.Get("release_id")) > 0 &&
		len(md.Get("release_uid")) > 0 &&
		len(md.Get("owner")) > 0 &&
		len(md.Get("trigger_id")) > 0 {
		pipelineVal, _ = structpb.NewValue(map[string]interface{}{
			"id":          md.Get("id")[0],
			"uid":         md.Get("uid")[0],
			"release_id":  md.Get("release_id")[0],
			"release_uid": md.Get("release_uid")[0],
			"owner":       md.Get("owner")[0],
			"trigger_id":  md.Get("trigger_id")[0],
		})
	}

	if outputs, err = h.service.Execute(ctx, ns, authUser, connID, req.GetTask(), req.GetInputs()); err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewConnectorDataPoint(ctx, dataPoint, pipelineVal)
		return nil, err
	} else {

		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			authUser.UID,
			eventName,
		)))
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED
		if err := h.service.WriteNewConnectorDataPoint(ctx, dataPoint, pipelineVal); err != nil {
			logger.Warn("usage and metric data write fail")
		}
	}
	return outputs, nil

}
