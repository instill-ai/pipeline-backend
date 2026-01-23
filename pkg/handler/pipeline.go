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
	"golang.org/x/mod/semver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/checkfield"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
	errorsx "github.com/instill-ai/x/errors"
	resourcex "github.com/instill-ai/x/resource"
)

// parsePipelineNamespaceFromParent extracts namespace ID from parent string.
// Format: namespaces/{namespace}
func parsePipelineNamespaceFromParent(parent string) (string, error) {
	parts := strings.Split(parent, "/")
	if len(parts) < 2 || parts[0] != "namespaces" {
		return "", fmt.Errorf("invalid parent format: %s", parent)
	}
	return parts[1], nil
}

// parsePipelineFromName extracts namespace ID and pipeline ID from name string.
// Format: namespaces/{namespace}/pipelines/{pipeline}
func parsePipelineFromName(name string) (namespaceID, pipelineID string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) < 4 || parts[0] != "namespaces" || parts[2] != "pipelines" {
		return "", "", fmt.Errorf("invalid pipeline name format: %s", name)
	}
	return parts[1], parts[3], nil
}

// parseReleaseFromName extracts namespace ID, pipeline ID and release ID from name string.
// Format: namespaces/{namespace}/pipelines/{pipeline}/releases/{release}
func parseReleaseFromName(name string) (namespaceID, pipelineID, releaseID string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) < 6 || parts[0] != "namespaces" || parts[2] != "pipelines" || parts[4] != "releases" {
		return "", "", "", fmt.Errorf("invalid release name format: %s", name)
	}
	return parts[1], parts[3], parts[5], nil
}

// ListPipelinesAdmin returns a paginated list of pipelines.
func (h *PrivateHandler) ListPipelinesAdmin(ctx context.Context, req *pipelinepb.ListPipelinesAdminRequest) (*pipelinepb.ListPipelinesAdminResponse, error) {

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
		return &pipelinepb.ListPipelinesAdminResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pipelinepb.ListPipelinesAdminResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelinesAdmin(ctx, req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		return &pipelinepb.ListPipelinesAdminResponse{}, err
	}

	resp := pipelinepb.ListPipelinesAdminResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

// LookUpPipelineAdmin returns the details of a pipeline.
func (h *PrivateHandler) LookUpPipelineAdmin(ctx context.Context, req *pipelinepb.LookUpPipelineAdminRequest) (*pipelinepb.LookUpPipelineAdminResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpPipelineRequiredFields); err != nil {
		return &pipelinepb.LookUpPipelineAdminResponse{}, errorsx.ErrCheckRequiredFields
	}

	view := pipelinepb.Pipeline_VIEW_BASIC
	if req.GetView() != pipelinepb.Pipeline_VIEW_UNSPECIFIED {
		view = req.GetView()
	}

	uid, err := resource.GetRscPermalinkUID(req.GetPermalink())
	if err != nil {
		return &pipelinepb.LookUpPipelineAdminResponse{}, err
	}
	pbPipeline, err := h.service.GetPipelineByUIDAdmin(ctx, uid, view)
	if err != nil {
		return &pipelinepb.LookUpPipelineAdminResponse{}, err
	}

	resp := pipelinepb.LookUpPipelineAdminResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

// GetHubStats returns the stats of the hub.
func (h *PublicHandler) GetHubStats(ctx context.Context, req *pipelinepb.GetHubStatsRequest) (*pipelinepb.GetHubStatsResponse, error) {

	if err := authenticateUser(ctx, true); err != nil {
		return &pipelinepb.GetHubStatsResponse{}, err
	}

	resp, err := h.service.GetHubStats(ctx)

	if err != nil {
		return &pipelinepb.GetHubStatsResponse{}, err
	}

	return resp, nil
}

// ListPublicPipelines returns a paginated list of public pipelines.
func (h *PublicHandler) ListPublicPipelines(ctx context.Context, req *pipelinepb.ListPublicPipelinesRequest) (*pipelinepb.ListPublicPipelinesResponse, error) {

	if err := authenticateUser(ctx, true); err != nil {
		return &pipelinepb.ListPublicPipelinesResponse{}, err
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
		return &pipelinepb.ListPublicPipelinesResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pipelinepb.ListPublicPipelinesResponse{}, err
	}

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return &pipelinepb.ListPublicPipelinesResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPublicPipelines(
		ctx, req.GetPageSize(), req.GetPageToken(), req.GetView(), req.Visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		return &pipelinepb.ListPublicPipelinesResponse{}, err
	}

	resp := pipelinepb.ListPublicPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

// CreatePipeline creates a new pipeline for a namespace.
func (h *PublicHandler) CreatePipeline(ctx context.Context, req *pipelinepb.CreatePipelineRequest) (resp *pipelinepb.CreatePipelineResponse, err error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.GetPipeline(), append(createPipelineRequiredFields, immutablePipelineFields...)); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetPipeline(), outputOnlyPipelineFields); err != nil {
		return nil, errorsx.ErrCheckOutputOnlyFields
	}

	// Note: Per AIP standard, id is OUTPUT_ONLY and auto-generated by the server.
	// The server generates the id (e.g., "pip-8f3A2k9E7c1") in the BeforeCreate hook.

	// Parse namespace ID from parent: namespaces/{namespace}
	namespaceID, err := parsePipelineNamespaceFromParent(req.GetParent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)

	if err != nil {
		return nil, err
	}

	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipelineToCreate := req.GetPipeline()

	pipeline, err := h.service.CreatePipeline(ctx, ns, pipelineToCreate)

	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &pipelinepb.CreatePipelineResponse{Pipeline: pipeline}, nil
}

