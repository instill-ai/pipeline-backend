package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/iancoleman/strcase"
	"go.einride.tech/aip/filtering"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/checkfield"
	"github.com/instill-ai/x/sterr"

	healthcheckPB "github.com/instill-ai/protogen-go/vdp/healthcheck/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

var tracer = otel.Tracer("pipeline-backend.public-handler.tracer")

// PublicHandler handles public API
type PublicHandler struct {
	pipelinePB.UnimplementedPipelinePublicServiceServer
	service service.Service
}

type Streamer interface {
	Context() context.Context
}

type TriggerPipelineBinaryFileUploadRequestInterface interface {
	GetName() string
	GetTaskInput() *modelPB.TaskInputStream
}

type TriggerPipelineRequestInterface interface {
	GetName() string
}

func receiveFromStreamer(i interface{}) (TriggerPipelineBinaryFileUploadRequestInterface, error) {
	switch s := i.(type) {
	case pipelinePB.PipelinePublicService_TriggerSyncPipelineBinaryFileUploadServer:
		return s.Recv()
	case pipelinePB.PipelinePublicService_TriggerAsyncPipelineBinaryFileUploadServer:
		return s.Recv()
	default:
		return nil, fmt.Errorf("error recieve from stream")
	}
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

func (h *PublicHandler) CreatePipeline(ctx context.Context, req *pipelinePB.CreatePipelineRequest) (*pipelinePB.CreatePipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "CreatePipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	// Validate JSON Schema
	if err := datamodel.ValidatePipelineJSONSchema(req.GetPipeline()); err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.Pipeline, append(createRequiredFields, immutableFields...)); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Pipeline, outputOnlyFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.Pipeline.GetId()); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.CreatePipeline(owner, PBToDBPipeline(ctx, owner.GetName(), req.GetPipeline()))
	if err != nil {
		span.SetStatus(1, err.Error())
		// Manually set the custom header to have a StatusBadRequest http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusBadRequest))); err != nil {
			return &pipelinePB.CreatePipelineResponse{Pipeline: &pipelinePB.Pipeline{Recipe: &pipelinePB.Recipe{}}}, err
		}
		return &pipelinePB.CreatePipelineResponse{Pipeline: &pipelinePB.Pipeline{}}, err
	}

	pbPipeline := DBToPBPipeline(ctx, dbPipeline)
	resp := pipelinePB.CreatePipelineResponse{
		Pipeline: pbPipeline,
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		span.SetStatus(1, err.Error())
		return nil, err
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"CreatePipeline",
		false,
		"",
	)))

	return &resp, nil
}

func (h *PublicHandler) ListPipelines(ctx context.Context, req *pipelinePB.ListPipelinesRequest) (*pipelinePB.ListPipelinesResponse, error) {

	ctx, span := tracer.Start(ctx, "ListPipelines",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	var mode pipelinePB.Pipeline_Mode
	var state pipelinePB.Pipeline_State
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareFunction("time.now", filtering.NewFunctionOverload("time.now", filtering.TypeTimestamp)),
		filtering.DeclareIdent("uid", filtering.TypeString),
		filtering.DeclareIdent("id", filtering.TypeString),
		filtering.DeclareIdent("description", filtering.TypeString),
		// only support "recipe.components.resource_name" for now
		filtering.DeclareIdent("recipe", filtering.TypeMap(filtering.TypeString, filtering.TypeMap(filtering.TypeString, filtering.TypeString))),
		filtering.DeclareEnumIdent("mode", mode.Type()),
		filtering.DeclareEnumIdent("state", state.Type()),
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

	dbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(owner, req.GetPageSize(), req.GetPageToken(), isBasicView, filter)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	pbPipelines := []*pipelinePB.Pipeline{}
	for idx := range dbPipelines {
		pbPipelines = append(pbPipelines, DBToPBPipeline(ctx, &dbPipelines[idx]))
		logger.Info(string(utils.ConstructAuditLog(
			span,
			owner,
			dbPipelines[idx],
			"ListPipelines",
			false,
			"",
		)))
	}

	resp := pipelinePB.ListPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *PublicHandler) GetPipeline(ctx context.Context, req *pipelinePB.GetPipelineRequest) (*pipelinePB.GetPipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "GetPipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.GetPipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.GetPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, isBasicView)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.GetPipelineResponse{}, err
	}

	pbPipeline := DBToPBPipeline(ctx, dbPipeline)
	resp := pipelinePB.GetPipelineResponse{
		Pipeline: pbPipeline,
	}

	logMessage := utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"GetPipeline",
		false,
		"",
	)
	logger.Info(string(logMessage))

	return &resp, nil
}

