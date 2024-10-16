package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/service"
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
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
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

func (h *PublicHandler) GetHubStats(ctx context.Context, req *pb.GetHubStatsRequest) (*pb.GetHubStatsResponse, error) {

	eventName := "GetHubStats"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return &pb.GetHubStatsResponse{}, err
	}

	resp, err := h.service.GetHubStats(ctx)

	if err != nil {
		span.SetStatus(1, err.Error())
		return &pb.GetHubStatsResponse{}, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return resp, nil
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
		// Currently, we only have a "featured" tag, so we'll only support single tag filter for now.
		filtering.DeclareIdent("tag", filtering.TypeString),
		// only support "recipe.components.resource_name" for now
		filtering.DeclareIdent("recipe", filtering.TypeMap(filtering.TypeString, filtering.TypeMap(filtering.TypeString, filtering.TypeString))),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("numberOfRuns", filtering.TypeInt),
		filtering.DeclareIdent("numberOfClones", filtering.TypeInt),
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
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

func (h *PublicHandler) CreateUserPipeline(ctx context.Context, req *pb.CreateUserPipelineRequest) (resp *pb.CreateUserPipelineResponse, err error) {
	r, err := h.CreateNamespacePipeline(ctx, &pb.CreateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Pipeline:    req.Pipeline,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateUserPipelineResponse{Pipeline: r.Pipeline}, nil
}

func (h *PublicHandler) CreateOrganizationPipeline(ctx context.Context, req *pb.CreateOrganizationPipelineRequest) (resp *pb.CreateOrganizationPipelineResponse, err error) {
	r, err := h.CreateNamespacePipeline(ctx, &pb.CreateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Pipeline:    req.Pipeline,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateOrganizationPipelineResponse{Pipeline: r.Pipeline}, nil
}

func (h *PublicHandler) CreateNamespacePipeline(ctx context.Context, req *pb.CreateNamespacePipelineRequest) (resp *pb.CreateNamespacePipelineResponse, err error) {
	eventName := "CreateNamespacePipeline"
	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

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

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)

	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipelineToCreate := req.GetPipeline()

	pipeline, err := h.service.CreateNamespacePipeline(ctx, ns, pipelineToCreate)

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

	return &pb.CreateNamespacePipelineResponse{Pipeline: pipeline}, nil
}

func (h *PublicHandler) ListUserPipelines(ctx context.Context, req *pb.ListUserPipelinesRequest) (resp *pb.ListUserPipelinesResponse, err error) {
	r, err := h.ListNamespacePipelines(ctx, &pb.ListNamespacePipelinesRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
		View:        req.View,
		Visibility:  req.Visibility,
		Filter:      req.Filter,
		OrderBy:     req.OrderBy,
		ShowDeleted: req.ShowDeleted,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListUserPipelinesResponse{
		Pipelines:     r.Pipelines,
		NextPageToken: r.NextPageToken,
		TotalSize:     r.TotalSize,
	}, nil
}

func (h *PublicHandler) ListOrganizationPipelines(ctx context.Context, req *pb.ListOrganizationPipelinesRequest) (resp *pb.ListOrganizationPipelinesResponse, err error) {
	r, err := h.ListNamespacePipelines(ctx, &pb.ListNamespacePipelinesRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
		View:        req.View,
		Visibility:  req.Visibility,
		Filter:      req.Filter,
		OrderBy:     req.OrderBy,
		ShowDeleted: req.ShowDeleted,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListOrganizationPipelinesResponse{
		Pipelines:     r.Pipelines,
		NextPageToken: r.NextPageToken,
		TotalSize:     r.TotalSize,
	}, nil
}

func (h *PublicHandler) ListNamespacePipelines(ctx context.Context, req *pb.ListNamespacePipelinesRequest) (resp *pb.ListNamespacePipelinesResponse, err error) {

	eventName := "ListNamespacePipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("q", filtering.TypeString),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		// Currently, we only have a "featured" tag, so we'll only support single tag filter for now.
		filtering.DeclareIdent("tag", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		// only support "recipe.components.resource_name" for now
		filtering.DeclareIdent("recipe", filtering.TypeMap(filtering.TypeString, filtering.TypeMap(filtering.TypeString, filtering.TypeString))),
		filtering.DeclareIdent("owner", filtering.TypeString),
		filtering.DeclareIdent("numberOfRuns", filtering.TypeInt),
		filtering.DeclareIdent("numberOfClones", filtering.TypeInt),
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
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
	visibility := req.GetVisibility()

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListNamespacePipelines(ctx, ns, req.GetPageSize(), req.GetPageToken(), req.GetView(), &visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return &pb.ListNamespacePipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

func (h *PublicHandler) GetUserPipeline(ctx context.Context, req *pb.GetUserPipelineRequest) (resp *pb.GetUserPipelineResponse, err error) {
	r, err := h.GetNamespacePipeline(ctx, &pb.GetNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pb.GetUserPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

func (h *PublicHandler) GetOrganizationPipeline(ctx context.Context, req *pb.GetOrganizationPipelineRequest) (resp *pb.GetOrganizationPipelineResponse, err error) {
	r, err := h.GetNamespacePipeline(ctx, &pb.GetNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pb.GetOrganizationPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

func (h *PublicHandler) GetNamespacePipeline(ctx context.Context, req *pb.GetNamespacePipelineRequest) (*pb.GetNamespacePipelineResponse, error) {

	eventName := "GetNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, req.GetView())

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

	return &pb.GetNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

func (h *PublicHandler) UpdateUserPipeline(ctx context.Context, req *pb.UpdateUserPipelineRequest) (resp *pb.UpdateUserPipelineResponse, err error) {
	r, err := h.UpdateNamespacePipeline(ctx, &pb.UpdateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Pipeline.Name, "/")[1],
		PipelineId:  strings.Split(req.Pipeline.Name, "/")[3],
		Pipeline:    req.Pipeline,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

func (h *PublicHandler) UpdateOrganizationPipeline(ctx context.Context, req *pb.UpdateOrganizationPipelineRequest) (resp *pb.UpdateOrganizationPipelineResponse, err error) {
	r, err := h.UpdateNamespacePipeline(ctx, &pb.UpdateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Pipeline.Name, "/")[1],
		PipelineId:  strings.Split(req.Pipeline.Name, "/")[3],
		Pipeline:    req.Pipeline,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateOrganizationPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

func (h *PublicHandler) UpdateNamespacePipeline(ctx context.Context, req *pb.UpdateNamespacePipelineRequest) (*pb.UpdateNamespacePipelineResponse, error) {

	eventName := "UpdateNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineReq := req.GetPipeline()
	if pbPipelineReq.Id == "" {
		pbPipelineReq.Id = req.PipelineId
	}
	pbUpdateMask := req.GetUpdateMask()

	// metadata field is type google.protobuf.Struct, which needs to be updated as a whole
	for idx, path := range pbUpdateMask.Paths {
		if strings.Split(path, ".")[0] == "metadata" {
			pbUpdateMask.Paths[idx] = "metadata"
		}
		if strings.Split(path, ".")[0] == "recipe" {
			pbUpdateMask.Paths[idx] = "recipe"
		}
	}
	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReq) {
		return nil, ErrUpdateMask
	}

	getResp, err := h.GetNamespacePipeline(ctx, &pb.GetNamespacePipelineRequest{NamespaceId: req.NamespaceId, PipelineId: req.PipelineId, View: pb.Pipeline_VIEW_RECIPE.Enum()})
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
		return &pb.UpdateNamespacePipelineResponse{Pipeline: getResp.GetPipeline()}, nil
	}

	pbPipelineToUpdate := getResp.GetPipeline()
	pbPipelineToUpdate.Recipe = nil

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

	// In the future, we'll make YAML the only input data type for pipeline
	// recipes. Until then, if the YAML recipe is empty, we'll use the JSON
	// recipe as the input data. Therefore, we set `RawRecipe` to an empty
	// string here.
	if req.GetPipeline().Recipe != nil {
		pbPipelineToUpdate.RawRecipe = ""
	}

	pbPipeline, err := h.service.UpdateNamespacePipelineByID(ctx, ns, req.PipelineId, pbPipelineToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return &pb.UpdateNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

func (h *PublicHandler) DeleteUserPipeline(ctx context.Context, req *pb.DeleteUserPipelineRequest) (resp *pb.DeleteUserPipelineResponse, err error) {
	_, err = h.DeleteNamespacePipeline(ctx, &pb.DeleteNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteUserPipelineResponse{}, nil
}
func (h *PublicHandler) DeleteOrganizationPipeline(ctx context.Context, req *pb.DeleteOrganizationPipelineRequest) (resp *pb.DeleteOrganizationPipelineResponse, err error) {
	_, err = h.DeleteNamespacePipeline(ctx, &pb.DeleteNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteOrganizationPipelineResponse{}, nil
}

func (h *PublicHandler) DeleteNamespacePipeline(ctx context.Context, req *pb.DeleteNamespacePipelineRequest) (*pb.DeleteNamespacePipelineResponse, error) {

	eventName := "DeleteNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	existPipeline, err := h.GetNamespacePipeline(ctx, &pb.GetNamespacePipelineRequest{NamespaceId: req.NamespaceId, PipelineId: req.PipelineId})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineByID(ctx, ns, req.PipelineId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
		customotel.SetEventResource(existPipeline.GetPipeline()),
	)))

	return &pb.DeleteNamespacePipelineResponse{}, nil
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

func (h *PublicHandler) ValidateUserPipeline(ctx context.Context, req *pb.ValidateUserPipelineRequest) (resp *pb.ValidateUserPipelineResponse, err error) {
	r, err := h.ValidateNamespacePipeline(ctx, &pb.ValidateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pb.ValidateUserPipelineResponse{Errors: r.Errors, Success: r.Success}, nil
}

func (h *PublicHandler) ValidateOrganizationPipeline(ctx context.Context, req *pb.ValidateOrganizationPipelineRequest) (resp *pb.ValidateOrganizationPipelineResponse, err error) {
	r, err := h.ValidateNamespacePipeline(ctx, &pb.ValidateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pb.ValidateOrganizationPipelineResponse{Errors: r.Errors, Success: r.Success}, nil
}

func (h *PublicHandler) ValidateNamespacePipeline(ctx context.Context, req *pb.ValidateNamespacePipelineRequest) (*pb.ValidateNamespacePipelineResponse, error) {

	eventName := "ValidateNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	validationErrors, err := h.service.ValidateNamespacePipelineByID(ctx, ns, req.PipelineId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	return &pb.ValidateNamespacePipelineResponse{Errors: validationErrors, Success: len(validationErrors) == 0}, nil
}

func (h *PublicHandler) RenameUserPipeline(ctx context.Context, req *pb.RenameUserPipelineRequest) (resp *pb.RenameUserPipelineResponse, err error) {
	r, err := h.RenameNamespacePipeline(ctx, &pb.RenameNamespacePipelineRequest{
		NamespaceId:   strings.Split(req.Name, "/")[1],
		PipelineId:    strings.Split(req.Name, "/")[3],
		NewPipelineId: req.NewPipelineId,
	})
	if err != nil {
		return nil, err
	}
	return &pb.RenameUserPipelineResponse{Pipeline: r.Pipeline}, nil
}

func (h *PublicHandler) RenameOrganizationPipeline(ctx context.Context, req *pb.RenameOrganizationPipelineRequest) (resp *pb.RenameOrganizationPipelineResponse, err error) {
	r, err := h.RenameNamespacePipeline(ctx, &pb.RenameNamespacePipelineRequest{
		NamespaceId:   strings.Split(req.Name, "/")[1],
		PipelineId:    strings.Split(req.Name, "/")[3],
		NewPipelineId: req.NewPipelineId,
	})
	if err != nil {
		return nil, err
	}
	return &pb.RenameOrganizationPipelineResponse{Pipeline: r.Pipeline}, nil
}

func (h *PublicHandler) RenameNamespacePipeline(ctx context.Context, req *pb.RenameNamespacePipelineRequest) (*pb.RenameNamespacePipelineResponse, error) {

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

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
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

	pbPipeline, err := h.service.UpdateNamespacePipelineIDByID(ctx, ns, req.PipelineId, newID)
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

	return &pb.RenameNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

func (h *PublicHandler) CloneNamespacePipeline(ctx context.Context, req *pb.CloneNamespacePipelineRequest) (*pb.CloneNamespacePipelineResponse, error) {

	eventName := "CloneNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.CloneNamespacePipeline(
		ctx,
		ns,
		req.PipelineId,
		req.GetTargetNamespaceId(),
		req.GetTargetPipelineId(),
		req.GetDescription(),
		req.GetSharing(),
	)
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
	return &pb.CloneNamespacePipelineResponse{}, nil
}

func (h *PublicHandler) CloneNamespacePipelineRelease(ctx context.Context, req *pb.CloneNamespacePipelineReleaseRequest) (*pb.CloneNamespacePipelineReleaseResponse, error) {

	eventName := "CloneNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipeline, err := h.service.CloneNamespacePipelineRelease(
		ctx,
		ns,
		uuid.FromStringOrNil(pipeline.Uid),
		req.ReleaseId,
		req.GetTargetNamespaceId(),
		req.GetTargetPipelineId(),
		req.GetDescription(),
		req.GetSharing(),
	)
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
	return &pb.CloneNamespacePipelineReleaseResponse{}, nil
}

func (h *PublicHandler) preTriggerNamespacePipeline(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pb.Pipeline, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", nil, false, ErrCheckRequiredFields
	}

	id := req.GetPipelineId()
	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return ns, id, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, id, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.GetPipelineId(), pb.Pipeline_VIEW_FULL)
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

func (h *PublicHandler) SendNamespacePipelineEvent(ctx context.Context, req *pb.SendNamespacePipelineEventRequest) (resp *pb.SendNamespacePipelineEventResponse, err error) {

	eventName := "SendNamespacePipelineEvent"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return nil, ErrCheckRequiredFields
	}

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, err
	}

	ok, err := h.service.CheckPipelineEventCode(ctx, ns, req.GetPipelineId(), req.GetCode())
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, service.ErrUnauthenticated
	}

	output, err := h.service.HandleNamespacePipelineEventByID(ctx, ns, req.GetPipelineId(), req.GetEvent(), req.GetData(), logUUID.String())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return &pb.SendNamespacePipelineEventResponse{Data: output}, nil
}

func (h *PublicHandler) TriggerUserPipeline(ctx context.Context, req *pb.TriggerUserPipelineRequest) (resp *pb.TriggerUserPipelineResponse, err error) {
	r, err := h.TriggerNamespacePipeline(ctx, &pb.TriggerNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerUserPipelineResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

func (h *PublicHandler) TriggerOrganizationPipeline(ctx context.Context, req *pb.TriggerOrganizationPipelineRequest) (resp *pb.TriggerOrganizationPipelineResponse, err error) {
	r, err := h.TriggerNamespacePipeline(ctx, &pb.TriggerNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerOrganizationPipelineResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

func (h *PublicHandler) TriggerNamespacePipeline(ctx context.Context, req *pb.TriggerNamespacePipelineRequest) (resp *pb.TriggerNamespacePipelineResponse, err error) {
	eventName := "TriggerNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	ns, id, _, returnTraces, err := h.preTriggerNamespacePipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	outputs, metadata, err := h.service.TriggerNamespacePipelineByID(ctx, ns, id, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	return &pb.TriggerNamespacePipelineResponse{Outputs: outputs, Metadata: metadata}, nil
}

func (h *PublicHandler) TriggerAsyncUserPipeline(ctx context.Context, req *pb.TriggerAsyncUserPipelineRequest) (resp *pb.TriggerAsyncUserPipelineResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipeline(ctx, &pb.TriggerAsyncNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerAsyncUserPipelineResponse{Operation: r.Operation}, nil
}

func (h *PublicHandler) TriggerAsyncOrganizationPipeline(ctx context.Context, req *pb.TriggerAsyncOrganizationPipelineRequest) (resp *pb.TriggerAsyncOrganizationPipelineResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipeline(ctx, &pb.TriggerAsyncNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerAsyncOrganizationPipelineResponse{Operation: r.Operation}, nil
}

func (h *PublicHandler) TriggerAsyncNamespacePipeline(ctx context.Context, req *pb.TriggerAsyncNamespacePipelineRequest) (resp *pb.TriggerAsyncNamespacePipelineResponse, err error) {

	eventName := "TriggerAsyncNamespacePipeline"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, id, dbPipeline, returnTraces, err := h.preTriggerNamespacePipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	operation, err := h.service.TriggerAsyncNamespacePipelineByID(ctx, ns, id, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
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

	return &pb.TriggerAsyncNamespacePipelineResponse{Operation: operation}, nil
}

func (h *PublicHandler) CreateUserPipelineRelease(ctx context.Context, req *pb.CreateUserPipelineReleaseRequest) (resp *pb.CreateUserPipelineReleaseResponse, err error) {
	r, err := h.CreateNamespacePipelineRelease(ctx, &pb.CreateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PipelineId:  strings.Split(req.Parent, "/")[3],
		Release:     req.Release,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateUserPipelineReleaseResponse{Release: r.Release}, nil
}

func (h *PublicHandler) CreateOrganizationPipelineRelease(ctx context.Context, req *pb.CreateOrganizationPipelineReleaseRequest) (resp *pb.CreateOrganizationPipelineReleaseResponse, err error) {
	r, err := h.CreateNamespacePipelineRelease(ctx, &pb.CreateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PipelineId:  strings.Split(req.Parent, "/")[3],
		Release:     req.Release,
	})
	if err != nil {
		return nil, err
	}
	return &pb.CreateOrganizationPipelineReleaseResponse{Release: r.Release}, nil
}

func (h *PublicHandler) CreateNamespacePipelineRelease(ctx context.Context, req *pb.CreateNamespacePipelineReleaseRequest) (*pb.CreateNamespacePipelineReleaseResponse, error) {
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

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	// TODO: We temporarily removed the release validation due to a malfunction
	// in the validation function. We'll add it back after we fix the validation
	// function.
	// _, err = h.service.ValidateNamespacePipelineByID(ctx, ns, pipeline.Id)
	// if err != nil {
	// 	return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }

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

	return &pb.CreateNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

func (h *PublicHandler) ListUserPipelineReleases(ctx context.Context, req *pb.ListUserPipelineReleasesRequest) (resp *pb.ListUserPipelineReleasesResponse, err error) {
	r, err := h.ListNamespacePipelineReleases(ctx, &pb.ListNamespacePipelineReleasesRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PipelineId:  strings.Split(req.Parent, "/")[3],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
		View:        req.View,
		Filter:      req.Filter,
		ShowDeleted: req.ShowDeleted,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListUserPipelineReleasesResponse{Releases: r.Releases, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

func (h *PublicHandler) ListOrganizationPipelineReleases(ctx context.Context, req *pb.ListOrganizationPipelineReleasesRequest) (resp *pb.ListOrganizationPipelineReleasesResponse, err error) {
	r, err := h.ListNamespacePipelineReleases(ctx, &pb.ListNamespacePipelineReleasesRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PipelineId:  strings.Split(req.Parent, "/")[3],
		PageSize:    req.PageSize,
		PageToken:   req.PageToken,
		View:        req.View,
		Filter:      req.Filter,
		ShowDeleted: req.ShowDeleted,
	})
	if err != nil {
		return nil, err
	}
	return &pb.ListOrganizationPipelineReleasesResponse{Releases: r.Releases, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

func (h *PublicHandler) ListNamespacePipelineReleases(ctx context.Context, req *pb.ListNamespacePipelineReleasesRequest) (resp *pb.ListNamespacePipelineReleasesResponse, err error) {

	eventName := "ListNamespacePipelineReleases"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
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
		filtering.DeclareIdent("createTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("updateTime", filtering.TypeTimestamp),
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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListNamespacePipelineReleases(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return &pb.ListNamespacePipelineReleasesResponse{
		Releases:      pbPipelineReleases,
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil

}

func (h *PublicHandler) GetUserPipelineRelease(ctx context.Context, req *pb.GetUserPipelineReleaseRequest) (resp *pb.GetUserPipelineReleaseResponse, err error) {
	r, err := h.GetNamespacePipelineRelease(ctx, &pb.GetNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pb.GetUserPipelineReleaseResponse{Release: r.Release}, nil
}

func (h *PublicHandler) GetOrganizationPipelineRelease(ctx context.Context, req *pb.GetOrganizationPipelineReleaseRequest) (resp *pb.GetOrganizationPipelineReleaseResponse, err error) {
	r, err := h.GetNamespacePipelineRelease(ctx, &pb.GetNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pb.GetOrganizationPipelineReleaseResponse{Release: r.Release}, nil
}

func (h *PublicHandler) GetNamespacePipelineRelease(ctx context.Context, req *pb.GetNamespacePipelineReleaseRequest) (resp *pb.GetNamespacePipelineReleaseResponse, err error) {

	eventName := "GetNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.ReleaseId, req.GetView())
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

	return &pb.GetNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

func (h *PublicHandler) UpdateUserPipelineRelease(ctx context.Context, req *pb.UpdateUserPipelineReleaseRequest) (resp *pb.UpdateUserPipelineReleaseResponse, err error) {
	r, err := h.UpdateNamespacePipelineRelease(ctx, &pb.UpdateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Release.Name, "/")[1],
		PipelineId:  strings.Split(req.Release.Name, "/")[3],
		ReleaseId:   strings.Split(req.Release.Name, "/")[5],
		Release:     req.Release,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateUserPipelineReleaseResponse{Release: r.Release}, nil
}

func (h *PublicHandler) UpdateOrganizationPipelineRelease(ctx context.Context, req *pb.UpdateOrganizationPipelineReleaseRequest) (resp *pb.UpdateOrganizationPipelineReleaseResponse, err error) {
	r, err := h.UpdateNamespacePipelineRelease(ctx, &pb.UpdateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Release.Name, "/")[1],
		PipelineId:  strings.Split(req.Release.Name, "/")[3],
		ReleaseId:   strings.Split(req.Release.Name, "/")[5],
		Release:     req.Release,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pb.UpdateOrganizationPipelineReleaseResponse{Release: r.Release}, nil
}

func (h *PublicHandler) UpdateNamespacePipelineRelease(ctx context.Context, req *pb.UpdateNamespacePipelineReleaseRequest) (resp *pb.UpdateNamespacePipelineReleaseResponse, err error) {

	eventName := "UpdateNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pbPipelineReleaseReq := req.GetRelease()
	if pbPipelineReleaseReq.Id == "" {
		pbPipelineReleaseReq.Id = req.PipelineId
	}
	pbUpdateMask := req.GetUpdateMask()

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReleaseReq) {
		return nil, ErrUpdateMask
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	getResp, err := h.GetNamespacePipelineRelease(ctx, &pb.GetNamespacePipelineReleaseRequest{NamespaceId: req.NamespaceId, PipelineId: req.PipelineId, ReleaseId: req.ReleaseId, View: pb.Pipeline_VIEW_FULL.Enum()})
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
		return &pb.UpdateNamespacePipelineReleaseResponse{Release: getResp.GetRelease()}, nil
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

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.ReleaseId, pbPipelineReleaseToUpdate)
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

	return &pb.UpdateNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil
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

	splits := strings.Split(req.GetName(), "/")
	namespaceID := splits[1]
	pipelineID := splits[3]
	releaseID := splits[5]
	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
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

func (h *PublicHandler) DeleteUserPipelineRelease(ctx context.Context, req *pb.DeleteUserPipelineReleaseRequest) (resp *pb.DeleteUserPipelineReleaseResponse, err error) {
	_, err = h.DeleteNamespacePipelineRelease(ctx, &pb.DeleteNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteUserPipelineReleaseResponse{}, nil
}
func (h *PublicHandler) DeleteOrganizationPipelineRelease(ctx context.Context, req *pb.DeleteOrganizationPipelineReleaseRequest) (resp *pb.DeleteOrganizationPipelineReleaseResponse, err error) {
	_, err = h.DeleteNamespacePipelineRelease(ctx, &pb.DeleteNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
	})
	if err != nil {
		return nil, err
	}
	return &pb.DeleteOrganizationPipelineReleaseResponse{}, nil
}

func (h *PublicHandler) DeleteNamespacePipelineRelease(ctx context.Context, req *pb.DeleteNamespacePipelineReleaseRequest) (*pb.DeleteNamespacePipelineReleaseResponse, error) {

	eventName := "DeleteNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, err := h.service.GetRscNamespace(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	existPipelineRelease, err := h.GetNamespacePipelineRelease(ctx, &pb.GetNamespacePipelineReleaseRequest{
		NamespaceId: req.NamespaceId,
		PipelineId:  req.PipelineId,
		ReleaseId:   req.ReleaseId,
	})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.ReleaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
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

	return &pb.DeleteNamespacePipelineReleaseResponse{}, nil
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

	splits := strings.Split(req.GetName(), "/")
	namespaceID := splits[1]
	pipelineID := splits[3]
	releaseID := splits[5]
	ns, err := h.service.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	existPipelineRelease, err := h.GetNamespacePipelineRelease(ctx, &pb.GetNamespacePipelineReleaseRequest{
		NamespaceId: namespaceID,
		PipelineId:  pipelineID,
		ReleaseId:   releaseID,
	})
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

func (h *PublicHandler) preTriggerNamespacePipelineRelease(ctx context.Context, req TriggerPipelineReleaseRequestInterface) (resource.Namespace, string, *pb.Pipeline, *pb.PipelineRelease, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", nil, nil, false, ErrCheckRequiredFields
	}

	ns, err := h.service.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return ns, "", nil, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.GetPipelineId(), pb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), req.GetReleaseId(), pb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", nil, nil, false, err
	}
	returnTraces := false
	if resource.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true" {
		returnTraces = true
	}

	return ns, req.GetReleaseId(), pbPipeline, pbPipelineRelease, returnTraces, nil

}

func (h *PublicHandler) TriggerUserPipelineRelease(ctx context.Context, req *pb.TriggerUserPipelineReleaseRequest) (resp *pb.TriggerUserPipelineReleaseResponse, err error) {
	r, err := h.TriggerNamespacePipelineRelease(ctx, &pb.TriggerNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerUserPipelineReleaseResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

func (h *PublicHandler) TriggerOrganizationPipelineRelease(ctx context.Context, req *pb.TriggerOrganizationPipelineReleaseRequest) (resp *pb.TriggerOrganizationPipelineReleaseResponse, err error) {
	r, err := h.TriggerNamespacePipelineRelease(ctx, &pb.TriggerNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerOrganizationPipelineReleaseResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

func (h *PublicHandler) TriggerNamespacePipelineRelease(ctx context.Context, req *pb.TriggerNamespacePipelineReleaseRequest) (resp *pb.TriggerNamespacePipelineReleaseResponse, err error) {

	eventName := "TriggerNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, releaseID, pbPipeline, _, returnTraces, err := h.preTriggerNamespacePipelineRelease(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	outputs, metadata, err := h.service.TriggerNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(customotel.NewLogMessage(
		ctx,
		span,
		logUUID.String(),
		eventName,
	)))

	return &pb.TriggerNamespacePipelineReleaseResponse{Outputs: outputs, Metadata: metadata}, nil
}

func (h *PublicHandler) TriggerAsyncUserPipelineRelease(ctx context.Context, req *pb.TriggerAsyncUserPipelineReleaseRequest) (resp *pb.TriggerAsyncUserPipelineReleaseResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipelineRelease(ctx, &pb.TriggerAsyncNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerAsyncUserPipelineReleaseResponse{Operation: r.Operation}, nil
}

func (h *PublicHandler) TriggerAsyncOrganizationPipelineRelease(ctx context.Context, req *pb.TriggerAsyncOrganizationPipelineReleaseRequest) (resp *pb.TriggerAsyncOrganizationPipelineReleaseResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipelineRelease(ctx, &pb.TriggerAsyncNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pb.TriggerAsyncOrganizationPipelineReleaseResponse{Operation: r.Operation}, nil
}

func (h *PublicHandler) TriggerAsyncNamespacePipelineRelease(ctx context.Context, req *pb.TriggerAsyncNamespacePipelineReleaseRequest) (resp *pb.TriggerAsyncNamespacePipelineReleaseResponse, err error) {

	eventName := "TriggerAsyncNamespacePipelineRelease"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	ns, releaseID, pbPipeline, pbPipelineRelease, returnTraces, err := h.preTriggerNamespacePipelineRelease(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	operation, err := h.service.TriggerAsyncNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
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

	return &pb.TriggerAsyncNamespacePipelineReleaseResponse{Operation: operation}, nil
}

func (h *PublicHandler) GetOperation(ctx context.Context, req *pb.GetOperationRequest) (*pb.GetOperationResponse, error) {

	operation, err := h.service.GetOperation(ctx, req.OperationId)
	if err != nil {
		return &pb.GetOperationResponse{}, err
	}

	return &pb.GetOperationResponse{
		Operation: operation,
	}, nil
}

func mergeInputsIntoData(inputs []*structpb.Struct, data []*pb.TriggerData) []*pb.TriggerData {
	// Backward compatibility for `inputs``
	var merged []*pb.TriggerData
	if inputs != nil {
		merged = make([]*pb.TriggerData, len(inputs))
		for idx, input := range inputs {
			merged[idx] = &pb.TriggerData{
				Variable: input,
			}
		}
	} else {
		merged = data
	}
	return merged
}

func (h *PublicHandler) ListPipelineRuns(ctx context.Context, req *pb.ListPipelineRunsRequest) (*pb.ListPipelineRunsResponse, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()
	logger.Info("ListPipelineRuns starts", zap.String("logUUID", logUUID.String()), zap.String("pipelineID", req.GetPipelineId()))

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("pipelineTriggerUID", filtering.TypeString),
		filtering.DeclareIdent("status", filtering.TypeString),
		filtering.DeclareIdent("source", filtering.TypeString),
		filtering.DeclareIdent("startTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("completeTime", filtering.TypeTimestamp),
	}...)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	resp, err := h.service.ListPipelineRuns(ctx, req, filter)
	if err != nil {
		logger.Error("failed in ListPipelineRuns", zap.String("logUUID", logUUID.String()), zap.String("pipelineID", req.GetPipelineId()), zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to list pipeline runs")
	}

	logger.Info("ListPipelineRuns finished", zap.String("logUUID", logUUID.String()), zap.String("pipelineID", req.GetPipelineId()))

	return resp, nil
}

func (h *PublicHandler) ListComponentRuns(ctx context.Context, req *pb.ListComponentRunsRequest) (*pb.ListComponentRunsResponse, error) {
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("pipelineTriggerUID", filtering.TypeString),
		filtering.DeclareIdent("componentID", filtering.TypeString),
		filtering.DeclareIdent("status", filtering.TypeString),
		filtering.DeclareIdent("startedTime", filtering.TypeTimestamp),
		filtering.DeclareIdent("completedTime", filtering.TypeTimestamp),
	}...)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	resp, err := h.service.ListComponentRuns(ctx, req, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("Failed to list component runs. error: %s", err.Error()))
	}

	return resp, nil
}

// todo: rename function to ListPipelineRunsByRequester in protobuf and update message names here
func (h *PublicHandler) ListPipelineRunsByCreditOwner(ctx context.Context, req *pb.ListPipelineRunsByCreditOwnerRequest) (*pb.ListPipelineRunsByCreditOwnerResponse, error) {
	logger, _ := logger.GetZapLogger(ctx)
	logUUID, _ := uuid.NewV4()
	logger.Info("ListPipelineRunsByRequester starts", zap.String("logUUID", logUUID.String()))

	resp, err := h.service.ListPipelineRunsByRequester(ctx, req)
	if err != nil {
		logger.Error("failed in ListPipelineRunsByRequester", zap.String("logUUID", logUUID.String()), zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to list pipeline runs")
	}

	logger.Info("ListPipelineRunsByRequester finished", zap.String("logUUID", logUUID.String()))

	return resp, nil
}
