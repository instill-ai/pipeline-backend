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
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
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

	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectors, totalSize, nextPageToken, err := h.service.ListConnectors(ctx, userUid, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
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

	_, userUid, err := h.service.GetCtxUser(ctx)
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

	connector, err := h.service.GetConnectorByUID(ctx, userUid, connUID, parseView(int32(*req.GetView().Enum())), true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	resp.Connector = connector

	return resp, nil
}

type CreateOrganizationConnectorRequestInterface interface {
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

func (h *PublicHandler) createNamespaceConnector(ctx context.Context, connector *pipelinePB.Connector, req CreateOrganizationConnectorRequestInterface) (connectorCreated *pipelinePB.Connector, err error) {

	eventName := "createNamespaceConnector"

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
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// TODO: ACL
	if ns.NsType == resource.User && ns.String() != resource.UserUidToUserPermalink(userUid) {
		st, err := sterr.CreateErrorBadRequest(
			"[handler] create connector error",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Description: "can not create in other user's namespace",
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		span.SetStatus(1, st.Err().Error())
		return nil, st.Err()
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

	if strings.HasPrefix(req.GetParent(), "users") {
		connector.Owner = &pipelinePB.Connector_User{User: req.GetParent()}
	} else {
		connector.Owner = &pipelinePB.Connector_Organization{Organization: req.GetParent()}
	}

	connectorCreated, err = h.service.CreateNamespaceConnector(ctx, ns, userUid, connector)

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
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

	eventName := "listNamespaceConnectors"

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
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	connectors, totalSize, nextPageToken, err = h.service.ListNamespaceConnectors(ctx, ns, userUid, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
	)))

	return connectors, nextPageToken, int32(totalSize), nil

}

type GetUserConnectorRequestInterface interface {
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

func (h *PublicHandler) getNamespaceConnector(ctx context.Context, req GetUserConnectorRequestInterface) (connector *pipelinePB.Connector, err error) {
	eventName := "getNamespaceConnector"

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
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	connector, err = h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, parseView(int32(*req.GetView().Enum())), true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	return connector, nil
}

func (h *PublicHandler) UpdateUserConnector(ctx context.Context, req *pipelinePB.UpdateUserConnectorRequest) (resp *pipelinePB.UpdateUserConnectorResponse, err error) {

	eventName := "UpdateUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var mask fieldmask_utils.Mask

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Connector.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	resp = &pipelinePB.UpdateUserConnectorResponse{}

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
		return resp, st.Err()
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
		return resp, st.Err()
	}

	existedConnector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_FULL, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
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
		return resp, st.Err()
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
		return resp, st.Err()
	}

	if mask.IsEmpty() {
		existedConnector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_FULL, true)
		if err != nil {
			span.SetStatus(1, err.Error())
			return resp, err
		}
		return &pipelinePB.UpdateUserConnectorResponse{
			Connector: existedConnector,
		}, nil
	}

	pbConnectorToUpdate := existedConnector
	if pbConnectorToUpdate.State == pipelinePB.Connector_STATE_CONNECTED {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[service] update connector",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "UPDATE",
					Subject:     fmt.Sprintf("id %s", req.Connector.Id),
					Description: fmt.Sprintf("Cannot update a connected %s connector", req.Connector.Id),
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
		return resp, err
	}
	configuration := &structpb.Struct{}
	h.service.KeepCredentialFieldsWithMaskString(dbConnDefID, pbConnectorToUpdate.Configuration)
	proto.Merge(configuration, pbConnectorToUpdate.Configuration)

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbConnectorReq, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	h.service.RemoveCredentialFieldsWithMaskString(dbConnDefID, req.Connector.Configuration)
	proto.Merge(configuration, req.Connector.Configuration)
	pbConnectorToUpdate.Configuration = configuration

	connector, err := h.service.UpdateNamespaceConnectorByID(ctx, ns, userUid, connID, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	resp.Connector = connector

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))
	return resp, nil
}

func (h *PublicHandler) DeleteUserConnector(ctx context.Context, req *pipelinePB.DeleteUserConnectorRequest) (resp *pipelinePB.DeleteUserConnectorResponse, err error) {

	eventName := "DeleteUserConnectorByID"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	// Cast all used types and data

	resp = &pipelinePB.DeleteUserConnectorResponse{}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	if err := h.service.DeleteNamespaceConnectorByID(ctx, ns, userUid, connID); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(dbConnector),
	)))

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return resp, err
	}
	return resp, nil
}