func (h *PublicHandler) UpdatePipeline(ctx context.Context, req *pipelinePB.UpdatePipelineRequest) (*pipelinePB.UpdatePipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "UpdatePipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	pbPipelineReq := req.GetPipeline()
	pbUpdateMask := req.GetUpdateMask()

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReq) {
		return &pipelinePB.UpdatePipelineResponse{}, status.Error(codes.InvalidArgument, "The update_mask is invalid")
	}

	getResp, err := h.GetPipeline(ctx, &pipelinePB.GetPipelineRequest{Name: pbPipelineReq.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.UpdatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		span.SetStatus(1, err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if mask.IsEmpty() {
		return &pipelinePB.UpdatePipelineResponse{
			Pipeline: getResp.GetPipeline(),
		}, nil
	}

	pbPipelineToUpdate := getResp.GetPipeline()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReq, pbPipelineToUpdate, immutableFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.UpdatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReq, pbPipelineToUpdate)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipeline(pbPipelineToUpdate.GetId(), owner, PBToDBPipeline(ctx, owner.GetName(), pbPipelineToUpdate))
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	resp := pipelinePB.UpdatePipelineResponse{
		Pipeline: DBToPBPipeline(ctx, dbPipeline),
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"UpdatePipeline",
		false,
		"",
	)))

	return &resp, nil
}

func (h *PublicHandler) DeletePipeline(ctx context.Context, req *pipelinePB.DeletePipelineRequest) (*pipelinePB.DeletePipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "DeletePipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	existPipeline, err := h.GetPipeline(ctx, &pipelinePB.GetPipelineRequest{Name: req.GetName()})
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	if err := h.service.DeletePipeline(existPipeline.GetPipeline().GetId(), owner); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*PBToDBPipeline(ctx, owner.Id, existPipeline.Pipeline),
		"DeletePipeline",
		false,
		"",
	)))

	return &pipelinePB.DeletePipelineResponse{}, nil
}

func (h *PublicHandler) LookUpPipeline(ctx context.Context, req *pipelinePB.LookUpPipelineRequest) (*pipelinePB.LookUpPipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "LookUpPipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.LookUpPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	uidStr, err := resource.GetPermalinkUID(req.GetPermalink())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	uid, err := uuid.FromString(uidStr)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByUID(uid, owner, isBasicView)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	pbPipeline := DBToPBPipeline(ctx, dbPipeline)
	resp := pipelinePB.LookUpPipelineResponse{
		Pipeline: pbPipeline,
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"LookUpPipeline",
		false,
		"",
	)))

	return &resp, nil
}

func (h *PublicHandler) ActivatePipeline(ctx context.Context, req *pipelinePB.ActivatePipelineRequest) (*pipelinePB.ActivatePipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "ActivatePipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, activateRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ActivatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ActivatePipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ActivatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipelineState(id, owner, datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE))
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.ActivatePipelineResponse{}, err
	}

	resp := pipelinePB.ActivatePipelineResponse{
		Pipeline: DBToPBPipeline(ctx, dbPipeline),
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"ActivatePipeline",
		false,
		"",
	)))

	return &resp, nil
}

func (h *PublicHandler) DeactivatePipeline(ctx context.Context, req *pipelinePB.DeactivatePipelineRequest) (*pipelinePB.DeactivatePipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "DeactivatePipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, deactivateRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeactivatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeactivatePipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeactivatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipelineState(id, owner, datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE))
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.DeactivatePipelineResponse{}, err
	}

	resp := pipelinePB.DeactivatePipelineResponse{
		Pipeline: DBToPBPipeline(ctx, dbPipeline),
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"DeactivatePipeline",
		false,
		"",
	)))

	return &resp, nil
}

func (h *PublicHandler) RenamePipeline(ctx context.Context, req *pipelinePB.RenamePipelineRequest) (*pipelinePB.RenamePipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "RenamePipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, renameRequiredFields); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.RenamePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.RenamePipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.RenamePipelineResponse{}, err
	}

	newID := req.GetNewPipelineId()
	if err := checkfield.CheckResourceID(newID); err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.RenamePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := h.service.UpdatePipelineID(id, owner, newID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.RenamePipelineResponse{}, err
	}

	resp := pipelinePB.RenamePipelineResponse{
		Pipeline: DBToPBPipeline(ctx, dbPipeline),
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"RenamePipeline",
		false,
		"",
	)))

	return &resp, nil
}

