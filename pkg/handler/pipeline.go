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
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

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
		return &pipelinePB.LookUpPipelineAdminResponse{}, status.Error(codes.InvalidArgument, err.Error())
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

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListPipelineReleasesAdmin(ctx, req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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

func (h *PublicHandler) ListPipelines(ctx context.Context, req *pipelinePB.ListPipelinesRequest) (*pipelinePB.ListPipelinesResponse, error) {

	eventName := "ListPipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

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
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(ctx, userUid, req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
	)))

	resp := pipelinePB.ListPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

func (h *PublicHandler) CreateUserPipeline(ctx context.Context, req *pipelinePB.CreateUserPipelineRequest) (*pipelinePB.CreateUserPipelineResponse, error) {

	eventName := "CreateUserPipeline"

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
	if err := checkfield.CheckRequiredFields(req.Pipeline, append(createPipelineRequiredFields, immutablePipelineFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Pipeline, outputOnlyPipelineFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.Pipeline.GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.Parent)

	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, err
	}

	_, userUid, err := h.service.GetCtxUser(ctx)

	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, err
	}

	pipelineToCreate := req.GetPipeline()

	name, err := h.service.ConvertOwnerPermalinkToName(fmt.Sprintf("users/%s", userUid))
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, err
	}

	pipelineToCreate.Owner = &pipelinePB.Pipeline_User{User: name}

	pipeline, err := h.service.CreateNamespacePipeline(ctx, ns, userUid, pipelineToCreate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, err
	}

	resp := pipelinePB.CreateUserPipelineResponse{
		Pipeline: pipeline,
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pipeline),
	)))

	return &resp, nil
}

func (h *PublicHandler) ListUserPipelines(ctx context.Context, req *pipelinePB.ListUserPipelinesRequest) (*pipelinePB.ListUserPipelinesResponse, error) {

	eventName := "ListUserPipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.Parent)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListUserPipelinesResponse{}, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListUserPipelinesResponse{}, err
	}

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
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListUserPipelinesResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListUserPipelinesResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListNamespacePipelines(ctx, ns, userUid, req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListUserPipelinesResponse{}, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
	)))

	resp := pipelinePB.ListUserPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

