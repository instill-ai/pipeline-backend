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
	"go.einride.tech/aip/ordering"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/checkfield"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func (h *PrivateHandler) ListPipelinesAdmin(ctx context.Context, req *pb.ListPipelinesAdminRequest) (*pb.ListPipelinesAdminResponse, error) {

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
		return &pb.ListPipelinesAdminResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pb.ListPipelinesAdminResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelinesAdmin(ctx, req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		return &pb.ListPipelinesAdminResponse{}, err
	}

	resp := pb.ListPipelinesAdminResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

func (h *PrivateHandler) LookUpPipelineAdmin(ctx context.Context, req *pb.LookUpPipelineAdminRequest) (*pb.LookUpPipelineAdminResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpPipelineRequiredFields); err != nil {
		return &pb.LookUpPipelineAdminResponse{}, ErrCheckRequiredFields
	}

	view := pb.Pipeline_VIEW_BASIC
	if req.GetView() != pb.Pipeline_VIEW_UNSPECIFIED {
		view = req.GetView()
	}

	uid, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return &pb.LookUpPipelineAdminResponse{}, err
	}
	pbPipeline, err := h.service.GetPipelineByUIDAdmin(ctx, uid, view)
	if err != nil {
		return &pb.LookUpPipelineAdminResponse{}, err
	}

	resp := pb.LookUpPipelineAdminResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *PublicHandler) ListPipelines(ctx context.Context, req *pb.ListPipelinesRequest) (*pb.ListPipelinesResponse, error) {

	eventName := "ListPipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return &pb.ListPipelinesResponse{}, err
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
		return &pb.ListPipelinesResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pb.ListPipelinesResponse{}, err
	}

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pb.ListPipelinesResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(
		ctx, req.GetPageSize(), req.GetPageToken(), req.GetView(), req.Visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pb.ListPipelinesResponse{}, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	resp := pb.ListPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

type CreateNamespacePipelineRequestInterface interface {
	GetPipeline() *pb.Pipeline
	GetParent() string
}

func (h *PublicHandler) CreateUserPipeline(ctx context.Context, req *pb.CreateUserPipelineRequest) (resp *pb.CreateUserPipelineResponse, err error) {
	resp = &pb.CreateUserPipelineResponse{}
	resp.Pipeline, err = h.createNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationPipeline(ctx context.Context, req *pb.CreateOrganizationPipelineRequest) (resp *pb.CreateOrganizationPipelineResponse, err error) {
	resp = &pb.CreateOrganizationPipelineResponse{}
	resp.Pipeline, err = h.createNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespacePipeline(ctx context.Context, req CreateNamespacePipelineRequestInterface) (pipeline *pb.Pipeline, err error) {

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
		return nil, fmt.Errorf("%w: invalid secret ID: %w", errdomain.ErrInvalidArgument, err)
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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pipeline),
	)))

	return pipeline, nil
}

type ListNamespacePipelinesRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() pb.Pipeline_View
	GetVisibility() pb.Pipeline_Visibility
	GetFilter() string
	GetParent() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserPipelines(ctx context.Context, req *pb.ListUserPipelinesRequest) (resp *pb.ListUserPipelinesResponse, err error) {
	resp = &pb.ListUserPipelinesResponse{}
	resp.Pipelines, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelines(ctx, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationPipelines(ctx context.Context, req *pb.ListOrganizationPipelinesRequest) (resp *pb.ListOrganizationPipelinesResponse, err error) {
	resp = &pb.ListOrganizationPipelinesResponse{}
	resp.Pipelines, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelines(ctx, req)
	return resp, err
}

func (h *PublicHandler) listNamespacePipelines(ctx context.Context, req ListNamespacePipelinesRequestInterface) (pipelines []*pb.Pipeline, nextPageToken string, totalSize int32, err error) {

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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListNamespacePipelines(ctx, ns, req.GetPageSize(), req.GetPageToken(), req.GetView(), &visibility, filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbPipelines, nextPageToken, int32(totalSize), nil
}

type GetNamespacePipelineRequestInterface interface {
	GetName() string
	GetView() pb.Pipeline_View
}

func (h *PublicHandler) GetUserPipeline(ctx context.Context, req *pb.GetUserPipelineRequest) (resp *pb.GetUserPipelineResponse, err error) {
	resp = &pb.GetUserPipelineResponse{}
	resp.Pipeline, err = h.getNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationPipeline(ctx context.Context, req *pb.GetOrganizationPipelineRequest) (resp *pb.GetOrganizationPipelineResponse, err error) {
	resp = &pb.GetOrganizationPipelineResponse{}
	resp.Pipeline, err = h.getNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespacePipeline(ctx context.Context, req GetNamespacePipelineRequestInterface) (*pb.Pipeline, error) {

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

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, id, req.GetView())

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipeline),
	)))

	return pbPipeline, nil
}

