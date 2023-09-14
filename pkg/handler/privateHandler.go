package handler

import (
	"context"
	"fmt"

	"github.com/gogo/status"
	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/operator"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// PrivateHandler handles private API
type PrivateHandler struct {
	pipelinePB.UnimplementedPipelinePrivateServiceServer
	service  service.Service
	operator operator.Operator
}

// NewPrivateHandler initiates a handler instance
func NewPrivateHandler(ctx context.Context, s service.Service) pipelinePB.PipelinePrivateServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PrivateHandler{
		service:  s,
		operator: operator.InitOperator(),
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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelinesAdmin(ctx, int64(req.GetPageSize()), req.GetPageToken(), parseView(req.GetView()), filter, req.GetShowDeleted())
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
	if err := checkfield.CheckRequiredFields(req, lookUpRequiredFields); err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	view := pipelinePB.View_VIEW_BASIC
	if req.GetView() != pipelinePB.View_VIEW_UNSPECIFIED {
		view = req.GetView()
	}

	uid, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, err
	}
	pbPipeline, err := h.service.GetPipelineByUIDAdmin(ctx, uid, view)
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

	dbDef, err := h.operator.GetOperatorDefinitionById(connID)
	if err != nil {
		return resp, err
	}
	resp.OperatorDefinition = proto.Clone(dbDef).(*pipelinePB.OperatorDefinition)
	if parseView(req.GetView()) == pipelinePB.View_VIEW_BASIC {
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

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListPipelineReleasesAdmin(ctx, int64(req.GetPageSize()), req.GetPageToken(), parseView(req.GetView()), filter, req.GetShowDeleted())
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