func (h *PublicHandler) ConnectUserConnector(ctx context.Context, req *pipelinePB.ConnectUserConnectorRequest) (resp *pipelinePB.ConnectUserConnectorResponse, err error) {

	eventName := "ConnectUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &pipelinePB.ConnectUserConnectorResponse{}

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
		return resp, st.Err()
	}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connector.Uid))

	if err != nil {
		st, _ := sterr.CreateErrorBadRequest(
			fmt.Sprintf("[handler] connect connector error %v", err),
			[]*errdetails.BadRequest_FieldViolation{},
		)
		span.SetStatus(1, fmt.Sprintf("connect connector error %v", err))
		return resp, st.Err()
	}
	if *state != pipelinePB.Connector_STATE_CONNECTED {
		st, _ := sterr.CreateErrorBadRequest(
			"[handler] connect connector error not Connector_STATE_CONNECTED",
			[]*errdetails.BadRequest_FieldViolation{},
		)
		span.SetStatus(1, "connect connector error not Connector_STATE_CONNECTED")
		return resp, st.Err()
	}

	connector, err = h.service.UpdateNamespaceConnectorStateByID(ctx, ns, userUid, connID, *state)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	resp.Connector = connector

	return resp, nil
}

func (h *PublicHandler) DisconnectUserConnector(ctx context.Context, req *pipelinePB.DisconnectUserConnectorRequest) (resp *pipelinePB.DisconnectUserConnectorResponse, err error) {

	eventName := "DisconnectUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &pipelinePB.DisconnectUserConnectorResponse{}

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
		return resp, st.Err()
	}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connector, err := h.service.UpdateNamespaceConnectorStateByID(ctx, ns, userUid, connID, pipelinePB.Connector_STATE_DISCONNECTED)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	resp.Connector = connector

	return resp, nil
}

func (h *PublicHandler) RenameUserConnector(ctx context.Context, req *pipelinePB.RenameUserConnectorRequest) (resp *pipelinePB.RenameUserConnectorResponse, err error) {

	eventName := "RenameUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string
	var connNewID string

	resp = &pipelinePB.RenameUserConnectorResponse{}

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
		return resp, st.Err()
	}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
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
		return resp, st.Err()
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
		return resp, st.Err()
	}

	connector, err := h.service.UpdateNamespaceConnectorIDByID(ctx, ns, userUid, connID, connNewID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	resp.Connector = connector
	return resp, nil
}

func (h *PublicHandler) WatchUserConnector(ctx context.Context, req *pipelinePB.WatchUserConnectorRequest) (resp *pipelinePB.WatchUserConnectorResponse, err error) {

	eventName := "WatchUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &pipelinePB.WatchUserConnectorResponse{}
	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
		)))
		return resp, err
	}

	state, err := h.service.GetConnectorState(uuid.FromStringOrNil(connector.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(connector),
		)))
		state = pipelinePB.Connector_STATE_ERROR.Enum()
	}

	resp.State = *state

	return resp, nil
}

func (h *PublicHandler) TestUserConnector(ctx context.Context, req *pipelinePB.TestUserConnectorRequest) (resp *pipelinePB.TestUserConnectorResponse, err error) {

	eventName := "TestUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &pipelinePB.TestUserConnectorResponse{}
	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connector.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connector),
	)))

	resp.State = *state

	return resp, nil
}

func (h *PublicHandler) ExecuteUserConnector(ctx context.Context, req *pipelinePB.ExecuteUserConnectorRequest) (resp *pipelinePB.ExecuteUserConnectorResponse, err error) {

	startTime := time.Now()
	eventName := "ExecuteUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.ExecuteUserConnectorResponse{}
	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connector, err := h.service.GetNamespaceConnectorByID(ctx, ns, userUid, connID, service.VIEW_FULL, true)
	if err != nil {
		return resp, err
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
		return resp, st.Err()
	}

	dataPoint := utils.ConnectorUsageMetricData{
		OwnerUID:               userUid.String(),
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

	if outputs, err := h.service.Execute(ctx, ns, userUid, connID, req.GetTask(), req.GetInputs()); err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewConnectorDataPoint(ctx, dataPoint, pipelineVal)
		return nil, err
	} else {
		resp.Outputs = outputs
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
		)))
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED
		if err := h.service.WriteNewConnectorDataPoint(ctx, dataPoint, pipelineVal); err != nil {
			logger.Warn("usage and metric data write fail")
		}
	}
	return resp, nil

}