func (h *PublicHandler) GetUserPipeline(ctx context.Context, req *pipelinePB.GetUserPipelineRequest) (*pipelinePB.GetUserPipelineResponse, error) {

	eventName := "GetUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, id, parseView(int32(*req.GetView().Enum())))

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.GetUserPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

func (h *PublicHandler) UpdateUserPipeline(ctx context.Context, req *pipelinePB.UpdateUserPipelineRequest) (*pipelinePB.UpdateUserPipelineResponse, error) {

	eventName := "UpdateUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(req.Pipeline.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
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
	}
	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReq) {
		return nil, status.Error(codes.InvalidArgument, "The update_mask is invalid")
	}

	getResp, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: pbPipelineReq.GetName(), View: pipelinePB.Pipeline_VIEW_RECIPE.Enum()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyPipelineFields)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if mask.IsEmpty() {
		return &pipelinePB.UpdateUserPipelineResponse{
			Pipeline: getResp.GetPipeline(),
		}, nil
	}

	pbPipelineToUpdate := getResp.GetPipeline()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReq, pbPipelineToUpdate, immutablePipelineFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReq, pbPipelineToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.UpdateNamespacePipelineByID(ctx, ns, userUid, id, pbPipelineToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.UpdateUserPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

func (h *PublicHandler) DeleteUserPipeline(ctx context.Context, req *pipelinePB.DeleteUserPipelineRequest) (*pipelinePB.DeleteUserPipelineResponse, error) {

	eventName := "DeleteUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	existPipeline, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineByID(ctx, ns, userUid, id); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(existPipeline.GetPipeline()),
	)))

	return &pipelinePB.DeleteUserPipelineResponse{}, nil
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	uid, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.GetPipelineByUID(ctx, userUid, uid, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.LookUpPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

func (h *PublicHandler) ValidateUserPipeline(ctx context.Context, req *pipelinePB.ValidateUserPipelineRequest) (*pipelinePB.ValidateUserPipelineResponse, error) {

	eventName := "ValidateUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.ValidateNamespacePipelineByID(ctx, ns, userUid, id)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	resp := pipelinePB.ValidateUserPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

func (h *PublicHandler) RenameUserPipeline(ctx context.Context, req *pipelinePB.RenameUserPipelineRequest) (*pipelinePB.RenameUserPipelineResponse, error) {

	eventName := "RenameUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, renamePipelineRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, id, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	newID := req.GetNewPipelineId()
	if err := checkfield.CheckResourceID(newID); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pbPipeline, err := h.service.UpdateNamespacePipelineIDByID(ctx, ns, userUid, id, newID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.RenameUserPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

func (h *PublicHandler) preTriggerUserPipeline(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, uuid.UUID, string, *pipelinePB.Pipeline, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, uuid.Nil, "", nil, false, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, id, err := h.service.GetRscNamespaceAndNameID(req.GetName())
	if err != nil {
		return ns, uuid.Nil, id, nil, false, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return ns, uuid.Nil, id, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, id, service.VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, id, nil, false, err
	}
	_, err = h.service.ValidateNamespacePipelineByID(ctx, ns, userUid, id)
	if err != nil {
		return ns, uuid.Nil, id, nil, false, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}
	returnTraces := false
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md.Get(constant.ReturnTracesKey)) > 0 {
			returnTraces, err = strconv.ParseBool(md.Get(constant.ReturnTracesKey)[0])
			if err != nil {
				return ns, uuid.Nil, id, nil, false, err
			}
		}
	}

	return ns, userUid, id, pbPipeline, returnTraces, nil

}

func (h *PublicHandler) TriggerUserPipeline(ctx context.Context, req *pipelinePB.TriggerUserPipelineRequest) (*pipelinePB.TriggerUserPipelineResponse, error) {

	startTime := time.Now()
	eventName := "TriggerUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, userUid, id, pbPipeline, returnTraces, err := h.preTriggerUserPipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           userUid.String(),
		TriggerMode:        mgmtPB.Mode_MODE_SYNC,
		PipelineID:         pbPipeline.Id,
		PipelineUID:        pbPipeline.Uid,
		PipelineReleaseID:  "",
		PipelineReleaseUID: uuid.Nil.String(),
		PipelineTriggerUID: logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
	}

	outputs, metadata, err := h.service.TriggerNamespacePipelineByID(ctx, ns, userUid, id, req.Inputs, logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewPipelineDataPoint(ctx, dataPoint)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipeline),
	)))

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED
	if err := h.service.WriteNewPipelineDataPoint(ctx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}

	return &pipelinePB.TriggerUserPipelineResponse{Outputs: outputs, Metadata: metadata}, nil
}

func (h *PublicHandler) TriggerAsyncUserPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncUserPipelineRequest) (*pipelinePB.TriggerAsyncUserPipelineResponse, error) {

	eventName := "TriggerAsyncUserPipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, userUid, id, dbPipeline, returnTraces, err := h.preTriggerUserPipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	operation, err := h.service.TriggerAsyncNamespacePipelineByID(ctx, ns, userUid, id, req.Inputs, logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(dbPipeline),
	)))

	return &pipelinePB.TriggerAsyncUserPipelineResponse{
		Operation: operation,
	}, nil
}

func (h *PublicHandler) CreateUserPipelineRelease(ctx context.Context, req *pipelinePB.CreateUserPipelineReleaseRequest) (*pipelinePB.CreateUserPipelineReleaseResponse, error) {
	eventName := "CreateUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.Release, append(releaseCreateRequiredFields, immutablePipelineFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Release, releaseOutputOnlyFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Return error if resource ID does not a semantic version
	if !semver.IsValid(req.Release.GetId()) {
		err := fmt.Errorf("not a sematic version")
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, pipelineId, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	_, err = h.service.ValidateNamespacePipelineByID(ctx, ns, userUid, pipeline.Id)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	pbPipelineRelease, err := h.service.CreateNamespacePipelineRelease(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), req.GetRelease())
	if err != nil {
		span.SetStatus(1, err.Error())
		// Manually set the custom header to have a StatusBadRequest http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusBadRequest))); err != nil {
			return nil, err
		}
		return nil, err
	}

	resp := pipelinePB.CreateUserPipelineReleaseResponse{
		Release: pbPipelineRelease,
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return &resp, nil

}