// ListPipelines returns a paginated list of pipelines for a namespace.
func (h *PublicHandler) ListPipelines(ctx context.Context, req *pipelinepb.ListPipelinesRequest) (resp *pipelinepb.ListPipelinesResponse, err error) {

	// Parse namespace ID from parent: namespaces/{namespace}
	namespaceID, err := parsePipelineNamespaceFromParent(req.GetParent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
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
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}
	visibility := req.GetVisibility()

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(ctx, ns, req.GetPageSize(), req.GetPageToken(), req.GetView(), &visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ListPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

// GetPipeline returns the details of a pipeline for a namespace.
func (h *PublicHandler) GetPipeline(ctx context.Context, req *pipelinepb.GetPipelineRequest) (*pipelinepb.GetPipelineResponse, error) {

	// Parse namespace ID and pipeline ID from name: namespaces/{namespace}/pipelines/{pipeline}
	namespaceID, pipelineID, err := parsePipelineFromName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pbPipeline, err := h.service.GetPipelineByID(ctx, ns, pipelineID, req.GetView())

	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetPipelineResponse{Pipeline: pbPipeline}, nil
}

// UpdatePipeline updates a pipeline for a namespace.
func (h *PublicHandler) UpdatePipeline(ctx context.Context, req *pipelinepb.UpdatePipelineRequest) (*pipelinepb.UpdatePipelineResponse, error) {

	// Parse namespace ID and pipeline ID from pipeline.name: namespaces/{namespace}/pipelines/{pipeline}
	namespaceID, pipelineID, err := parsePipelineFromName(req.GetPipeline().GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pbPipelineReq := req.GetPipeline()
	if pbPipelineReq.Id == "" {
		pbPipelineReq.Id = pipelineID
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
		return nil, errorsx.ErrUpdateMask
	}

	getResp, err := h.GetPipeline(ctx, &pipelinepb.GetPipelineRequest{Name: req.GetPipeline().GetName(), View: pipelinepb.Pipeline_VIEW_RECIPE.Enum()})
	if err != nil {
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, outputOnlyPipelineFields)
	if err != nil {
		return nil, errorsx.ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		return nil, errorsx.ErrFieldMask
	}

	if mask.IsEmpty() {
		return &pipelinepb.UpdatePipelineResponse{Pipeline: getResp.GetPipeline()}, nil
	}

	pbPipelineToUpdate := getResp.GetPipeline()
	pbPipelineToUpdate.Recipe = nil

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReq, pbPipelineToUpdate, immutablePipelineFields); err != nil {
		return nil, errorsx.ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReq, pbPipelineToUpdate)
	if err != nil {
		return nil, err
	}

	// In the future, we'll make YAML the only input data type for pipeline
	// recipes. Until then, if the YAML recipe is empty, we'll use the JSON
	// recipe as the input data. Therefore, we set `RawRecipe` to an empty
	// string here.
	if req.GetPipeline().Recipe != nil {
		pbPipelineToUpdate.RawRecipe = ""
	}

	pbPipeline, err := h.service.UpdatePipelineByID(ctx, ns, pipelineID, pbPipelineToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdatePipelineResponse{Pipeline: pbPipeline}, nil
}

// DeletePipeline deletes a pipeline for a namespace.
func (h *PublicHandler) DeletePipeline(ctx context.Context, req *pipelinepb.DeletePipelineRequest) (*pipelinepb.DeletePipelineResponse, error) {

	// Parse namespace ID and pipeline ID from name: namespaces/{namespace}/pipelines/{pipeline}
	namespaceID, pipelineID, err := parsePipelineFromName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	_, err = h.GetPipeline(ctx, &pipelinepb.GetPipelineRequest{Name: req.GetName()})
	if err != nil {
		return nil, err
	}

	if err := h.service.DeletePipelineByID(ctx, ns, pipelineID); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeletePipelineResponse{}, nil
}

// ValidatePipeline validates a pipeline for a namespace.
func (h *PublicHandler) ValidatePipeline(ctx context.Context, req *pipelinepb.ValidatePipelineRequest) (*pipelinepb.ValidatePipelineResponse, error) {

	// Parse namespace_id and pipeline_id from name resource name
	// Format: namespaces/{namespace}/pipelines/{pipeline}
	name := req.GetName()
	parts := strings.Split(name, "/")
	if len(parts) < 4 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid resource name: %s", name))
	}
	namespaceID := parts[1]
	pipelineID := parts[3]

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	validationErrors, err := h.service.ValidatePipelineByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	return &pipelinepb.ValidatePipelineResponse{Errors: validationErrors, Success: len(validationErrors) == 0}, nil
}

// RenamePipeline renames a pipeline for a namespace.
func (h *PublicHandler) RenamePipeline(ctx context.Context, req *pipelinepb.RenamePipelineRequest) (*pipelinepb.RenamePipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, renamePipelineRequiredFields); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	// Parse namespace_id and pipeline_id from name resource name
	// Format: namespaces/{namespace}/pipelines/{pipeline}
	name := req.GetName()
	parts := strings.Split(name, "/")
	if len(parts) < 4 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid resource name: %s", name))
	}
	namespaceID := parts[1]
	pipelineID := parts[3]

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	newID := req.GetNewPipelineId()
	if err := checkfield.CheckResourceID(newID); err != nil {
		return nil, fmt.Errorf("%w: invalid pipeline ID: %w", errorsx.ErrInvalidArgument, err)
	}

	pbPipeline, err := h.service.UpdatePipelineIDByID(ctx, ns, pipelineID, newID)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.RenamePipelineResponse{Pipeline: pbPipeline}, nil
}

