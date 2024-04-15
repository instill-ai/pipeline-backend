package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"strconv"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PrivateHandler) ListPipelinesAdmin(ctx context.Context, req *pipelinePB.ListPipelinesAdminRequest) (*pipelinePB.ListPipelinesAdminResponse, error) {

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelinesAdmin(ctx, req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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
		return &pipelinePB.LookUpPipelineAdminResponse{}, ErrCheckRequiredFields
	}

	view := pipelinePB.Pipeline_VIEW_BASIC
	if req.GetView() != pipelinePB.Pipeline_VIEW_UNSPECIFIED {
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

func (h *PublicHandler) ListPipelines(ctx context.Context, req *pipelinePB.ListPipelinesRequest) (*pipelinePB.ListPipelinesResponse, error) {

	eventName := "ListPipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
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
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(
		ctx, req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), req.Visibility, filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	resp := pipelinePB.ListPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

type CreateNamespacePipelineRequestInterface interface {
	GetPipeline() *pipelinePB.Pipeline
	GetParent() string
}

func (h *PublicHandler) CreateUserPipeline(ctx context.Context, req *pipelinePB.CreateUserPipelineRequest) (resp *pipelinePB.CreateUserPipelineResponse, err error) {
	resp = &pipelinePB.CreateUserPipelineResponse{}
	resp.Pipeline, err = h.createNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationPipeline(ctx context.Context, req *pipelinePB.CreateOrganizationPipelineRequest) (resp *pipelinePB.CreateOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.CreateOrganizationPipelineResponse{}
	resp.Pipeline, err = h.createNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespacePipeline(ctx context.Context, req CreateNamespacePipelineRequestInterface) (pipeline *pipelinePB.Pipeline, err error) {

	eventName := "CreateNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Validate JSON Schema
	// if err := datamodel.ValidatePipelineJSONSchema(req.GetPipeline()); err != nil {
	// 	span.SetStatus(1, err.Error())
	// 	return nil, status.Error(codes.InvalidArgument, err.Error())
	// }

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.GetPipeline(), append(createPipelineRequiredFields, immutablePipelineFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetPipeline(), outputOnlyPipelineFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.GetPipeline().GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrResourceID
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipelineToCreate := req.GetPipeline()

	pipeline, err = h.service.CreateNamespacePipeline(ctx, ns, pipelineToCreate)

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pipeline),
	)))

	return pipeline, nil
}

type ListNamespacePipelinesRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() pipelinePB.Pipeline_View
	GetVisibility() pipelinePB.Pipeline_Visibility
	GetFilter() string
	GetParent() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserPipelines(ctx context.Context, req *pipelinePB.ListUserPipelinesRequest) (resp *pipelinePB.ListUserPipelinesResponse, err error) {
	resp = &pipelinePB.ListUserPipelinesResponse{}
	resp.Pipelines, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelines(ctx, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationPipelines(ctx context.Context, req *pipelinePB.ListOrganizationPipelinesRequest) (resp *pipelinePB.ListOrganizationPipelinesResponse, err error) {
	resp = &pipelinePB.ListOrganizationPipelinesResponse{}
	resp.Pipelines, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelines(ctx, req)
	return resp, err
}

func (h *PublicHandler) listNamespacePipelines(ctx context.Context, req ListNamespacePipelinesRequestInterface) (pipelines []*pipelinePB.Pipeline, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespacePipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, _, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
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
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}
	visibility := req.GetVisibility()

	pbPipelines, totalSize, nextPageToken, err := h.service.ListNamespacePipelines(ctx, ns, req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), &visibility, filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbPipelines, nextPageToken, int32(totalSize), nil
}

type GetNamespacePipelineRequestInterface interface {
	GetName() string
	GetView() pipelinePB.Pipeline_View
}

func (h *PublicHandler) GetUserPipeline(ctx context.Context, req *pipelinePB.GetUserPipelineRequest) (resp *pipelinePB.GetUserPipelineResponse, err error) {
	resp = &pipelinePB.GetUserPipelineResponse{}
	resp.Pipeline, err = h.getNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationPipeline(ctx context.Context, req *pipelinePB.GetOrganizationPipelineRequest) (resp *pipelinePB.GetOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.GetOrganizationPipelineResponse{}
	resp.Pipeline, err = h.getNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespacePipeline(ctx context.Context, req GetNamespacePipelineRequestInterface) (*pipelinePB.Pipeline, error) {

	eventName := "GetNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, id, parseView(int32(*req.GetView().Enum())))

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return pbPipeline, nil
}

type UpdateNamespacePipelineRequestInterface interface {
	GetPipeline() *pipelinePB.Pipeline
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserPipeline(ctx context.Context, req *pipelinePB.UpdateUserPipelineRequest) (resp *pipelinePB.UpdateUserPipelineResponse, err error) {
	resp = &pipelinePB.UpdateUserPipelineResponse{}
	resp.Pipeline, err = h.updateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationPipeline(ctx context.Context, req *pipelinePB.UpdateOrganizationPipelineRequest) (resp *pipelinePB.UpdateOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.UpdateOrganizationPipelineResponse{}
	resp.Pipeline, err = h.updateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespacePipeline(ctx context.Context, req UpdateNamespacePipelineRequestInterface) (*pipelinePB.Pipeline, error) {

	eventName := "UpdateNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	// logUUID, _ := uuid.NewV4()

	// logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetPipeline().Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineReq := req.GetPipeline()
	pbUpdateMask := req.GetUpdateMask()

	// metadata field is type google.protobuf.Struct, which needs to be updated as a whole
	for idx, path := range pbUpdateMask.Paths {
		if strings.Contains(path, "metadata") {
			pbUpdateMask.Paths[idx] = "metadata"
		}
		if strings.Contains(path, "recipe") {
			pbUpdateMask.Paths[idx] = "recipe"
		}
	}
	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReq) {
		return nil, ErrUpdateMask
	}

	getResp, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: pbPipelineReq.GetName(), View: pipelinePB.Pipeline_VIEW_RECIPE.Enum()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyPipelineFields)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrFieldMask
	}

	if mask.IsEmpty() {
		return getResp.GetPipeline(), nil
	}

	pbPipelineToUpdate := getResp.GetPipeline()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReq, pbPipelineToUpdate, immutablePipelineFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReq, pbPipelineToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.UpdateNamespacePipelineByID(ctx, ns, id, pbPipelineToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// logger.Info(string(custom_otel.NewLogMessage(
	// 	span,
	// 	logUUID.String(),
	// 	authUser.UID,
	// 	eventName,
	// 	custom_otel.SetEventResource(pbPipeline),
	// )))

	return pbPipeline, nil
}

type DeleteNamespacePipelineRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserPipeline(ctx context.Context, req *pipelinePB.DeleteUserPipelineRequest) (resp *pipelinePB.DeleteUserPipelineResponse, err error) {
	resp = &pipelinePB.DeleteUserPipelineResponse{}
	err = h.deleteNamespacePipeline(ctx, req)
	return resp, err
}
func (h *PublicHandler) DeleteOrganizationPipeline(ctx context.Context, req *pipelinePB.DeleteOrganizationPipelineRequest) (resp *pipelinePB.DeleteOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.DeleteOrganizationPipelineResponse{}
	err = h.deleteNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) deleteNamespacePipeline(ctx context.Context, req DeleteNamespacePipelineRequestInterface) error {

	eventName := "DeleteNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}
	existPipeline, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	if err := h.service.DeleteNamespacePipelineByID(ctx, ns, id); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(existPipeline.GetPipeline()),
	)))

	return nil
}

func (h *PublicHandler) LookUpPipeline(ctx context.Context, req *pipelinePB.LookUpPipelineRequest) (*pipelinePB.LookUpPipelineResponse, error) {

	eventName := "LookUpPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpPipelineRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckRequiredFields
	}

	uid, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.GetPipelineByUID(ctx, uid, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.LookUpPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

type ValidateNamespacePipelineRequest interface {
	GetName() string
}

func (h *PublicHandler) ValidateUserPipeline(ctx context.Context, req *pipelinePB.ValidateUserPipelineRequest) (resp *pipelinePB.ValidateUserPipelineResponse, err error) {
	resp = &pipelinePB.ValidateUserPipelineResponse{}
	resp.Pipeline, err = h.validateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) ValidateOrganizationPipeline(ctx context.Context, req *pipelinePB.ValidateOrganizationPipelineRequest) (resp *pipelinePB.ValidateOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.ValidateOrganizationPipelineResponse{}
	resp.Pipeline, err = h.validateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) validateNamespacePipeline(ctx context.Context, req ValidateNamespacePipelineRequest) (*pipelinePB.Pipeline, error) {

	eventName := "ValidateNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.ValidateNamespacePipelineByID(ctx, ns, id)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return pbPipeline, nil
}

type RenameNamespacePipelineRequestInterface interface {
	GetName() string
	GetNewPipelineId() string
}

func (h *PublicHandler) RenameUserPipeline(ctx context.Context, req *pipelinePB.RenameUserPipelineRequest) (resp *pipelinePB.RenameUserPipelineResponse, err error) {
	resp = &pipelinePB.RenameUserPipelineResponse{}
	resp.Pipeline, err = h.renameNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) RenameOrganizationPipeline(ctx context.Context, req *pipelinePB.RenameOrganizationPipelineRequest) (resp *pipelinePB.RenameOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.RenameOrganizationPipelineResponse{}
	resp.Pipeline, err = h.renameNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) renameNamespacePipeline(ctx context.Context, req RenameNamespacePipelineRequestInterface) (*pipelinePB.Pipeline, error) {

	eventName := "RenameNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, renamePipelineRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckRequiredFields
	}

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	newID := req.GetNewPipelineId()
	if err := checkfield.CheckResourceID(newID); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrResourceID
	}

	pbPipeline, err := h.service.UpdateNamespacePipelineIDByID(ctx, ns, id, newID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return pbPipeline, nil
}

type CloneNamespacePipelineRequestInterface interface {
	GetName() string
	GetTarget() string
}

func (h *PublicHandler) CloneUserPipeline(ctx context.Context, req *pipelinePB.CloneUserPipelineRequest) (resp *pipelinePB.CloneUserPipelineResponse, err error) {
	resp = &pipelinePB.CloneUserPipelineResponse{}
	resp.Pipeline, err = h.cloneNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) CloneOrganizationPipeline(ctx context.Context, req *pipelinePB.CloneOrganizationPipelineRequest) (resp *pipelinePB.CloneOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.CloneOrganizationPipelineResponse{}
	resp.Pipeline, err = h.cloneNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) cloneNamespacePipeline(ctx context.Context, req CloneNamespacePipelineRequestInterface) (*pipelinePB.Pipeline, error) {

	eventName := "CloneNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	targetNS, targetID, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetTarget())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.CloneNamespacePipeline(ctx, ns, id, targetNS, targetID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))
	return pbPipeline, nil
}

func (h *PublicHandler) preTriggerUserPipeline(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pipelinePB.Pipeline, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", nil, false, ErrCheckRequiredFields
	}

	ns, id, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetName())
	if err != nil {
		return ns, id, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, id, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, id, service.ViewFull)
	if err != nil {
		return ns, id, nil, false, err
	}
	// _, err = h.service.ValidateNamespacePipelineByID(ctx, ns,  id)
	// if err != nil {
	// 	return ns, nil, id, nil, false, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }
	returnTraces := false
	if resource.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true" {
		returnTraces = true
	}

	return ns, id, pbPipeline, returnTraces, nil

}

type TriggerNamespacePipelineRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
}

func (h *PublicHandler) TriggerUserPipeline(ctx context.Context, req *pipelinePB.TriggerUserPipelineRequest) (resp *pipelinePB.TriggerUserPipelineResponse, err error) {
	resp = &pipelinePB.TriggerUserPipelineResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerOrganizationPipeline(ctx context.Context, req *pipelinePB.TriggerOrganizationPipelineRequest) (resp *pipelinePB.TriggerOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.TriggerOrganizationPipelineResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) triggerNamespacePipeline(ctx context.Context, req TriggerNamespacePipelineRequestInterface) (outputs []*structpb.Struct, metadata *pipelinePB.TriggerMetadata, err error) {

	eventName := "TriggerNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	// logger, _ := logger.GetZapLogger(ctx)

	ns, id, _, returnTraces, err := h.preTriggerUserPipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, nil, err
	}

	outputs, metadata, err = h.service.TriggerNamespacePipelineByID(ctx, ns, id, req.GetInputs(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, nil, err
	}

	// logger.Info(string(custom_otel.NewLogMessage(
	// 	span,
	// 	logUUID.String(),
	// 	authUser.UID,
	// 	eventName,
	// 	custom_otel.SetEventResource(pbPipeline),
	// )))

	return outputs, metadata, nil
}

type TriggerAsyncNamespacePipelineRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
}