func (h *PublicHandler) ListUserPipelineReleases(ctx context.Context, req *pipelinePB.ListUserPipelineReleasesRequest) (*pipelinePB.ListUserPipelineReleasesResponse, error) {

	eventName := "ListUserPipelineReleases"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, err := h.service.GetRscNamespaceAndNameID(req.GetParent())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}

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
		span.SetStatus(1, err.Error())
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListNamespacePipelineReleases(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), req.GetPageSize(), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
	)))

	resp := pipelinePB.ListUserPipelineReleasesResponse{
		Releases:      pbPipelineReleases,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil

}

func (h *PublicHandler) GetUserPipelineRelease(ctx context.Context, req *pipelinePB.GetUserPipelineReleaseRequest) (*pipelinePB.GetUserPipelineReleaseResponse, error) {

	eventName := "GetUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.GetUserPipelineReleaseResponse{
		Release: pbPipelineRelease,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return &resp, nil

}

func (h *PublicHandler) UpdateUserPipelineRelease(ctx context.Context, req *pipelinePB.UpdateUserPipelineReleaseRequest) (*pipelinePB.UpdateUserPipelineReleaseResponse, error) {

	eventName := "UpdateUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.Release.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pbPipelineReleaseReq := req.GetRelease()
	pbUpdateMask := req.GetUpdateMask()

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReleaseReq) {
		return nil, status.Error(codes.InvalidArgument, "The update_mask is invalid")
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
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
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if mask.IsEmpty() {
		return &pipelinePB.UpdateUserPipelineReleaseResponse{
			Release: getResp.GetRelease(),
		}, nil
	}

	pbPipelineReleaseToUpdate := getResp.GetRelease()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReleaseReq, pbPipelineReleaseToUpdate, immutablePipelineFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReleaseReq, pbPipelineReleaseToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, pbPipelineReleaseToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.UpdateUserPipelineReleaseResponse{
		Release: pbPipelineRelease,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return &resp, nil
}

func (h *PublicHandler) RenameUserPipelineRelease(ctx context.Context, req *pipelinePB.RenameUserPipelineReleaseRequest) (*pipelinePB.RenameUserPipelineReleaseResponse, error) {

	eventName := "RenameUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, releaseRenameRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	newID := req.GetNewPipelineReleaseId()
	// Return error if resource ID does not a semantic version
	if !semver.IsValid(newID) {
		err := fmt.Errorf("not a sematic version")
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseIDByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, newID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pipelinePB.RenameUserPipelineReleaseResponse{
		Release: pbPipelineRelease,
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return &resp, nil
}

func (h *PublicHandler) DeleteUserPipelineRelease(ctx context.Context, req *pipelinePB.DeleteUserPipelineReleaseRequest) (*pipelinePB.DeleteUserPipelineReleaseResponse, error) {

	eventName := "DeleteUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pipelinePB.GetUserPipelineReleaseRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return &pipelinePB.DeleteUserPipelineReleaseResponse{}, nil
}

func (h *PublicHandler) SetDefaultUserPipelineRelease(ctx context.Context, req *pipelinePB.SetDefaultUserPipelineReleaseRequest) (*pipelinePB.SetDefaultUserPipelineReleaseResponse, error) {

	eventName := "SetDefaultUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pipelinePB.GetUserPipelineReleaseRequest{Name: req.GetName()})

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.SetDefaultNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, service.VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return &pipelinePB.SetDefaultUserPipelineReleaseResponse{Release: pbPipelineRelease}, nil
}

func (h *PublicHandler) RestoreUserPipelineRelease(ctx context.Context, req *pipelinePB.RestoreUserPipelineReleaseRequest) (*pipelinePB.RestoreUserPipelineReleaseResponse, error) {

	eventName := "RestoreUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pipelinePB.GetUserPipelineReleaseRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.RestoreNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, service.VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return &pipelinePB.RestoreUserPipelineReleaseResponse{Release: pbPipelineRelease}, nil
}

func (h *PublicHandler) preTriggerUserPipelineRelease(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, uuid.UUID, string, *pipelinePB.Pipeline, *pipelinePB.PipelineRelease, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, uuid.Nil, "", nil, nil, false, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, service.VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}
	returnTraces := false
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if len(md.Get(constant.ReturnTracesKey)) > 0 {
			returnTraces, err = strconv.ParseBool(md.Get(constant.ReturnTracesKey)[0])
			if err != nil {
				return ns, uuid.Nil, "", nil, nil, false, err
			}
		}
	}

	return ns, userUid, releaseId, pbPipeline, pbPipelineRelease, returnTraces, nil

}

func (h *PublicHandler) TriggerUserPipelineRelease(ctx context.Context, req *pipelinePB.TriggerUserPipelineReleaseRequest) (*pipelinePB.TriggerUserPipelineReleaseResponse, error) {

	startTime := time.Now()
	eventName := "TriggerUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, userUid, releaseId, pbPipeline, pbPipelineRelease, returnTraces, err := h.preTriggerUserPipelineRelease(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	dataPoint := utils.PipelineUsageMetricData{
		OwnerUID:           userUid.String(),
		TriggerMode:        mgmtPB.Mode_MODE_SYNC,
		PipelineID:         pbPipeline.Id,
		PipelineUID:        pbPipeline.Uid,
		PipelineReleaseID:  pbPipelineRelease.Id,
		PipelineReleaseUID: pbPipelineRelease.Uid,
		PipelineTriggerUID: logUUID.String(),
		TriggerTime:        startTime.Format(time.RFC3339Nano),
	}

	outputs, metadata, err := h.service.TriggerNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, req.Inputs, logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
		dataPoint.Status = mgmtPB.Status_STATUS_ERRORED
		_ = h.service.WriteNewPipelineDataPoint(ctx, dataPoint)
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	dataPoint.ComputeTimeDuration = time.Since(startTime).Seconds()
	dataPoint.Status = mgmtPB.Status_STATUS_COMPLETED
	if err := h.service.WriteNewPipelineDataPoint(ctx, dataPoint); err != nil {
		logger.Warn(err.Error())
	}

	return &pipelinePB.TriggerUserPipelineReleaseResponse{Outputs: outputs, Metadata: metadata}, nil
}