// ClonePipeline clones a pipeline for a namespace.
func (h *PublicHandler) ClonePipeline(ctx context.Context, req *pipelinepb.ClonePipelineRequest) (*pipelinepb.ClonePipelineResponse, error) {

	// Parse namespace_id and pipeline_id from name resource name (source)
	// Format: namespaces/{namespace}/pipelines/{pipeline}
	name := req.GetName()
	parts := strings.Split(name, "/")
	if len(parts) < 4 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid source resource name: %s", name))
	}
	namespaceID := parts[1]
	pipelineID := parts[3]

	// Parse target namespace_id and pipeline_id from target resource name
	// Format: namespaces/{namespace}/pipelines/{pipeline}
	target := req.GetTarget()
	targetParts := strings.Split(target, "/")
	if len(targetParts) < 4 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid target resource name: %s", target))
	}
	targetNamespaceID := targetParts[1]
	targetPipelineID := targetParts[3]

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	_, err = h.service.ClonePipeline(
		ctx,
		ns,
		pipelineID,
		targetNamespaceID,
		targetPipelineID,
		req.GetDescription(),
		req.GetSharing(),
	)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ClonePipelineResponse{}, nil
}

// ClonePipelineRelease clones a pipeline release for a namespace.
func (h *PublicHandler) ClonePipelineRelease(ctx context.Context, req *pipelinepb.ClonePipelineReleaseRequest) (*pipelinepb.ClonePipelineReleaseResponse, error) {

	// Parse namespace_id, pipeline_id, and release_id from name resource name (source)
	// Format: namespaces/{namespace}/pipelines/{pipeline}/releases/{release}
	name := req.GetName()
	parts := strings.Split(name, "/")
	if len(parts) < 6 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid source resource name: %s", name))
	}
	namespaceID := parts[1]
	pipelineID := parts[3]
	releaseID := parts[5]

	// Parse target namespace_id and pipeline_id from target resource name
	// Format: namespaces/{namespace}/pipelines/{pipeline}
	target := req.GetTarget()
	targetParts := strings.Split(target, "/")
	if len(targetParts) < 4 {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid target resource name: %s", target))
	}
	targetNamespaceID := targetParts[1]
	targetPipelineID := targetParts[3]

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	_, err = h.service.ClonePipelineRelease(
		ctx,
		ns,
		pipelineUID,
		releaseID,
		targetNamespaceID,
		targetPipelineID,
		req.GetDescription(),
		req.GetSharing(),
	)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ClonePipelineReleaseResponse{}, nil
}