func (h *PublicHandler) PreTriggerPipeline(ctx context.Context, req TriggerPipelineRequestInterface) (*mgmtPB.User, *datamodel.Pipeline, error) {

	logger, _ := logger.GetZapLogger(ctx)

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerRequiredFields); err != nil {
		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		return nil, nil, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return nil, nil, err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, false)
	if err != nil {
		return nil, nil, err
	}

	sources := utils.GetSourcesFromRecipe(dbPipeline.Recipe)
	if len(sources) == 0 {
		return nil, nil, status.Errorf(codes.Internal, "there is no source in pipeline's recipe")
	}

	if dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {
		switch {
		case strings.Contains(sources[0], "http") && !resource.IsGWProxied(ctx):
			st, err := sterr.CreateErrorPreconditionFailure(
				"[handler] trigger a HTTP pipeline with gRPC",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "TRIGGER",
						Subject:     fmt.Sprintf("id %s", id),
						Description: fmt.Sprintf("Pipeline id %s has a source-http connector which cannot be triggered by gRPC", id),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, nil, st.Err()

		case strings.Contains(sources[0], "grpc") && resource.IsGWProxied(ctx):
			st, err := sterr.CreateErrorPreconditionFailure(
				"[handler] trigger a HTTP pipeline with HTTP",
				[]*errdetails.PreconditionFailure_Violation{
					{
						Type:        "TRIGGER",
						Subject:     fmt.Sprintf("id %s", id),
						Description: fmt.Sprintf("Pipeline id %s has a source-grpc connector which cannot be triggered by HTTP", id),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, nil, st.Err()
		}
	}

	return owner, dbPipeline, nil

}

func (h *PublicHandler) TriggerSyncPipeline(ctx context.Context, req *pipelinePB.TriggerSyncPipelineRequest) (*pipelinePB.TriggerSyncPipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "TriggerSyncPipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, dbPipeline, err := h.PreTriggerPipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.TriggerSyncPipelineResponse{}, err
	}

	resp, err := h.service.TriggerSyncPipeline(req, owner, dbPipeline)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.TriggerSyncPipelineResponse{}, err
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"TriggerSyncPipeline",
		true,
		resp.String(),
	)))
	custom_otel.SetupSyncTriggerCounter().Add(
		ctx,
		1,
		metric.WithAttributeSet(
			attribute.NewSet(
				attribute.KeyValue{
					Key:   "ownerId",
					Value: attribute.StringValue(owner.Id),
				},
				attribute.KeyValue{
					Key:   "ownerUid",
					Value: attribute.StringValue(*owner.Uid),
				},
				attribute.KeyValue{
					Key:   "pipelineId",
					Value: attribute.StringValue(dbPipeline.ID),
				},
				attribute.KeyValue{
					Key:   "pipelineUid",
					Value: attribute.StringValue(dbPipeline.UID.String()),
				},
				attribute.KeyValue{
					Key:   "model",
					Value: attribute.StringValue(strings.Join(utils.GetModelsFromRecipe(dbPipeline.Recipe), ",")),
				},
			),
		),
	)

	return resp, nil
}

func (h *PublicHandler) TriggerAsyncPipeline(ctx context.Context, req *pipelinePB.TriggerAsyncPipelineRequest) (*pipelinePB.TriggerAsyncPipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "TriggerAsyncPipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, dbPipeline, err := h.PreTriggerPipeline(ctx, req)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.TriggerAsyncPipelineResponse{}, err
	}

	resp, err := h.service.TriggerAsyncPipeline(ctx, req, owner, dbPipeline)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.TriggerAsyncPipelineResponse{}, err
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"TriggerAsyncPipeline",
		true,
		resp.String(),
	)))

	return resp, nil
}

