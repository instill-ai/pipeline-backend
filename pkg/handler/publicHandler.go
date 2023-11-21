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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/mod/semver"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

// PublicHandler handles public API
type PublicHandler struct {
	pipelinePB.UnimplementedPipelinePublicServiceServer
	service service.Service
}

type Streamer interface {
	Context() context.Context
}

type TriggerPipelineRequestInterface interface {
	GetName() string
}

// NewPublicHandler initiates a handler instance
func NewPublicHandler(ctx context.Context, s service.Service) pipelinePB.PipelinePublicServiceServer {
	datamodel.InitJSONSchema(ctx)
	return &PublicHandler{
		service: s,
	}
}

// GetService returns the service
func (h *PublicHandler) GetService() service.Service {
	return h.service
}

// SetService sets the service
func (h *PublicHandler) SetService(s service.Service) {
	h.service = s
}

func (h *PublicHandler) Liveness(ctx context.Context, req *pipelinePB.LivenessRequest) (*pipelinePB.LivenessResponse, error) {
	return &pipelinePB.LivenessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) Readiness(ctx context.Context, req *pipelinePB.ReadinessRequest) (*pipelinePB.ReadinessResponse, error) {
	return &pipelinePB.ReadinessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *PublicHandler) ListOperatorDefinitions(ctx context.Context, req *pipelinePB.ListOperatorDefinitionsRequest) (resp *pipelinePB.ListOperatorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListOperatorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.ListOperatorDefinitionsResponse{}
	pageSize := req.GetPageSize()
	pageToken := req.GetPageToken()
	isBasicView := (req.GetView() == pipelinePB.ListOperatorDefinitionsRequest_VIEW_BASIC) || (req.GetView() == pipelinePB.ListOperatorDefinitionsRequest_VIEW_UNSPECIFIED)

	prevLastUid := ""

	if pageToken != "" {
		_, prevLastUid, err = paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list operator error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, st.Err()
		}
	}

	if pageSize == 0 {
		pageSize = repository.DefaultPageSize
	} else if pageSize > repository.MaxPageSize {
		pageSize = repository.MaxPageSize
	}

	defs := h.service.ListOperatorDefinitions(ctx)

	startIdx := 0
	lastUid := ""
	for idx, def := range defs {
		if def.Uid == prevLastUid {
			startIdx = idx + 1
			break
		}
	}

	page := []*pipelinePB.OperatorDefinition{}
	for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.OperatorDefinition)
		page = append(page, def)
		lastUid = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUid)
	}
	for _, def := range page {
		def.Name = fmt.Sprintf("operator-definitions/%s", def.Id)
		if isBasicView {
			def.Spec = nil
		}
		resp.OperatorDefinitions = append(
			resp.OperatorDefinitions,
			def)
	}
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(len(defs))

	logger.Info("ListOperatorDefinitions")

	return resp, nil
}

func (h *PublicHandler) GetOperatorDefinition(ctx context.Context, req *pipelinePB.GetOperatorDefinitionRequest) (resp *pipelinePB.GetOperatorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetOperatorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.GetOperatorDefinitionResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	isBasicView := (req.GetView() == pipelinePB.GetOperatorDefinitionRequest_VIEW_BASIC) || (req.GetView() == pipelinePB.GetOperatorDefinitionRequest_VIEW_UNSPECIFIED)

	dbDef, err := h.service.GetOperatorDefinitionById(ctx, connID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	resp.OperatorDefinition = proto.Clone(dbDef).(*pipelinePB.OperatorDefinition)
	if isBasicView {
		resp.OperatorDefinition.Spec = nil
	}
	resp.OperatorDefinition.Name = fmt.Sprintf("operator-definitions/%s", resp.OperatorDefinition.GetId())

	logger.Info("GetOperatorDefinition")
	return resp, nil
}

func (h *PublicHandler) ListPipelines(ctx context.Context, req *pipelinePB.ListPipelinesRequest) (*pipelinePB.ListPipelinesResponse, error) {

	eventName := "ListPipelines"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	_, userUid, err := h.service.GetUser(ctx)
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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(ctx, userUid, int64(req.GetPageSize()), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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

	_, userUid, err := h.service.GetUser(ctx)

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

	pipeline, err := h.service.CreateUserPipeline(ctx, ns, userUid, pipelineToCreate)
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
	_, userUid, err := h.service.GetUser(ctx)
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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListUserPipelines(ctx, ns, userUid, int64(req.GetPageSize()), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, id, parseView(int32(*req.GetView().Enum())))

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
	_, userUid, err := h.service.GetUser(ctx)
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

	getResp, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: pbPipelineReq.GetName(), View: pipelinePB.GetUserPipelineRequest_VIEW_RECIPE.Enum()})
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

	pbPipeline, err := h.service.UpdateUserPipelineByID(ctx, ns, userUid, id, pbPipelineToUpdate)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}
	existPipeline, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	if err := h.service.DeleteUserPipelineByID(ctx, ns, userUid, id); err != nil {
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
	_, userUid, err := h.service.GetUser(ctx)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipeline, err := h.service.ValidateUserPipelineByID(ctx, ns, userUid, id)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	newID := req.GetNewPipelineId()
	if err := checkfield.CheckResourceID(newID); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	pbPipeline, err := h.service.UpdateUserPipelineIDByID(ctx, ns, userUid, id, newID)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		return ns, uuid.Nil, id, nil, false, err
	}

	pbPipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, id, service.VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, id, nil, false, err
	}
	_, err = h.service.ValidateUserPipelineByID(ctx, ns, userUid, id)
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

	outputs, metadata, err := h.service.TriggerUserPipelineByID(ctx, ns, userUid, id, req.Inputs, logUUID.String(), returnTraces)
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

	operation, err := h.service.TriggerAsyncUserPipelineByID(ctx, ns, userUid, id, req.Inputs, logUUID.String(), returnTraces)
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