func (h *PublicHandler) TriggerAsyncUserPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncUserPipelineRequest) (resp *pipelinePB.TriggerAsyncUserPipelineResponse, err error) {
	resp = &pipelinePB.TriggerAsyncUserPipelineResponse{}
	resp.Operation, err = h.triggerAsyncNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerAsyncOrganizationPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncOrganizationPipelineRequest) (resp *pipelinePB.TriggerAsyncOrganizationPipelineResponse, err error) {
	resp = &pipelinePB.TriggerAsyncOrganizationPipelineResponse{}
	resp.Operation, err = h.triggerAsyncNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) triggerAsyncNamespacePipeline(ctx context.Context, req TriggerAsyncNamespacePipelineRequestInterface) (operation *longrunningpb.Operation, err error) {

	eventName := "TriggerAsyncNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, dbPipeline, returnTraces, err := h.preTriggerUserPipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	operation, err = h.service.TriggerAsyncNamespacePipelineByID(ctx, ns, id, req.GetInputs(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(dbPipeline),
	)))

	return operation, nil
}

type CreateNamespacePipelineReleaseRequestInterface interface {
	GetRelease() *pipelinePB.PipelineRelease
	GetParent() string
}

func (h *PublicHandler) CreateUserPipelineRelease(ctx context.Context, req *pipelinePB.CreateUserPipelineReleaseRequest) (resp *pipelinePB.CreateUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.CreateUserPipelineReleaseResponse{}
	resp.Release, err = h.createNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.CreateOrganizationPipelineReleaseRequest) (resp *pipelinePB.CreateOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.CreateOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.createNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespacePipelineRelease(ctx context.Context, req CreateNamespacePipelineReleaseRequestInterface) (*pipelinePB.PipelineRelease, error) {
	eventName := "CreateNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.GetRelease(), append(releaseCreateRequiredFields, immutablePipelineFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetRelease(), releaseOutputOnlyFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	// Return error if resource ID does not a semantic version
	if !semver.IsValid(req.GetRelease().GetId()) {
		span.SetStatus(1, ErrSematicVersion.Error())
		return nil, ErrSematicVersion
	}

	ns, pipelineID, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return nil, err
	}
	_, err = h.service.ValidateNamespacePipelineByID(ctx, ns, pipeline.Id)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	pbPipelineRelease, err := h.service.CreateNamespacePipelineRelease(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.GetRelease())
	if err != nil {
		span.SetStatus(1, err.Error())
		// Manually set the custom header to have a StatusBadRequest http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusBadRequest))); err != nil {
			return nil, err
		}
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil

}

type ListNamespacePipelineReleasesRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() pipelinePB.Pipeline_View
	GetFilter() string
	GetParent() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserPipelineReleases(ctx context.Context, req *pipelinePB.ListUserPipelineReleasesRequest) (resp *pipelinePB.ListUserPipelineReleasesResponse, err error) {
	resp = &pipelinePB.ListUserPipelineReleasesResponse{}
	resp.Releases, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelineReleases(ctx, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationPipelineReleases(ctx context.Context, req *pipelinePB.ListOrganizationPipelineReleasesRequest) (resp *pipelinePB.ListOrganizationPipelineReleasesResponse, err error) {
	resp = &pipelinePB.ListOrganizationPipelineReleasesResponse{}
	resp.Releases, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelineReleases(ctx, req)
	return resp, err
}

func (h *PublicHandler) listNamespacePipelineReleases(ctx context.Context, req ListNamespacePipelineReleasesRequestInterface) (releases []*pipelinePB.PipelineRelease, nextPageToken string, totalSize int32, err error) {

	eventName := "ListNamespacePipelineReleases"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineID, err := h.service.GetRscNamespaceAndNameID(ctx, req.GetParent())
	if err != nil {
		return nil, "", 0, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, "", 0, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
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
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return nil, "", 0, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListNamespacePipelineReleases(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbPipelineReleases, nextPageToken, totalSize, nil

}

type GetNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetView() pipelinePB.Pipeline_View
}

func (h *PublicHandler) GetUserPipelineRelease(ctx context.Context, req *pipelinePB.GetUserPipelineReleaseRequest) (resp *pipelinePB.GetUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.GetUserPipelineReleaseResponse{}
	resp.Release, err = h.getNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.GetOrganizationPipelineReleaseRequest) (resp *pipelinePB.GetOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.GetOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.getNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespacePipelineRelease(ctx context.Context, req GetNamespacePipelineReleaseRequestInterface) (release *pipelinePB.PipelineRelease, err error) {

	eventName := "GetNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineID, releaseID, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil

}

type UpdateNamespacePipelineReleaseRequestInterface interface {
	GetRelease() *pipelinePB.PipelineRelease
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserPipelineRelease(ctx context.Context, req *pipelinePB.UpdateUserPipelineReleaseRequest) (resp *pipelinePB.UpdateUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.UpdateUserPipelineReleaseResponse{}
	resp.Release, err = h.updateNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.UpdateOrganizationPipelineReleaseRequest) (resp *pipelinePB.UpdateOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.UpdateOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.updateNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespacePipelineRelease(ctx context.Context, req UpdateNamespacePipelineReleaseRequestInterface) (release *pipelinePB.PipelineRelease, err error) {

	eventName := "UpdateNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineID, releaseID, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(ctx, req.GetRelease().GetName())
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pbPipelineReleaseReq := req.GetRelease()
	pbUpdateMask := req.GetUpdateMask()

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReleaseReq) {
		return nil, ErrUpdateMask
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return nil, err
	}

	getResp, err := h.GetUserPipelineRelease(ctx, &pipelinePB.GetUserPipelineReleaseRequest{Name: pbPipelineReleaseReq.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, releaseOutputOnlyFields)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrFieldMask
	}

	if mask.IsEmpty() {
		return getResp.GetRelease(), nil
	}

	pbPipelineReleaseToUpdate := getResp.GetRelease()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReleaseReq, pbPipelineReleaseToUpdate, immutablePipelineFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReleaseReq, pbPipelineReleaseToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, pbPipelineReleaseToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil
}

type RenameNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetNewPipelineReleaseId() string
}

func (h *PublicHandler) RenameUserPipelineRelease(ctx context.Context, req *pipelinePB.RenameUserPipelineReleaseRequest) (resp *pipelinePB.RenameUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.RenameUserPipelineReleaseResponse{}
	resp.Release, err = h.renameNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) RenameOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.RenameOrganizationPipelineReleaseRequest) (resp *pipelinePB.RenameOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.RenameOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.renameNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) renameNamespacePipelineRelease(ctx context.Context, req RenameNamespacePipelineReleaseRequestInterface) (release *pipelinePB.PipelineRelease, err error) {

	eventName := "RenameNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, releaseRenameRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, ErrCheckRequiredFields
	}

	ns, pipelineID, releaseID, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return nil, err
	}

	newID := req.GetNewPipelineReleaseId()
	// Return error if resource ID does not a semantic version
	if !semver.IsValid(newID) {
		err := fmt.Errorf("not a sematic version")
		span.SetStatus(1, err.Error())
		return nil, ErrSematicVersion
	}

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseIDByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, newID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil
}

type DeleteNamespacePipelineReleaseRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserPipelineRelease(ctx context.Context, req *pipelinePB.DeleteUserPipelineReleaseRequest) (resp *pipelinePB.DeleteUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.DeleteUserPipelineReleaseResponse{}
	err = h.deleteNamespacePipelineRelease(ctx, req)
	return resp, err
}
func (h *PublicHandler) DeleteOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.DeleteOrganizationPipelineReleaseRequest) (resp *pipelinePB.DeleteOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.DeleteOrganizationPipelineReleaseResponse{}
	err = h.deleteNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) deleteNamespacePipelineRelease(ctx context.Context, req DeleteNamespacePipelineReleaseRequestInterface) error {

	eventName := "DeleteNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineID, releaseID, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(ctx, req.GetName())
	if err != nil {
		return err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return err
	}

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pipelinePB.GetUserPipelineReleaseRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return err
	}

	if err := h.service.DeleteNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return nil
}

type RestoreNamespacePipelineReleaseRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) RestoreUserPipelineRelease(ctx context.Context, req *pipelinePB.RestoreUserPipelineReleaseRequest) (resp *pipelinePB.RestoreUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.RestoreUserPipelineReleaseResponse{}
	resp.Release, err = h.restoreNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) RestoreOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.RestoreOrganizationPipelineReleaseRequest) (resp *pipelinePB.RestoreOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.RestoreOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.restoreNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) restoreNamespacePipelineRelease(ctx context.Context, req RestoreNamespacePipelineReleaseRequestInterface) (release *pipelinePB.PipelineRelease, err error) {

	eventName := "RestoreNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineID, releaseID, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(ctx, req.GetName())
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pipelinePB.GetUserPipelineReleaseRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewBasic)
	if err != nil {
		return nil, err
	}

	if err := h.service.RestoreNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, service.ViewFull)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return pbPipelineRelease, nil
}

func (h *PublicHandler) preTriggerUserPipelineRelease(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pipelinePB.Pipeline, *pipelinePB.PipelineRelease, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", nil, nil, false, ErrCheckRequiredFields
	}

	ns, pipelineID, releaseID, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(ctx, req.GetName())
	if err != nil {
		return ns, "", nil, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, service.ViewFull)
	if err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, service.ViewFull)
	if err != nil {
		return ns, "", nil, nil, false, err
	}
	returnTraces := false
	if resource.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true" {
		returnTraces = true
	}

	return ns, releaseID, pbPipeline, pbPipelineRelease, returnTraces, nil

}

type TriggerNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
}

func (h *PublicHandler) TriggerUserPipelineRelease(ctx context.Context, req *pipelinePB.TriggerUserPipelineReleaseRequest) (resp *pipelinePB.TriggerUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.TriggerUserPipelineReleaseResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.TriggerOrganizationPipelineReleaseRequest) (resp *pipelinePB.TriggerOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.TriggerOrganizationPipelineReleaseResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) triggerNamespacePipelineRelease(ctx context.Context, req TriggerNamespacePipelineReleaseRequestInterface) (outputs []*structpb.Struct, metadata *pipelinePB.TriggerMetadata, err error) {

	eventName := "TriggerNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, releaseID, pbPipeline, pbPipelineRelease, returnTraces, err := h.preTriggerUserPipelineRelease(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, nil, err
	}

	outputs, metadata, err = h.service.TriggerNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, req.GetInputs(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return outputs, metadata, nil
}

type TriggerAsyncNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
}

func (h *PublicHandler) TriggerAsyncUserPipelineRelease(ctx context.Context, req *pipelinePB.TriggerAsyncUserPipelineReleaseRequest) (resp *pipelinePB.TriggerAsyncUserPipelineReleaseResponse, err error) {
	resp = &pipelinePB.TriggerAsyncUserPipelineReleaseResponse{}
	resp.Operation, err = h.triggerAsyncNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerAsyncOrganizationPipelineRelease(ctx context.Context, req *pipelinePB.TriggerAsyncOrganizationPipelineReleaseRequest) (resp *pipelinePB.TriggerAsyncOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinePB.TriggerAsyncOrganizationPipelineReleaseResponse{}
	resp.Operation, err = h.triggerAsyncNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) triggerAsyncNamespacePipelineRelease(ctx context.Context, req TriggerAsyncNamespacePipelineReleaseRequestInterface) (operation *longrunningpb.Operation, err error) {

	eventName := "TriggerAsyncNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, releaseID, pbPipeline, pbPipelineRelease, returnTraces, err := h.preTriggerUserPipelineRelease(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	operation, err = h.service.TriggerAsyncNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, req.GetInputs(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return operation, nil
}