func (h *PublicHandler) TriggerAsyncUserPipelineRelease(ctx context.Context, req *pipelinePB.TriggerAsyncUserPipelineReleaseRequest) (*pipelinePB.TriggerAsyncUserPipelineReleaseResponse, error) {

	eventName := "TriggerAsyncUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, userUid, releaseId, pbPipeline, pbPipelineRelease, returnTraces, err := h.preTriggerUserPipelineRelease(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	operation, err := h.service.TriggerAsyncNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, req.Inputs, logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(pbPipelineRelease),
	)))

	return &pipelinePB.TriggerAsyncUserPipelineReleaseResponse{Operation: operation}, nil
}

func (h *PublicHandler) WatchUserPipelineRelease(ctx context.Context, req *pipelinePB.WatchUserPipelineReleaseRequest) (*pipelinePB.WatchUserPipelineReleaseResponse, error) {

	eventName := "WatchUserPipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, pipelineId, releaseId, err := h.service.GetRscNamespaceAndNameIDAndReleaseID(req.GetName())
	if err != nil {
		return nil, err
	}
	_, userUid, err := h.service.GetCtxUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(req.GetName()),
		)))
		return nil, err
	}

	dbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, service.VIEW_BASIC)
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(req.GetName()),
		)))
		return nil, err
	}
	state, err := h.service.GetPipelineState(uuid.FromStringOrNil(dbPipelineRelease.Uid))
	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(req.GetName()),
		)))
		return nil, err
	}

	return &pipelinePB.WatchUserPipelineReleaseResponse{
		State: *state,
	}, nil
}