func (h *PublicHandler) GetOperation(ctx context.Context, req *pipelinePB.GetOperationRequest) (*pipelinePB.GetOperationResponse, error) {

	operationId, err := resource.GetOperationID(req.Name)
	if err != nil {
		return &pipelinePB.GetOperationResponse{}, err
	}
	operation, err := h.service.GetOperation(ctx, operationId)
	if err != nil {
		return &pipelinePB.GetOperationResponse{}, err
	}

	return &pipelinePB.GetOperationResponse{
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	_, err = h.service.ValidateUserPipelineByID(ctx, ns, userUid, pipeline.Id)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	pbPipelineRelease, err := h.service.CreateUserPipelineRelease(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), req.GetRelease())
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
	_, userUid, err := h.service.GetUser(ctx)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListUserPipelineReleases(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), int64(req.GetPageSize()), req.GetPageToken(), parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, parseView(int32(*req.GetView().Enum())))
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
	_, userUid, err := h.service.GetUser(ctx)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
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

	pbPipelineRelease, err := h.service.UpdateUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, pbPipelineReleaseToUpdate)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
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

	pbPipelineRelease, err := h.service.UpdateUserPipelineReleaseIDByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, newID)
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
	_, userUid, err := h.service.GetUser(ctx)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
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
	_, userUid, err := h.service.GetUser(ctx)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.SetDefaultUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, service.VIEW_FULL)
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
	_, userUid, err := h.service.GetUser(ctx)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.RestoreUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, service.VIEW_FULL)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	pbPipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, service.VIEW_FULL)
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

	outputs, metadata, err := h.service.TriggerUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, req.Inputs, logUUID.String(), returnTraces)
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

	operation, err := h.service.TriggerAsyncUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, req.Inputs, logUUID.String(), returnTraces)
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	releaseId, err = h.service.ConvertReleaseIdAlias(ctx, ns, userUid, pipelineId, releaseId)
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, service.VIEW_BASIC)
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

	dbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, service.VIEW_BASIC)
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

func (h *PublicHandler) ListConnectorDefinitions(ctx context.Context, req *pipelinePB.ListConnectorDefinitionsRequest) (resp *pipelinePB.ListConnectorDefinitionsResponse, err error) {
	ctx, span := tracer.Start(ctx, "ListConnectorDefinitions",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.ListConnectorDefinitionsResponse{}
	pageSize := int64(req.GetPageSize())
	pageToken := req.GetPageToken()

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
	defs, totalSize, nextPageToken, err := h.service.ListConnectorDefinitions(ctx, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter)

	if err != nil {
		return nil, err
	}

	resp.ConnectorDefinitions = defs
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(totalSize)

	logger.Info("ListConnectorDefinitions")

	return resp, nil
}

func (h *PublicHandler) GetConnectorDefinition(ctx context.Context, req *pipelinePB.GetConnectorDefinitionRequest) (resp *pipelinePB.GetConnectorDefinitionResponse, err error) {
	ctx, span := tracer.Start(ctx, "GetConnectorDefinition",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.GetConnectorDefinitionResponse{}

	var connID string

	if connID, err = resource.GetRscNameID(req.GetName()); err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbDef, err := h.service.GetConnectorDefinitionByID(ctx, connID, parseView(int32(*req.GetView().Enum())))
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	resp.ConnectorDefinition = dbDef

	logger.Info("GetConnectorDefinition")
	return resp, nil

}

func (h *PublicHandler) ListConnectors(ctx context.Context, req *pipelinePB.ListConnectorsRequest) (resp *pipelinePB.ListConnectorsResponse, err error) {

	eventName := "ListConnectors"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var pageSize int64
	var pageToken string

	resp = &pipelinePB.ListConnectorsResponse{}
	pageSize = int64(req.GetPageSize())
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

	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResources, totalSize, nextPageToken, err := h.service.ListConnectors(ctx, userUid, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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

	resp.Connectors = connectorResources
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

	_, userUid, err := h.service.GetUser(ctx)
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

	connectorResource, err := h.service.GetConnectorByUID(ctx, userUid, connUID, parseView(int32(*req.GetView().Enum())), true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
	)))

	resp.Connector = connectorResource

	return resp, nil
}

func (h *PublicHandler) CreateUserConnector(ctx context.Context, req *pipelinePB.CreateUserConnectorRequest) (resp *pipelinePB.CreateUserConnectorResponse, err error) {

	eventName := "CreateUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var connID string

	resp = &pipelinePB.CreateUserConnectorResponse{}

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.Parent)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	userId, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	// TODO: ACL
	if ns.String() != resource.UserUidToUserPermalink(userUid) {
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
		return resp, st.Err()
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetConnector(), outputOnlyConnectorFields); err != nil {
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
		return resp, st.Err()
	}

	// Return error if REQUIRED fields are not provided in the requested payload
	if err := checkfield.CheckRequiredFields(req.GetConnector(), append(createConnectorRequiredFields, immutableConnectorFields...)); err != nil {
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
		return resp, st.Err()
	}

	connID = req.GetConnector().GetId()
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
		return resp, st.Err()
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
		return resp, st.Err()
	}

	req.Connector.Owner = &pipelinePB.Connector_User{User: fmt.Sprintf("users/%s", userId)}

	connectorResource, err := h.service.CreateUserConnector(ctx, ns, userUid, req.Connector)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
	)))

	resp.Connector = connectorResource

	if err != nil {
		return resp, err
	}

	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return resp, err
	}

	return resp, nil
}