// preTriggerPipeline is a helper function to pre-trigger a namespace pipeline.
func (h *PublicHandler) preTriggerPipeline(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pipelinepb.Pipeline, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", nil, false, errorsx.ErrCheckRequiredFields
	}

	// Parse namespace_id and pipeline_id from name resource name
	// Format: namespaces/{namespace}/pipelines/{pipeline}
	name := req.GetName()
	parts := strings.Split(name, "/")
	if len(parts) < 4 {
		return resource.Namespace{}, "", nil, false, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid resource name: %s", name))
	}
	namespaceID := parts[1]
	pipelineID := parts[3]

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return ns, pipelineID, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, pipelineID, nil, false, err
	}

	pbPipeline, err := h.service.GetPipelineByID(ctx, ns, pipelineID, pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, pipelineID, nil, false, err
	}
	// _, err = h.service.ValidatePipelineByID(ctx, ns,  id)
	// if err != nil {
	// 	return ns, nil, id, nil, false, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }
	returnTraces := resourcex.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true"

	return ns, pipelineID, pbPipeline, returnTraces, nil

}

// TriggerPipeline triggers a pipeline for a namespace.
func (h *PublicHandler) TriggerPipeline(ctx context.Context, req *pipelinepb.TriggerPipelineRequest) (resp *pipelinepb.TriggerPipelineResponse, err error) {

	ns, id, _, returnTraces, err := h.preTriggerPipeline(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	outputs, metadata, err := h.service.TriggerPipelineByID(ctx, ns, id, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	// TODO: it would be useful to return the trigger UID here.
	return &pipelinepb.TriggerPipelineResponse{Outputs: outputs, Metadata: metadata}, nil
}

// TriggerAsyncPipeline triggers an async pipeline for a namespace.
func (h *PublicHandler) TriggerAsyncPipeline(ctx context.Context, req *pipelinepb.TriggerAsyncPipelineRequest) (resp *pipelinepb.TriggerAsyncPipelineResponse, err error) {

	ns, id, _, returnTraces, err := h.preTriggerPipeline(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	operation, err := h.service.TriggerAsyncPipelineByID(ctx, ns, id, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerAsyncPipelineResponse{Operation: operation}, nil
}

// CreatePipelineRelease creates a pipeline release for a namespace.
func (h *PublicHandler) CreatePipelineRelease(ctx context.Context, req *pipelinepb.CreatePipelineReleaseRequest) (*pipelinepb.CreatePipelineReleaseResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.GetRelease(), append(releaseCreateRequiredFields, immutablePipelineFields...)); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetRelease(), releaseOutputOnlyFields); err != nil {
		return nil, fmt.Errorf("%w: %v", errorsx.ErrCheckOutputOnlyFields, err)
	}

	// Return error if resource ID does not a semantic version
	if !semver.IsValid(req.GetRelease().GetId()) {
		return nil, errorsx.ErrSematicVersion
	}

	// Parse namespace ID and pipeline ID from parent: namespaces/{namespace}/pipelines/{pipeline}
	namespaceID, pipelineID, err := parsePipelineFromName(req.GetParent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	// TODO: We temporarily removed the release validation due to a malfunction
	// in the validation function. We'll add it back after we fix the validation
	// function.
	// _, err = h.service.ValidatePipelineByID(ctx, ns, pipelineID)
	// if err != nil {
	// 	return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }

	pbPipelineRelease, err := h.service.CreatePipelineRelease(ctx, ns, pipelineUID, req.GetRelease())
	if err != nil {
		// Manually set the custom header to have a StatusBadRequest http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusBadRequest))); err != nil {
			return nil, err
		}
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &pipelinepb.CreatePipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

// ListPipelineReleases lists pipeline releases for a namespace.
func (h *PublicHandler) ListPipelineReleases(ctx context.Context, req *pipelinepb.ListPipelineReleasesRequest) (resp *pipelinepb.ListPipelineReleasesResponse, err error) {

	// Parse namespace ID and pipeline ID from parent: namespaces/{namespace}/pipelines/{pipeline}
	namespaceID, pipelineID, err := parsePipelineFromName(req.GetParent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
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
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListPipelineReleases(ctx, ns, pipelineUID, req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ListPipelineReleasesResponse{
		Releases:      pbPipelineReleases,
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil

}

// GetPipelineRelease gets a pipeline release for a namespace.
func (h *PublicHandler) GetPipelineRelease(ctx context.Context, req *pipelinepb.GetPipelineReleaseRequest) (resp *pipelinepb.GetPipelineReleaseResponse, err error) {

	// Parse namespace ID, pipeline ID and release ID from name: namespaces/{namespace}/pipelines/{pipeline}/releases/{release}
	namespaceID, pipelineID, releaseID, err := parseReleaseFromName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetPipelineReleaseByID(ctx, ns, pipelineUID, releaseID, req.GetView())
	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetPipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

// UpdatePipelineRelease updates a pipeline release for a namespace.
func (h *PublicHandler) UpdatePipelineRelease(ctx context.Context, req *pipelinepb.UpdatePipelineReleaseRequest) (resp *pipelinepb.UpdatePipelineReleaseResponse, err error) {

	// Parse namespace ID, pipeline ID and release ID from release.name: namespaces/{namespace}/pipelines/{pipeline}/releases/{release}
	namespaceID, pipelineID, releaseID, err := parseReleaseFromName(req.GetRelease().GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pbPipelineReleaseReq := req.GetRelease()
	if pbPipelineReleaseReq.Id == "" {
		pbPipelineReleaseReq.Id = releaseID
	}
	pbUpdateMask := req.GetUpdateMask()

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReleaseReq) {
		return nil, errorsx.ErrUpdateMask
	}

	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	getResp, err := h.GetPipelineRelease(ctx, &pipelinepb.GetPipelineReleaseRequest{Name: req.GetRelease().GetName(), View: pipelinepb.Pipeline_VIEW_FULL.Enum()})
	if err != nil {
		return nil, err
	}

	pbUpdateMask, err = checkfield.CheckUpdateOutputOnlyFields(pbUpdateMask, releaseOutputOnlyFields)
	if err != nil {
		return nil, errorsx.ErrCheckOutputOnlyFields
	}

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(pbUpdateMask, strcase.ToCamel)
	if err != nil {
		return nil, errorsx.ErrFieldMask
	}

	if mask.IsEmpty() {
		return &pipelinepb.UpdatePipelineReleaseResponse{Release: getResp.GetRelease()}, nil
	}

	pbPipelineReleaseToUpdate := getResp.GetRelease()

	// Return error if IMMUTABLE fields are intentionally changed
	if err := checkfield.CheckUpdateImmutableFields(pbPipelineReleaseReq, pbPipelineReleaseToUpdate, immutablePipelineFields); err != nil {
		return nil, errorsx.ErrCheckUpdateImmutableFields
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReleaseReq, pbPipelineReleaseToUpdate)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.UpdatePipelineReleaseByID(ctx, ns, pipelineUID, releaseID, pbPipelineReleaseToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdatePipelineReleaseResponse{Release: pbPipelineRelease}, nil
}

// RenamePipelineReleaseRequestInterface is the interface for the request to rename a pipeline release.
// DeletePipelineRelease deletes a pipeline release for a namespace.
func (h *PublicHandler) DeletePipelineRelease(ctx context.Context, req *pipelinepb.DeletePipelineReleaseRequest) (*pipelinepb.DeletePipelineReleaseResponse, error) {

	// Parse namespace ID, pipeline ID and release ID from name: namespaces/{namespace}/pipelines/{pipeline}/releases/{release}
	namespaceID, pipelineID, releaseID, err := parseReleaseFromName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	_, err = h.GetPipelineRelease(ctx, &pipelinepb.GetPipelineReleaseRequest{
		Name: req.GetName(),
	})
	if err != nil {
		return nil, err
	}

	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	if err := h.service.DeletePipelineReleaseByID(ctx, ns, pipelineUID, releaseID); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeletePipelineReleaseResponse{}, nil
}

func (h *PublicHandler) preTriggerPipelineRelease(ctx context.Context, req TriggerPipelineReleaseRequestInterface) (resource.Namespace, string, uuid.UUID, *pipelinepb.PipelineRelease, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", uuid.Nil, nil, false, errorsx.ErrCheckRequiredFields
	}

	// Parse namespace_id, pipeline_id, and release_id from name resource name
	// Format: namespaces/{namespace}/pipelines/{pipeline}/releases/{release}
	name := req.GetName()
	parts := strings.Split(name, "/")
	if len(parts) < 6 {
		return resource.Namespace{}, "", uuid.Nil, nil, false, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid resource name: %s", name))
	}
	namespaceID := parts[1]
	pipelineID := parts[3]
	releaseID := parts[5]

	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return ns, "", uuid.Nil, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, "", uuid.Nil, nil, false, err
	}

	pipelineUID, err := h.service.GetPipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return ns, "", uuid.Nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetPipelineReleaseByID(ctx, ns, pipelineUID, releaseID, pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", uuid.Nil, nil, false, err
	}
	returnTraces := resourcex.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true"

	return ns, releaseID, pipelineUID, pbPipelineRelease, returnTraces, nil

}

// TriggerPipelineRelease triggers a pipeline release for a namespace.
func (h *PublicHandler) TriggerPipelineRelease(ctx context.Context, req *pipelinepb.TriggerPipelineReleaseRequest) (resp *pipelinepb.TriggerPipelineReleaseResponse, err error) {

	ns, releaseID, pipelineUID, _, returnTraces, err := h.preTriggerPipelineRelease(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	outputs, metadata, err := h.service.TriggerPipelineReleaseByID(ctx, ns, pipelineUID, releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerPipelineReleaseResponse{Outputs: outputs, Metadata: metadata}, nil
}

// TriggerAsyncPipelineRelease triggers an async pipeline release for a namespace.
func (h *PublicHandler) TriggerAsyncPipelineRelease(ctx context.Context, req *pipelinepb.TriggerAsyncPipelineReleaseRequest) (resp *pipelinepb.TriggerAsyncPipelineReleaseResponse, err error) {

	ns, releaseID, pipelineUID, _, returnTraces, err := h.preTriggerPipelineRelease(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	operation, err := h.service.TriggerAsyncPipelineReleaseByID(ctx, ns, pipelineUID, releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerAsyncPipelineReleaseResponse{Operation: operation}, nil
}

// GetOperation gets an operation.
func (h *PublicHandler) GetOperation(ctx context.Context, req *pipelinepb.GetOperationRequest) (*pipelinepb.GetOperationResponse, error) {

	operation, err := h.service.GetOperation(ctx, req.OperationId)
	if err != nil {
		return &pipelinepb.GetOperationResponse{}, err
	}

	return &pipelinepb.GetOperationResponse{
		Operation: operation,
	}, nil
}

func mergeInputsIntoData(inputs []*structpb.Struct, data []*pipelinepb.TriggerData) []*pipelinepb.TriggerData {
	// Backward compatibility for `inputs``
	var merged []*pipelinepb.TriggerData
	if inputs != nil {
		merged = make([]*pipelinepb.TriggerData, len(inputs))
		for idx, input := range inputs {
			merged[idx] = &pipelinepb.TriggerData{
				Variable: input,
			}
		}
	} else {
		merged = data
	}
	return merged
}

// ListPipelineRuns lists pipeline runs.
func (h *PublicHandler) ListPipelineRuns(ctx context.Context, req *pipelinepb.ListPipelineRunsRequest) (*pipelinepb.ListPipelineRunsResponse, error) {
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
		return nil, status.Error(codes.Internal, "Failed to list pipeline runs")
	}

	return resp, nil
}

// ListComponentRuns lists component runs.
func (h *PublicHandler) ListComponentRuns(ctx context.Context, req *pipelinepb.ListComponentRunsRequest) (*pipelinepb.ListComponentRunsResponse, error) {
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

// ListPipelineRunsByRequester lists pipeline runs by requester.
func (h *PublicHandler) ListPipelineRunsByRequester(ctx context.Context, req *pipelinepb.ListPipelineRunsByRequesterRequest) (*pipelinepb.ListPipelineRunsByRequesterResponse, error) {
	resp, err := h.service.ListPipelineRunsByRequester(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, "Failed to list pipeline runs")
	}
	return resp, nil
}
