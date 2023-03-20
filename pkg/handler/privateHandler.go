package handler

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type privateHandler struct {
	pipelinePB.UnimplementedPipelinePrivateServiceServer
	service service.Service
}

// NewPublicHandler initiates a handler instance
func NewPrivateHandler(s service.Service) pipelinePB.PipelinePrivateServiceServer {
	datamodel.InitJSONSchema()
	return &privateHandler{
		service: s,
	}
}

func (h *privateHandler) ListPipelinesAdmin(ctx context.Context, req *pipelinePB.ListPipelinesAdminRequest) (*pipelinePB.ListPipelinesAdminResponse, error) {

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	var mode pipelinePB.Pipeline_Mode
	var state pipelinePB.Pipeline_State
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		filtering.DeclareIdent("recipe", filtering.TypeMap(filtering.TypeString, filtering.TypeString)),
		filtering.DeclareEnumIdent("mode", mode.Type()),
		filtering.DeclareEnumIdent("state", state.Type()),
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

	dbPipelines, totalSize, nextPageToken, err := h.service.ListPipelinesAdmin(req.GetPageSize(), req.GetPageToken(), isBasicView, filter)
	if err != nil {
		return &pipelinePB.ListPipelinesAdminResponse{}, err
	}

	pbPipelines := []*pipelinePB.Pipeline{}
	for _, dbPipeline := range dbPipelines {
		pbPipelines = append(pbPipelines, DBToPBPipeline(&dbPipeline))
	}

	resp := pipelinePB.ListPipelinesAdminResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *privateHandler) GetPipelineAdmin(ctx context.Context, req *pipelinePB.GetPipelineAdminRequest) (*pipelinePB.GetPipelineAdminResponse, error) {

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return &pipelinePB.GetPipelineAdminResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByIDAdmin(id, isBasicView)
	if err != nil {
		return &pipelinePB.GetPipelineAdminResponse{}, err
	}

	pbPipeline := DBToPBPipeline(dbPipeline)
	resp := pipelinePB.GetPipelineAdminResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *privateHandler) LookUpPipelineAdmin(ctx context.Context, req *pipelinePB.LookUpPipelineAdminRequest) (*pipelinePB.LookUpPipelineAdminResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpRequiredFields); err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	uidStr, err := resource.GetPermalinkUID(req.GetPermalink())
	if err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, err
	}

	uid, err := uuid.FromString(uidStr)
	if err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByUIDAdmin(uid, isBasicView)
	if err != nil {
		return &pipelinePB.LookUpPipelineAdminResponse{}, err
	}

	pbPipeline := DBToPBPipeline(dbPipeline)
	resp := pipelinePB.LookUpPipelineAdminResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}
