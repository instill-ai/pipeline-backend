package handler

import (
	"bytes"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/internal/sterr"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/checkfield"

	healthcheckPB "github.com/instill-ai/protogen-go/vdp/healthcheck/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type handler struct {
	pipelinePB.UnimplementedPipelineServiceServer
	service service.Service
}

// NewHandler initiates a handler instance
func NewHandler(s service.Service) pipelinePB.PipelineServiceServer {
	datamodel.InitJSONSchema()
	return &handler{
		service: s,
	}
}

func (h *handler) Liveness(ctx context.Context, req *pipelinePB.LivenessRequest) (*pipelinePB.LivenessResponse, error) {
	return &pipelinePB.LivenessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) Readiness(ctx context.Context, req *pipelinePB.ReadinessRequest) (*pipelinePB.ReadinessResponse, error) {
	return &pipelinePB.ReadinessResponse{
		HealthCheckResponse: &healthcheckPB.HealthCheckResponse{
			Status: healthcheckPB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) CreatePipeline(ctx context.Context, req *pipelinePB.CreatePipelineRequest) (*pipelinePB.CreatePipelineResponse, error) {

	// Validate JSON Schema
	if err := datamodel.ValidatePipelineJSONSchema(req.GetPipeline()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.Pipeline, append(createRequiredFields, immutableFields...)); err != nil {
		return &pipelinePB.CreatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.Pipeline, outputOnlyFields); err != nil {
		return &pipelinePB.CreatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.Pipeline.GetId()); err != nil {
		return &pipelinePB.CreatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.CreatePipeline(PBToDBPipeline(owner, req.GetPipeline()))
	if err != nil {
		// Manually set the custom header to have a StatusBadRequest http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusBadRequest))); err != nil {
			return &pipelinePB.CreatePipelineResponse{Pipeline: &pipelinePB.Pipeline{Recipe: &pipelinePB.Recipe{}}}, err
		}
		return &pipelinePB.CreatePipelineResponse{Pipeline: &pipelinePB.Pipeline{}}, err
	}

	pbPipeline := DBToPBPipeline(dbPipeline)
	resp := pipelinePB.CreatePipelineResponse{
		Pipeline: pbPipeline,
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (h *handler) ListPipeline(ctx context.Context, req *pipelinePB.ListPipelineRequest) (*pipelinePB.ListPipelineResponse, error) {

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

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
		return &pipelinePB.ListPipelineResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

	dbPipelines, totalSize, nextPageToken, err := h.service.ListPipeline(owner, req.GetPageSize(), req.GetPageToken(), isBasicView, filter)
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

	pbPipelines := []*pipelinePB.Pipeline{}
	for _, dbPipeline := range dbPipelines {
		pbPipelines = append(pbPipelines, DBToPBPipeline(&dbPipeline))
	}

	resp := pipelinePB.ListPipelineResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}

	return &resp, nil
}

func (h *handler) GetPipeline(ctx context.Context, req *pipelinePB.GetPipelineRequest) (*pipelinePB.GetPipelineResponse, error) {

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, isBasicView)
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	pbPipeline := DBToPBPipeline(dbPipeline)
	resp := pipelinePB.GetPipelineResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *handler) UpdatePipeline(ctx context.Context, req *pipelinePB.UpdatePipelineRequest) (*pipelinePB.UpdatePipelineResponse, error) {

	owner, err := resource.GetOwner(ctx)
	if err != nil {
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
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyFields)
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
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
		return &pipelinePB.UpdatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReq, pbPipelineToUpdate)
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipeline(pbPipelineToUpdate.GetId(), owner, PBToDBPipeline(owner, pbPipelineToUpdate))
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	resp := pipelinePB.UpdatePipelineResponse{
		Pipeline: DBToPBPipeline(dbPipeline),
	}

	return &resp, nil
}

func (h *handler) DeletePipeline(ctx context.Context, req *pipelinePB.DeletePipelineRequest) (*pipelinePB.DeletePipelineResponse, error) {

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	existPipeline, err := h.GetPipeline(ctx, &pipelinePB.GetPipelineRequest{Name: req.GetName()})
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	if err := h.service.DeletePipeline(existPipeline.GetPipeline().GetId(), owner); err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	return &pipelinePB.DeletePipelineResponse{}, nil
}

func (h *handler) LookUpPipeline(ctx context.Context, req *pipelinePB.LookUpPipelineRequest) (*pipelinePB.LookUpPipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpRequiredFields); err != nil {
		return &pipelinePB.LookUpPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	isBasicView := (req.GetView() == pipelinePB.View_VIEW_BASIC) || (req.GetView() == pipelinePB.View_VIEW_UNSPECIFIED)

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	uidStr, err := resource.GetPermalinkUID(req.GetPermalink())
	if err != nil {
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	uid, err := uuid.FromString(uidStr)
	if err != nil {
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByUID(uid, owner, isBasicView)
	if err != nil {
		return &pipelinePB.LookUpPipelineResponse{}, err
	}

	pbPipeline := DBToPBPipeline(dbPipeline)
	resp := pipelinePB.LookUpPipelineResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *handler) ActivatePipeline(ctx context.Context, req *pipelinePB.ActivatePipelineRequest) (*pipelinePB.ActivatePipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, activateRequiredFields); err != nil {
		return &pipelinePB.ActivatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.ActivatePipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return &pipelinePB.ActivatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipelineState(id, owner, datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE))
	if err != nil {
		return &pipelinePB.ActivatePipelineResponse{}, err
	}

	resp := pipelinePB.ActivatePipelineResponse{
		Pipeline: DBToPBPipeline(dbPipeline),
	}

	return &resp, nil
}

func (h *handler) DeactivatePipeline(ctx context.Context, req *pipelinePB.DeactivatePipelineRequest) (*pipelinePB.DeactivatePipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, deactivateRequiredFields); err != nil {
		return &pipelinePB.DeactivatePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.DeactivatePipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return &pipelinePB.DeactivatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipelineState(id, owner, datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE))
	if err != nil {
		return &pipelinePB.DeactivatePipelineResponse{}, err
	}

	resp := pipelinePB.DeactivatePipelineResponse{
		Pipeline: DBToPBPipeline(dbPipeline),
	}

	return &resp, nil
}

func (h *handler) RenamePipeline(ctx context.Context, req *pipelinePB.RenamePipelineRequest) (*pipelinePB.RenamePipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, renameRequiredFields); err != nil {
		return &pipelinePB.RenamePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.RenamePipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return &pipelinePB.RenamePipelineResponse{}, err
	}

	newID := req.GetNewPipelineId()
	if err := checkfield.CheckResourceID(newID); err != nil {
		return &pipelinePB.RenamePipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	dbPipeline, err := h.service.UpdatePipelineID(id, owner, newID)
	if err != nil {
		return &pipelinePB.RenamePipelineResponse{}, err
	}

	resp := pipelinePB.RenamePipelineResponse{
		Pipeline: DBToPBPipeline(dbPipeline),
	}

	return &resp, nil
}

func (h *handler) TriggerPipeline(ctx context.Context, req *pipelinePB.TriggerPipelineRequest) (*pipelinePB.TriggerPipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerRequiredFields); err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, status.Error(codes.InvalidArgument, err.Error())
	}

	owner, err := resource.GetOwner(ctx)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	id, err := resource.GetRscNameID(req.GetName())
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, false)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if dbPipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {
		switch {
		case strings.Contains(dbPipeline.Recipe.Source, "http") && !resource.IsGWProxied(ctx):
			return &pipelinePB.TriggerPipelineResponse{},
				sterr.CreateErrorPreconditionFailure(
					"Trigger a HTTP pipeline with gRPC",
					"TRIGGER",
					id,
					fmt.Sprintf("Pipeline id %s has a source-http connector which cannot be triggered by gRPC", id),
				).Err()
		case strings.Contains(dbPipeline.Recipe.Source, "grpc") && resource.IsGWProxied(ctx):
			return &pipelinePB.TriggerPipelineResponse{},
				sterr.CreateErrorPreconditionFailure(
					"Trigger a gRPC pipeline with HTTP",
					"TRIGGER",
					id,
					fmt.Sprintf("Pipeline id %s has a source-grpc connector which cannot be triggered by HTTP", id),
				).Err()
		}
	}

	triggerModelResp, err := h.service.TriggerPipeline(req, dbPipeline)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if triggerModelResp == nil {
		return &pipelinePB.TriggerPipelineResponse{}, nil
	}

	resp := pipelinePB.TriggerPipelineResponse{
		Output: triggerModelResp.Output,
	}

	return &resp, nil

}

