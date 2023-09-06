package handler

import (
	"context"
	"fmt"
	"net/http"

	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/mod/semver"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/operator"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	healthcheckPB "github.com/instill-ai/protogen-go/common/healthcheck/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the public_handler, we should convert all id to uuid when calling service

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

// PublicHandler handles public API
type PublicHandler struct {
	pipelinePB.UnimplementedPipelinePublicServiceServer
	service  service.Service
	operator operator.Operator
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
		service:  s,
		operator: operator.InitOperator(),
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
	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

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

	defs := h.operator.ListOperatorDefinitions()

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
	resp.TotalSize = int64(len(defs))

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
	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	dbDef, err := h.operator.GetOperatorDefinitionById(connID)
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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(ctx, userUid, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), filter, req.GetShowDeleted())
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
		TotalSize:     totalSize,
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
	if err := checkfield.CheckRequiredFields(req.Pipeline, append(createRequiredFields, immutableFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreateUserPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Pipeline, outputOnlyFields); err != nil {
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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListUserPipelines(ctx, ns, userUid, req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), filter, req.GetShowDeleted())
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
		TotalSize:     totalSize,
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

	pbPipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, id, parseView(req.GetView()))

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

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReq) {
		return nil, status.Error(codes.InvalidArgument, "The update_mask is invalid")
	}

	getResp, err := h.GetUserPipeline(ctx, &pipelinePB.GetUserPipelineRequest{Name: pbPipelineReq.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
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
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReq, pbPipelineToUpdate, immutableFields); err != nil {
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
	if err := checkfield.CheckRequiredFields(req, lookUpRequiredFields); err != nil {
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

	pbPipeline, err := h.service.GetPipelineByUID(ctx, userUid, uid, parseView(req.GetView()))
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

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, validateRequiredFields); err != nil {
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

	pbPipeline, err := h.service.ValidateUserPipelineByID(ctx, ns, userUid, id)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
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
	if err := checkfield.CheckRequiredFields(req, renameRequiredFields); err != nil {
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
	if err := checkfield.CheckRequiredFields(req, triggerRequiredFields); err != nil {
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

	pbPipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, id, pipelinePB.View_VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, id, nil, false, err
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

	dataPoint := utils.UsageMetricData{
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
		_ = h.service.WriteNewDataPoint(ctx, dataPoint)
		return nil, err
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
	if err := h.service.WriteNewDataPoint(ctx, dataPoint); err != nil {
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
		return nil, err
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
	if err := checkfield.CheckRequiredFields(req.Release, append(releaseCreateRequiredFields, immutableFields...)); err != nil {
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	_, err = h.service.ValidateUserPipelineByID(ctx, ns, userUid, pipeline.Id)
	if err != nil {
		return nil, err
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListUserPipelineReleases(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), req.GetPageSize(), req.GetPageToken(), parseView(req.GetView()), filter, req.GetShowDeleted())
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
		TotalSize:     totalSize,
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, parseView(req.GetView()))
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
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
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReleaseReq, pbPipelineReleaseToUpdate, immutableFields); err != nil {
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.SetDefaultUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, pipelinePB.View_VIEW_FULL)
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.RestoreUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, pipelinePB.View_VIEW_FULL)
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
	if err := checkfield.CheckRequiredFields(req, triggerRequiredFields); err != nil {
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

	pbPipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_FULL)
	if err != nil {
		return ns, uuid.Nil, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pbPipeline.Uid), releaseId, pipelinePB.View_VIEW_FULL)
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

	dataPoint := utils.UsageMetricData{
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
		_ = h.service.WriteNewDataPoint(ctx, dataPoint)
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
	if err := h.service.WriteNewDataPoint(ctx, dataPoint); err != nil {
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

	pipeline, err := h.service.GetUserPipelineByID(ctx, ns, userUid, pipelineId, pipelinePB.View_VIEW_BASIC)
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

	dbPipelineRelease, err := h.service.GetUserPipelineReleaseByID(ctx, ns, userUid, uuid.FromStringOrNil(pipeline.Uid), releaseId, pipelinePB.View_VIEW_BASIC)
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
	state, err := h.service.GetResourceState(uuid.FromStringOrNil(dbPipelineRelease.Uid))
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