func (h *PublicHandler) ListUserConnectors(ctx context.Context, req *pipelinePB.ListUserConnectorsRequest) (resp *pipelinePB.ListUserConnectorsResponse, err error) {

	eventName := "ListUserConnectors"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	var pageSize int64
	var pageToken string

	resp = &pipelinePB.ListUserConnectorsResponse{}
	pageSize = int64(req.GetPageSize())
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

	ns, _, err := h.service.GetRscNamespaceAndNameID(req.Parent)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResources, totalSize, nextPageToken, err := h.service.ListUserConnectors(ctx, ns, userUid, pageSize, pageToken, parseView(int32(*req.GetView().Enum())), filter, req.GetShowDeleted())
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

	resp.Connectors = connectorResources
	resp.NextPageToken = nextPageToken
	resp.TotalSize = int32(totalSize)

	return resp, nil

}

func (h *PublicHandler) GetUserConnector(ctx context.Context, req *pipelinePB.GetUserConnectorRequest) (resp *pipelinePB.GetUserConnectorResponse, err error) {
	eventName := "GetUserConnector"

	ctx, span := tracer.Start(ctx, eventName,
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logUUID, _ := uuid.NewV4()

	logger, _ := logger.GetZapLogger(ctx)

	resp = &pipelinePB.GetUserConnectorResponse{}

	ns, connID, err := h.service.GetRscNamespaceAndNameID(req.Name)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResource, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, parseView(int32(*req.GetView().Enum())), true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	resp.Connector = connectorResource

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
	)))

	return resp, nil
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
	_, userUid, err := h.service.GetUser(ctx)
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

	existedConnector, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_FULL, false)
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
		existedConnector, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_FULL, true)
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

	connectorResource, err := h.service.UpdateUserConnectorByID(ctx, ns, userUid, connID, pbConnectorToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	resp.Connector = connectorResource

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	dbConnector, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	if err := h.service.DeleteUserConnectorByID(ctx, ns, userUid, connID); err != nil {
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResource, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connectorResource.Uid))

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

	connectorResource, err = h.service.UpdateUserConnectorStateByID(ctx, ns, userUid, connID, *state)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
	)))

	resp.Connector = connectorResource

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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResource, err := h.service.UpdateUserConnectorStateByID(ctx, ns, userUid, connID, pipelinePB.Connector_STATE_DISCONNECTED)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
	)))

	resp.Connector = connectorResource

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
	_, userUid, err := h.service.GetUser(ctx)
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

	connectorResource, err := h.service.UpdateUserConnectorIDByID(ctx, ns, userUid, connID, connNewID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
	)))

	resp.Connector = connectorResource
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResource, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
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

	state, err := h.service.GetConnectorState(uuid.FromStringOrNil(connectorResource.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		logger.Info(string(custom_otel.NewLogMessage(
			span,
			logUUID.String(),
			userUid,
			eventName,
			custom_otel.SetErrorMessage(err.Error()),
			custom_otel.SetEventResource(connectorResource),
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResource, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_BASIC, true)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	state, err := h.service.CheckConnectorByUID(ctx, uuid.FromStringOrNil(connectorResource.Uid))

	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	logger.Info(string(custom_otel.NewLogMessage(
		span,
		logUUID.String(),
		userUid,
		eventName,
		custom_otel.SetEventResource(connectorResource),
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
	_, userUid, err := h.service.GetUser(ctx)
	if err != nil {
		span.SetStatus(1, err.Error())
		return resp, err
	}

	connectorResource, err := h.service.GetUserConnectorByID(ctx, ns, userUid, connID, service.VIEW_FULL, true)
	if err != nil {
		return resp, err
	}
	if connectorResource.Tombstone {
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
		ConnectorID:            connectorResource.Id,
		ConnectorUID:           connectorResource.Uid,
		ConnectorExecuteUID:    logUUID.String(),
		ConnectorDefinitionUid: connectorResource.ConnectorDefinition.Uid,
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