type UpdateNamespacePipelineRequestInterface interface {
	GetPipeline() *pb.Pipeline
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserPipeline(ctx context.Context, req *pb.UpdateUserPipelineRequest) (resp *pb.UpdateUserPipelineResponse, err error) {
	resp = &pb.UpdateUserPipelineResponse{}
	resp.Pipeline, err = h.updateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationPipeline(ctx context.Context, req *pb.UpdateOrganizationPipelineRequest) (resp *pb.UpdateOrganizationPipelineResponse, err error) {
	resp = &pb.UpdateOrganizationPipelineResponse{}
	resp.Pipeline, err = h.updateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespacePipeline(ctx context.Context, req UpdateNamespacePipelineRequestInterface) (*pb.Pipeline, error) {

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

	getResp, err := h.GetUserPipeline(ctx, &pb.GetUserPipelineRequest{Name: pbPipelineReq.GetName(), View: pb.Pipeline_VIEW_RECIPE.Enum()})
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

	// logger.Info(string(customotel.NewLogMessage(
	// 	span,
	// 	logUUID.String(),
	// 	authUser.UID,
	// 	eventName,
	// 	customotel.SetEventResource(pbPipeline),
	// )))

	return pbPipeline, nil
}

type DeleteNamespacePipelineRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserPipeline(ctx context.Context, req *pb.DeleteUserPipelineRequest) (resp *pb.DeleteUserPipelineResponse, err error) {
	resp = &pb.DeleteUserPipelineResponse{}
	err = h.deleteNamespacePipeline(ctx, req)
	return resp, err
}
func (h *PublicHandler) DeleteOrganizationPipeline(ctx context.Context, req *pb.DeleteOrganizationPipelineRequest) (resp *pb.DeleteOrganizationPipelineResponse, err error) {
	resp = &pb.DeleteOrganizationPipelineResponse{}
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
	existPipeline, err := h.GetUserPipeline(ctx, &pb.GetUserPipelineRequest{Name: req.GetName()})
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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(existPipeline.GetPipeline()),
	)))

	return nil
}

func (h *PublicHandler) LookUpPipeline(ctx context.Context, req *pb.LookUpPipelineRequest) (*pb.LookUpPipelineResponse, error) {

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

	pbPipeline, err := h.service.GetPipelineByUID(ctx, uid, req.GetView())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	resp := pb.LookUpPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipeline),
	)))

	return &resp, nil
}

type ValidateNamespacePipelineRequest interface {
	GetName() string
}

func (h *PublicHandler) ValidateUserPipeline(ctx context.Context, req *pb.ValidateUserPipelineRequest) (resp *pb.ValidateUserPipelineResponse, err error) {
	resp = &pb.ValidateUserPipelineResponse{}
	resp.Pipeline, err = h.validateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) ValidateOrganizationPipeline(ctx context.Context, req *pb.ValidateOrganizationPipelineRequest) (resp *pb.ValidateOrganizationPipelineResponse, err error) {
	resp = &pb.ValidateOrganizationPipelineResponse{}
	resp.Pipeline, err = h.validateNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) validateNamespacePipeline(ctx context.Context, req ValidateNamespacePipelineRequest) (*pb.Pipeline, error) {

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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipeline),
	)))

	return pbPipeline, nil
}

type RenameNamespacePipelineRequestInterface interface {
	GetName() string
	GetNewPipelineId() string
}