func (h *PublicHandler) PreTriggerPipelineBinaryFileUpload(streamer Streamer) (*mgmtPB.User, *datamodel.Pipeline, *modelPB.Model, interface{}, error) {

	owner, err := resource.GetOwner(streamer.Context(), h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		return nil, nil, nil, nil, err
	}
	data, err := receiveFromStreamer(streamer)

	if err != nil {
		return nil, nil, nil, nil, status.Errorf(codes.Unknown, "Cannot receive trigger info")
	}

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(data, triggerBinaryRequiredFields); err != nil {
		return nil, nil, nil, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := resource.GetRscNameID(data.GetName())
	if err != nil {
		return nil, nil, nil, nil, err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, false)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var textToImageInput utils.TextToImageInput
	var textGenerationInput utils.TextGenerationInput

	var allContentFiles []byte
	var fileLengths []uint64

	var model *modelPB.Model

	var firstChunk = true
	models := utils.GetModelsFromRecipe(dbPipeline.Recipe)

	for {
		data, err := receiveFromStreamer(streamer)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, nil, nil, nil, status.Errorf(codes.Internal, "failed unexpectedly while reading chunks from stream: %s", err.Error())
		}
		if firstChunk { // Get one time for first chunk.
			firstChunk = false
			pipelineName := data.GetName()
			pipeline, err := h.service.GetPipelineByID(strings.TrimSuffix(pipelineName, "pipelines/"), owner, false)
			if err != nil {
				return nil, nil, nil, nil, status.Errorf(codes.Internal, "do not find the pipeline: %s", err.Error())
			}
			if pipeline.Recipe == nil || len(models) == 0 {
				return nil, nil, nil, nil, status.Errorf(codes.Internal, "there is no model in pipeline's recipe")
			}
			model, err = h.service.GetModelByName(owner, models[0])
			if err != nil {
				return nil, nil, nil, nil, status.Errorf(codes.Internal, "could not find model: %s", err.Error())
			}

			switch model.Task {
			case modelPB.Model_TASK_CLASSIFICATION:
				fileLengths = data.GetTaskInput().GetClassification().FileLengths
				if data.GetTaskInput().GetClassification().GetContent() != nil {
					allContentFiles = append(allContentFiles, data.GetTaskInput().GetClassification().GetContent()...)
				}
			case modelPB.Model_TASK_DETECTION:
				fileLengths = data.GetTaskInput().GetDetection().FileLengths
				if data.GetTaskInput().GetDetection().GetContent() != nil {
					allContentFiles = append(allContentFiles, data.GetTaskInput().GetDetection().GetContent()...)
				}
			case modelPB.Model_TASK_KEYPOINT:
				fileLengths = data.GetTaskInput().GetKeypoint().FileLengths
				if data.GetTaskInput().GetKeypoint().GetContent() != nil {
					allContentFiles = append(allContentFiles, data.GetTaskInput().GetKeypoint().GetContent()...)
				}
			case modelPB.Model_TASK_OCR:
				fileLengths = data.GetTaskInput().GetOcr().FileLengths
				if data.GetTaskInput().GetOcr().GetContent() != nil {
					allContentFiles = append(allContentFiles, data.GetTaskInput().GetOcr().GetContent()...)
				}
			case modelPB.Model_TASK_INSTANCE_SEGMENTATION:
				fileLengths = data.GetTaskInput().GetInstanceSegmentation().FileLengths
				if data.GetTaskInput().GetInstanceSegmentation().GetContent() != nil {
					allContentFiles = append(allContentFiles, data.GetTaskInput().GetInstanceSegmentation().GetContent()...)
				}
			case modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
				fileLengths = data.GetTaskInput().GetSemanticSegmentation().FileLengths
				if data.GetTaskInput().GetSemanticSegmentation().GetContent() != nil {
					allContentFiles = append(allContentFiles, data.GetTaskInput().GetSemanticSegmentation().GetContent()...)
				}
			case modelPB.Model_TASK_TEXT_TO_IMAGE:
				textToImageInput = utils.TextToImageInput{
					Prompt:   data.GetTaskInput().GetTextToImage().GetPrompt(),
					Steps:    data.GetTaskInput().GetTextToImage().GetSteps(),
					CfgScale: data.GetTaskInput().GetTextToImage().GetCfgScale(),
					Seed:     data.GetTaskInput().GetTextToImage().GetSeed(),
					Samples:  data.GetTaskInput().GetTextToImage().GetSamples(),
				}
			case modelPB.Model_TASK_TEXT_GENERATION:
				textGenerationInput = utils.TextGenerationInput{
					Prompt:        data.GetTaskInput().GetTextGeneration().GetPrompt(),
					OutputLen:     data.GetTaskInput().GetTextGeneration().GetOutputLen(),
					BadWordsList:  data.GetTaskInput().GetTextGeneration().GetBadWordsList(),
					StopWordsList: data.GetTaskInput().GetTextGeneration().GetStopWordsList(),
					TopK:          data.GetTaskInput().GetTextGeneration().GetTopk(),
					Seed:          data.GetTaskInput().GetTextGeneration().GetSeed(),
				}
			default:
				return nil, nil, nil, nil, fmt.Errorf("unsupported task input type")
			}
			continue
		}

		switch model.Task {
		case modelPB.Model_TASK_CLASSIFICATION:
			allContentFiles = append(allContentFiles, data.GetTaskInput().GetClassification().Content...)
		case modelPB.Model_TASK_DETECTION:
			allContentFiles = append(allContentFiles, data.GetTaskInput().GetDetection().Content...)
		case modelPB.Model_TASK_KEYPOINT:
			allContentFiles = append(allContentFiles, data.GetTaskInput().GetKeypoint().Content...)
		case modelPB.Model_TASK_OCR:
			allContentFiles = append(allContentFiles, data.GetTaskInput().GetOcr().Content...)
		case modelPB.Model_TASK_INSTANCE_SEGMENTATION:
			allContentFiles = append(allContentFiles, data.GetTaskInput().GetInstanceSegmentation().Content...)
		case modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
			allContentFiles = append(allContentFiles, data.GetTaskInput().GetSemanticSegmentation().Content...)
		default:
			return nil, nil, nil, nil, fmt.Errorf("unsupported task input type")
		}

	}

	switch model.Task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		if len(fileLengths) == 0 {
			return nil, nil, nil, nil, status.Errorf(codes.InvalidArgument, "no file lengths")
		}
		if len(allContentFiles) == 0 {
			return nil, nil, nil, nil, status.Errorf(codes.InvalidArgument, "no content files")
		}
		imageInput := utils.ImageInput{
			Content:     allContentFiles,
			FileLengths: fileLengths,
		}
		return owner, dbPipeline, model, &imageInput, nil
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		return owner, dbPipeline, model, &textToImageInput, nil
	case modelPB.Model_TASK_TEXT_GENERATION:
		return owner, dbPipeline, model, &textGenerationInput, nil
	}

	return nil, nil, nil, nil, status.Errorf(codes.InvalidArgument, "unsupported task")
}

