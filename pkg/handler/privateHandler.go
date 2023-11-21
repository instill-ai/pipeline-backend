package handler

import (
	"context"
	"fmt"

	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// PrivateHandler handles private API
type PrivateHandler struct {
	pipelinePB.UnimplementedPipelinePrivateServiceServer
	service service.Service
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(ctx context.Context, s service.Service) pipelinePB.PipelinePrivateServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PrivateHandler{
		service: s,
	}
}

// GetService returns the service
func (h *PrivateHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PrivateHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PrivateHandler) ListPipelinesAdmin(ctx context.Context, req *pipelinePB.ListPipelinesAdminRequest) (*pipelinePB.ListPipelinesAdminResponse, error) {

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		// only support "recipe.components.resource_name" for now
		filtering.DeclareIdent("recipe", filtering.TypeMap(filtering.TypeString, filtering.TypeMap(filtering.TypeString, filtering.TypeString))),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("create_time", filtering.TypeTimestamp),
		filtering.DeclareIdent("update_time", filtering.TypeTimestamp),
	}...)
	if err != nil {
		return &pipelinePB.ListPipelinesAdminResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pipelinePB.ListPipelinesAdminResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelinesAdmin(ctx, int64(req.GetPageSize()), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		return &pipelinePB.ListPipelinesAdminResponse{}, err
	}

	resp := pipelinePB.ListPipelinesAdminResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

func (h *PrivateHandler) LookUpPipelineAdmin(ctx context.Context, req *pipelinePB.LookUpPipelineAdminRequest) (*pipelinePB.LookUpPipelineAdminResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpPipelineRequiredFields); err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	view := pipelinePB.LookUpPipelineAdminRequest_VIEW_BASIC
	if req.GetView() != pipelinePB.LookUpPipelineAdminRequest_VIEW_UNSPECIFIED {
		view = req.GetView()
	}

	uid, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, err
	}
	pbPipeline, err := h.service.GetPipelineByUIDAdmin(ctx, uid, service.View(view))
	if err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, err
	}

	resp := pipelinePB.LookUpPipelineAdminResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *PrivateHandler) LookUpOperatorDefinitionAdmin(ctx context.Context, req *pipelinePB.LookUpOperatorDefinitionAdminRequest) (resp *pipelinePB.LookUpOperatorDefinitionAdminResponse, err error) {

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.LookUpOperatorDefinitionAdminResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetPermalink()); err != nil {
		return resp, err
	}

	dbDef, err := h.service.GetOperatorDefinitionById(ctx, connID)
	if err != nil {
		return resp, err
	}
	resp.OperatorDefinition = proto.Clone(dbDef).(*pipelinePB.OperatorDefinition)
	if parseView(int32(*req.GetView().Enum())) == service.VIEW_BASIC {
		resp.OperatorDefinition.Spec = nil
	}
	resp.OperatorDefinition.Name = fmt.Sprintf("operator-definitions/%s", resp.OperatorDefinition.GetId())

	logger.Info("GetOperatorDefinitionAdmin")
	return resp, nil
}

func (h *PrivateHandler) ListPipelineReleasesAdmin(ctx context.Context, req *pipelinePB.ListPipelineReleasesAdminRequest) (*pipelinePB.ListPipelineReleasesAdminResponse, error) {

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		// only support "recipe.components.resource_name" for now
		filtering.DeclareIdent("recipe", filtering.TypeMap(filtering.TypeString, filtering.TypeMap(filtering.TypeString, filtering.TypeString))),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("create_time", filtering.TypeTimestamp),
		filtering.DeclareIdent("update_time", filtering.TypeTimestamp),
	}...)
	if err != nil {
		return &pipelinePB.ListPipelineReleasesAdminResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pipelinePB.ListPipelineReleasesAdminResponse{}, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListPipelineReleasesAdmin(ctx, int64(req.GetPageSize()), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		return &pipelinePB.ListPipelineReleasesAdminResponse{}, err
	}

	resp := pipelinePB.ListPipelineReleasesAdminResponse{
		Releases:      pbPipelineReleases,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

func (h *PrivateHandler) ListConnectorsAdmin(ctx context.Context, req *pipelinePB.ListConnectorsAdminRequest) (resp *pipelinePB.ListConnectorsAdminResponse, err error) {

	var pageSize int64
	var pageToken string

	resp = &pipelinePB.ListConnectorsAdminResponse{}
	pageSize = int64(req.GetPageSize())
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

	connectorResources, totalSize, nextPageToken, err := h.service.ListConnectorsAdmin(ctx, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		return nil, err
	}

	resp.Connectors = connectorResources
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

	connectorResource, err := h.service.GetConnectorByUIDAdmin(ctx, connUID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		return nil, err
	}

	resp.Connector = connectorResource

	return resp, nil
}

func (h *PrivateHandler) CheckConnector(ctx context.Context, req *pipelinePB.CheckConnectorRequest) (resp *pipelinePB.CheckConnectorResponse, err error) {

	resp = &pipelinePB.CheckConnectorResponse{}
	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}

	connectorResource, err := h.service.GetConnectorByUIDAdmin(ctx, connUID, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	if connectorResource.Tombstone {
		resp.State = pipelinePB.Connector_STATE_ERROR
		return resp, nil
	}

	if connectorResource.State == pipelinePB.Connector_STATE_CONNECTED {
		state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connectorResource.Uid))
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

func (h *PrivateHandler) LookUpConnectorDefinitionAdmin(ctx context.Context, req *pipelinePB.LookUpConnectorDefinitionAdminRequest) (resp *pipelinePB.LookUpConnectorDefinitionAdminResponse, err error) {

	resp = &pipelinePB.LookUpConnectorDefinitionAdminResponse{}

	connUID, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return resp, err
	}

	// TODO add a service wrapper
	def, err := h.service.GetConnectorDefinitionByUIDAdmin(ctx, connUID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		return resp, err
	}
	resp.ConnectorDefinition = def

	return resp, nil
}