func (h *PublicHandler) RenameUserPipeline(ctx context.Context, req *pb.RenameUserPipelineRequest) (resp *pb.RenameUserPipelineResponse, err error) {
	resp = &pb.RenameUserPipelineResponse{}
	resp.Pipeline, err = h.renameNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) RenameOrganizationPipeline(ctx context.Context, req *pb.RenameOrganizationPipelineRequest) (resp *pb.RenameOrganizationPipelineResponse, err error) {
	resp = &pb.RenameOrganizationPipelineResponse{}
	resp.Pipeline, err = h.renameNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) renameNamespacePipeline(ctx context.Context, req RenameNamespacePipelineRequestInterface) (*pb.Pipeline, error) {

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
		return nil, fmt.Errorf("%w: invalid pipeline ID: %w", errdomain.ErrInvalidArgument, err)
	}

	pbPipeline, err := h.service.UpdateNamespacePipelineIDByID(ctx, ns, id, newID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipeline),
	)))

	return pbPipeline, nil
}

type CloneNamespacePipelineRequestInterface interface {
	GetName() string
	GetTarget() string
}

func (h *PublicHandler) CloneUserPipeline(ctx context.Context, req *pb.CloneUserPipelineRequest) (resp *pb.CloneUserPipelineResponse, err error) {
	resp = &pb.CloneUserPipelineResponse{}
	resp.Pipeline, err = h.cloneNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) CloneOrganizationPipeline(ctx context.Context, req *pb.CloneOrganizationPipelineRequest) (resp *pb.CloneOrganizationPipelineResponse, err error) {
	resp = &pb.CloneOrganizationPipelineResponse{}
	resp.Pipeline, err = h.cloneNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) cloneNamespacePipeline(ctx context.Context, req CloneNamespacePipelineRequestInterface) (*pb.Pipeline, error) {

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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipeline),
	)))
	return pbPipeline, nil
}

func (h *PublicHandler) preTriggerUserPipeline(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pb.Pipeline, bool, error) {

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

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, id, pb.Pipeline_VIEW_FULL)
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
	GetSecrets() map[string]string
}