func (h *PublicHandler) TriggerPipelineBinaryFileUpload(stream pipelinePB.PipelinePublicService_TriggerSyncPipelineBinaryFileUploadServer) error {

	ctx, span := tracer.Start(stream.Context(), "TriggerPipelineBinaryFileUpload",
		trace.WithSpanKind(trace.SpanKindServer))

	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, dbPipeline, model, input, err := h.PreTriggerPipelineBinaryFileUpload(stream)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	obj, err := h.service.TriggerSyncPipelineBinaryFileUpload(owner, dbPipeline, model.Task, &input)
	if err != nil {
		span.SetStatus(1, err.Error())
		return err
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"TriggerPipelineBinaryFileUpload",
		true,
		obj.String(),
	)))
	custom_otel.SetupSyncTriggerCounter().Add(
		ctx,
		1,
		metric.WithAttributeSet(
			attribute.NewSet(
				attribute.KeyValue{
					Key:   "ownerId",
					Value: attribute.StringValue(owner.Id),
				},
				attribute.KeyValue{
					Key:   "ownerUid",
					Value: attribute.StringValue(*owner.Uid),
				},
				attribute.KeyValue{
					Key:   "pipelineId",
					Value: attribute.StringValue(dbPipeline.ID),
				},
				attribute.KeyValue{
					Key:   "pipelineUid",
					Value: attribute.StringValue(dbPipeline.UID.String()),
				},
				attribute.KeyValue{
					Key:   "model",
					Value: attribute.StringValue(strings.Join(utils.GetModelsFromRecipe(dbPipeline.Recipe), ",")),
				},
			),
		),
	)

	stream.SendAndClose(obj)

	return nil
}

func (h *PublicHandler) WatchPipeline(ctx context.Context, req *pipelinePB.WatchPipelineRequest) (*pipelinePB.WatchPipelineResponse, error) {

	ctx, span := tracer.Start(ctx, "WatchPipeline",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, err := resource.GetOwner(ctx, h.service.GetMgmtPrivateServiceClient(), h.service.GetRedisClient())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.WatchPipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.WatchPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, false)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.WatchPipelineResponse{}, err
	}
	state, err := h.service.GetResourceState(dbPipeline.UID)
	if err != nil {
		span.SetStatus(1, err.Error())
		return &pipelinePB.WatchPipelineResponse{}, err
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		owner,
		*dbPipeline,
		"WatchPipeline",
		false,
		state.String(),
	)))

	return &pipelinePB.WatchPipelineResponse{
		State: *state,
	}, nil
}