func (h *handler) TriggerPipelineBinaryFileUpload(stream pipelinePB.PipelineService_TriggerPipelineBinaryFileUploadServer) error {

	owner, err := resource.GetOwner(stream.Context())
	if err != nil {
		return err
	}

	data, err := stream.Recv()

	if err != nil {
		return status.Errorf(codes.Unknown, "Cannot receive trigger info")
	}

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(data, triggerBinaryRequiredFields); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	id, err := resource.GetRscNameID(data.GetName())
	if err != nil {
		return err
	}

	dbPipeline, err := h.service.GetPipelineByID(id, owner, false)
	if err != nil {
		return err
	}

	// Read chuck
	var fileLengths []uint64
	buf := bytes.Buffer{}
	for {
		data, err := stream.Recv()
		if len(fileLengths) == 0 {
			fileLengths = data.GetFileLengths()
		}
		if err != nil {
			if err == io.EOF {
				break
			}

			return status.Errorf(codes.Internal, "failed unexpectedly while reading chunks from stream: %s", err.Error())
		}
		if data.Content == nil {
			continue
		}

		if _, err := buf.Write(data.Content); err != nil {
			return status.Errorf(codes.Internal, "failed unexpectedly while reading chunks from stream: %s", err.Error())
		}
	}

	obj, err := h.service.TriggerPipelineBinaryFileUpload(buf, fileLengths, dbPipeline)
	if err != nil {
		return err
	}

	stream.SendAndClose(obj)

	return nil
}