func (h *PublicHandler) TriggerUserPipeline(ctx context.Context, req *pb.TriggerUserPipelineRequest) (resp *pb.TriggerUserPipelineResponse, err error) {
	resp = &pb.TriggerUserPipelineResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerOrganizationPipeline(ctx context.Context, req *pb.TriggerOrganizationPipelineRequest) (resp *pb.TriggerOrganizationPipelineResponse, err error) {
	resp = &pb.TriggerOrganizationPipelineResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) triggerNamespacePipeline(ctx context.Context, req TriggerNamespacePipelineRequestInterface) (outputs []*structpb.Struct, metadata *pb.TriggerMetadata, err error) {

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

	outputs, metadata, err = h.service.TriggerNamespacePipelineByID(ctx, ns, id, req.GetInputs(), req.GetSecrets(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, nil, err
	}

	// logger.Info(string(customotel.NewLogMessage(
	// 	span,
	// 	logUUID.String(),
	// 	authUser.UID,
	// 	eventName,
	// 	customotel.SetEventResource(pbPipeline),
	// )))

	return outputs, metadata, nil
}

type TriggerAsyncNamespacePipelineRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
	GetSecrets() map[string]string
}

func (h *PublicHandler) TriggerAsyncUserPipeline(ctx context.Context, req *pb.TriggerAsyncUserPipelineRequest) (resp *pb.TriggerAsyncUserPipelineResponse, err error) {
	resp = &pb.TriggerAsyncUserPipelineResponse{}
	resp.Operation, err = h.triggerAsyncNamespacePipeline(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerAsyncOrganizationPipeline(ctx context.Context, req *pb.TriggerAsyncOrganizationPipelineRequest) (resp *pb.TriggerAsyncOrganizationPipelineResponse, err error) {
	resp = &pb.TriggerAsyncOrganizationPipelineResponse{}
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

	operation, err = h.service.TriggerAsyncNamespacePipelineByID(ctx, ns, id, req.GetInputs(), req.GetSecrets(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(dbPipeline),
	)))

	return operation, nil
}

type CreateNamespacePipelineReleaseRequestInterface interface {
	GetRelease() *pb.PipelineRelease
	GetParent() string
}

func (h *PublicHandler) CreateUserPipelineRelease(ctx context.Context, req *pb.CreateUserPipelineReleaseRequest) (resp *pb.CreateUserPipelineReleaseResponse, err error) {
	resp = &pb.CreateUserPipelineReleaseResponse{}
	resp.Release, err = h.createNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) CreateOrganizationPipelineRelease(ctx context.Context, req *pb.CreateOrganizationPipelineReleaseRequest) (resp *pb.CreateOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.CreateOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.createNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) createNamespacePipelineRelease(ctx context.Context, req CreateNamespacePipelineReleaseRequestInterface) (*pb.PipelineRelease, error) {
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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil

}

type ListNamespacePipelineReleasesRequestInterface interface {
	GetPageSize() int32
	GetPageToken() string
	GetView() pb.Pipeline_View
	GetFilter() string
	GetParent() string
	GetShowDeleted() bool
}

func (h *PublicHandler) ListUserPipelineReleases(ctx context.Context, req *pb.ListUserPipelineReleasesRequest) (resp *pb.ListUserPipelineReleasesResponse, err error) {
	resp = &pb.ListUserPipelineReleasesResponse{}
	resp.Releases, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelineReleases(ctx, req)
	return resp, err
}

func (h *PublicHandler) ListOrganizationPipelineReleases(ctx context.Context, req *pb.ListOrganizationPipelineReleasesRequest) (resp *pb.ListOrganizationPipelineReleasesResponse, err error) {
	resp = &pb.ListOrganizationPipelineReleasesResponse{}
	resp.Releases, resp.NextPageToken, resp.TotalSize, err = h.listNamespacePipelineReleases(ctx, req)
	return resp, err
}

func (h *PublicHandler) listNamespacePipelineReleases(ctx context.Context, req ListNamespacePipelineReleasesRequestInterface) (releases []*pb.PipelineRelease, nextPageToken string, totalSize int32, err error) {

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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, "", 0, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListNamespacePipelineReleases(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, "", 0, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return pbPipelineReleases, nextPageToken, totalSize, nil

}

type GetNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetView() pb.Pipeline_View
}

func (h *PublicHandler) GetUserPipelineRelease(ctx context.Context, req *pb.GetUserPipelineReleaseRequest) (resp *pb.GetUserPipelineReleaseResponse, err error) {
	resp = &pb.GetUserPipelineReleaseResponse{}
	resp.Release, err = h.getNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) GetOrganizationPipelineRelease(ctx context.Context, req *pb.GetOrganizationPipelineReleaseRequest) (resp *pb.GetOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.GetOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.getNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) getNamespacePipelineRelease(ctx context.Context, req GetNamespacePipelineReleaseRequestInterface) (release *pb.PipelineRelease, err error) {

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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, req.GetView())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil

}

type UpdateNamespacePipelineReleaseRequestInterface interface {
	GetRelease() *pb.PipelineRelease
	GetUpdateMask() *fieldmaskpb.FieldMask
}

func (h *PublicHandler) UpdateUserPipelineRelease(ctx context.Context, req *pb.UpdateUserPipelineReleaseRequest) (resp *pb.UpdateUserPipelineReleaseResponse, err error) {
	resp = &pb.UpdateUserPipelineReleaseResponse{}
	resp.Release, err = h.updateNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) UpdateOrganizationPipelineRelease(ctx context.Context, req *pb.UpdateOrganizationPipelineReleaseRequest) (resp *pb.UpdateOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.UpdateOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.updateNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) updateNamespacePipelineRelease(ctx context.Context, req UpdateNamespacePipelineReleaseRequestInterface) (release *pb.PipelineRelease, err error) {

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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	getResp, err := h.GetUserPipelineRelease(ctx, &pb.GetUserPipelineReleaseRequest{Name: pbPipelineReleaseReq.GetName()})
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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil
}

type RenameNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetNewPipelineReleaseId() string
}

func (h *PublicHandler) RenameUserPipelineRelease(ctx context.Context, req *pb.RenameUserPipelineReleaseRequest) (resp *pb.RenameUserPipelineReleaseResponse, err error) {
	resp = &pb.RenameUserPipelineReleaseResponse{}
	resp.Release, err = h.renameNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) RenameOrganizationPipelineRelease(ctx context.Context, req *pb.RenameOrganizationPipelineReleaseRequest) (resp *pb.RenameOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.RenameOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.renameNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) renameNamespacePipelineRelease(ctx context.Context, req RenameNamespacePipelineReleaseRequestInterface) (release *pb.PipelineRelease, err error) {

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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipelineRelease),
	)))

	return pbPipelineRelease, nil
}

type DeleteNamespacePipelineReleaseRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) DeleteUserPipelineRelease(ctx context.Context, req *pb.DeleteUserPipelineReleaseRequest) (resp *pb.DeleteUserPipelineReleaseResponse, err error) {
	resp = &pb.DeleteUserPipelineReleaseResponse{}
	err = h.deleteNamespacePipelineRelease(ctx, req)
	return resp, err
}
func (h *PublicHandler) DeleteOrganizationPipelineRelease(ctx context.Context, req *pb.DeleteOrganizationPipelineReleaseRequest) (resp *pb.DeleteOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.DeleteOrganizationPipelineReleaseResponse{}
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

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pb.GetUserPipelineReleaseRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
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

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return nil
}

type RestoreNamespacePipelineReleaseRequestInterface interface {
	GetName() string
}

func (h *PublicHandler) RestoreUserPipelineRelease(ctx context.Context, req *pb.RestoreUserPipelineReleaseRequest) (resp *pb.RestoreUserPipelineReleaseResponse, err error) {
	resp = &pb.RestoreUserPipelineReleaseResponse{}
	resp.Release, err = h.restoreNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) RestoreOrganizationPipelineRelease(ctx context.Context, req *pb.RestoreOrganizationPipelineReleaseRequest) (resp *pb.RestoreOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.RestoreOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.restoreNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) restoreNamespacePipelineRelease(ctx context.Context, req RestoreNamespacePipelineReleaseRequestInterface) (release *pb.PipelineRelease, err error) {

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

	existPipelineRelease, err := h.GetUserPipelineRelease(ctx, &pb.GetUserPipelineReleaseRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.RestoreNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, pb.Pipeline_VIEW_FULL)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(existPipelineRelease.GetRelease()),
	)))

	return pbPipelineRelease, nil
}

func (h *PublicHandler) preTriggerUserPipelineRelease(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pb.Pipeline, *pb.PipelineRelease, bool, error) {

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

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, pb.Pipeline_VIEW_FULL)
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
	GetSecrets() map[string]string
}

func (h *PublicHandler) TriggerUserPipelineRelease(ctx context.Context, req *pb.TriggerUserPipelineReleaseRequest) (resp *pb.TriggerUserPipelineReleaseResponse, err error) {
	resp = &pb.TriggerUserPipelineReleaseResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerOrganizationPipelineRelease(ctx context.Context, req *pb.TriggerOrganizationPipelineReleaseRequest) (resp *pb.TriggerOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.TriggerOrganizationPipelineReleaseResponse{}
	resp.Outputs, resp.Metadata, err = h.triggerNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) triggerNamespacePipelineRelease(ctx context.Context, req TriggerNamespacePipelineReleaseRequestInterface) (outputs []*structpb.Struct, metadata *pb.TriggerMetadata, err error) {

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

	outputs, metadata, err = h.service.TriggerNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, req.GetInputs(), req.GetSecrets(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipelineRelease),
	)))

	return outputs, metadata, nil
}

type TriggerAsyncNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetInputs() []*structpb.Struct
	GetSecrets() map[string]string
}

func (h *PublicHandler) TriggerAsyncUserPipelineRelease(ctx context.Context, req *pb.TriggerAsyncUserPipelineReleaseRequest) (resp *pb.TriggerAsyncUserPipelineReleaseResponse, err error) {
	resp = &pb.TriggerAsyncUserPipelineReleaseResponse{}
	resp.Operation, err = h.triggerAsyncNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) TriggerAsyncOrganizationPipelineRelease(ctx context.Context, req *pb.TriggerAsyncOrganizationPipelineReleaseRequest) (resp *pb.TriggerAsyncOrganizationPipelineReleaseResponse, err error) {
	resp = &pb.TriggerAsyncOrganizationPipelineReleaseResponse{}
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

	operation, err = h.service.TriggerAsyncNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, req.GetInputs(), req.GetSecrets(), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(pbPipelineRelease),
	)))

	return operation, nil
}

func (h *PublicHandler) GetOperation(ctx context.Context, req *pb.GetOperationRequest) (*pb.GetOperationResponse, error) {

	operationID, err := resource.GetOperationID(req.Name)
	if err != nil {
		return &pb.GetOperationResponse{}, err
	}
	operation, err := h.service.GetOperation(ctx, operationID)
	if err != nil {
		return &pb.GetOperationResponse{}, err
	}

	return &pb.GetOperationResponse{
		Operation: operation,
	}, nil
}
