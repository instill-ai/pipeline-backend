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

// ListPipelines returns a paginated list of pipelines.
func (h *PublicHandler) ListPipelines(ctx context.Context, req *pipelinepb.ListPipelinesRequest) (*pipelinepb.ListPipelinesResponse, error) {

	if err := authenticateUser(ctx, true); err != nil {
		return &pipelinepb.ListPipelinesResponse{}, err
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
		return &pipelinepb.ListPipelinesResponse{}, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return &pipelinepb.ListPipelinesResponse{}, err
	}

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return &pipelinepb.ListPipelinesResponse{}, err
	}

	pbPipelines, totalSize, nextPageToken, err := h.service.ListPipelines(
		ctx, req.GetPageSize(), req.GetPageToken(), req.GetView(), req.Visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		return &pipelinepb.ListPipelinesResponse{}, err
	}

	resp := pipelinepb.ListPipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     int32(totalSize),
	}

	return &resp, nil
}

// CreateNamespacePipeline creates a new pipeline for a namespace.
func (h *PublicHandler) CreateNamespacePipeline(ctx context.Context, req *pipelinepb.CreateNamespacePipelineRequest) (resp *pipelinepb.CreateNamespacePipelineResponse, err error) {

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

	pipeline, err := h.service.CreateNamespacePipeline(ctx, ns, pipelineToCreate)

	if err != nil {
		return nil, err
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &pipelinepb.CreateNamespacePipelineResponse{Pipeline: pipeline}, nil
}

// ListNamespacePipelines returns a paginated list of pipelines for a namespace.
func (h *PublicHandler) ListNamespacePipelines(ctx context.Context, req *pipelinepb.ListNamespacePipelinesRequest) (resp *pipelinepb.ListNamespacePipelinesResponse, err error) {

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

	pbPipelines, totalSize, nextPageToken, err := h.service.ListNamespacePipelines(ctx, ns, req.GetPageSize(), req.GetPageToken(), req.GetView(), &visibility, filter, req.GetShowDeleted(), orderBy)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ListNamespacePipelinesResponse{
		Pipelines:     pbPipelines,
		NextPageToken: nextPageToken,
		TotalSize:     totalSize,
	}, nil
}

// GetNamespacePipeline returns the details of a pipeline for a namespace.
func (h *PublicHandler) GetNamespacePipeline(ctx context.Context, req *pipelinepb.GetNamespacePipelineRequest) (*pipelinepb.GetNamespacePipelineResponse, error) {

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

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, req.GetView())

	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

// UpdateNamespacePipeline updates a pipeline for a namespace.
func (h *PublicHandler) UpdateNamespacePipeline(ctx context.Context, req *pipelinepb.UpdateNamespacePipelineRequest) (*pipelinepb.UpdateNamespacePipelineResponse, error) {

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

	getResp, err := h.GetNamespacePipeline(ctx, &pipelinepb.GetNamespacePipelineRequest{Name: req.GetPipeline().GetName(), View: pipelinepb.Pipeline_VIEW_RECIPE.Enum()})
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
		return &pipelinepb.UpdateNamespacePipelineResponse{Pipeline: getResp.GetPipeline()}, nil
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

	pbPipeline, err := h.service.UpdateNamespacePipelineByID(ctx, ns, pipelineID, pbPipelineToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

// DeleteNamespacePipeline deletes a pipeline for a namespace.
func (h *PublicHandler) DeleteNamespacePipeline(ctx context.Context, req *pipelinepb.DeleteNamespacePipelineRequest) (*pipelinepb.DeleteNamespacePipelineResponse, error) {

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
	_, err = h.GetNamespacePipeline(ctx, &pipelinepb.GetNamespacePipelineRequest{Name: req.GetName()})
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineByID(ctx, ns, pipelineID); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteNamespacePipelineResponse{}, nil
}

// ValidateNamespacePipeline validates a pipeline for a namespace.
func (h *PublicHandler) ValidateNamespacePipeline(ctx context.Context, req *pipelinepb.ValidateNamespacePipelineRequest) (*pipelinepb.ValidateNamespacePipelineResponse, error) {

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

	validationErrors, err := h.service.ValidateNamespacePipelineByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	return &pipelinepb.ValidateNamespacePipelineResponse{Errors: validationErrors, Success: len(validationErrors) == 0}, nil
}

// RenameNamespacePipeline renames a pipeline for a namespace.
func (h *PublicHandler) RenameNamespacePipeline(ctx context.Context, req *pipelinepb.RenameNamespacePipelineRequest) (*pipelinepb.RenameNamespacePipelineResponse, error) {

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

	pbPipeline, err := h.service.UpdateNamespacePipelineIDByID(ctx, ns, pipelineID, newID)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.RenameNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

// CloneNamespacePipeline clones a pipeline for a namespace.
func (h *PublicHandler) CloneNamespacePipeline(ctx context.Context, req *pipelinepb.CloneNamespacePipelineRequest) (*pipelinepb.CloneNamespacePipelineResponse, error) {

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

	_, err = h.service.CloneNamespacePipeline(
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

	return &pipelinepb.CloneNamespacePipelineResponse{}, nil
}

// CloneNamespacePipelineRelease clones a pipeline release for a namespace.
func (h *PublicHandler) CloneNamespacePipelineRelease(ctx context.Context, req *pipelinepb.CloneNamespacePipelineReleaseRequest) (*pipelinepb.CloneNamespacePipelineReleaseResponse, error) {

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
	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	_, err = h.service.CloneNamespacePipelineRelease(
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

	return &pipelinepb.CloneNamespacePipelineReleaseResponse{}, nil
}

// preTriggerNamespacePipeline is a helper function to pre-trigger a namespace pipeline.
func (h *PublicHandler) preTriggerNamespacePipeline(ctx context.Context, req TriggerPipelineRequestInterface) (resource.Namespace, string, *pipelinepb.Pipeline, bool, error) {

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

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, pipelineID, nil, false, err
	}
	// _, err = h.service.ValidateNamespacePipelineByID(ctx, ns,  id)
	// if err != nil {
	// 	return ns, nil, id, nil, false, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }
	returnTraces := resourcex.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true"

	return ns, pipelineID, pbPipeline, returnTraces, nil

}

// TriggerNamespacePipeline triggers a pipeline for a namespace.
func (h *PublicHandler) TriggerNamespacePipeline(ctx context.Context, req *pipelinepb.TriggerNamespacePipelineRequest) (resp *pipelinepb.TriggerNamespacePipelineResponse, err error) {

	ns, id, _, returnTraces, err := h.preTriggerNamespacePipeline(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	outputs, metadata, err := h.service.TriggerNamespacePipelineByID(ctx, ns, id, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	// TODO: it would be useful to return the trigger UID here.
	return &pipelinepb.TriggerNamespacePipelineResponse{Outputs: outputs, Metadata: metadata}, nil
}

// TriggerAsyncNamespacePipeline triggers an async pipeline for a namespace.
func (h *PublicHandler) TriggerAsyncNamespacePipeline(ctx context.Context, req *pipelinepb.TriggerAsyncNamespacePipelineRequest) (resp *pipelinepb.TriggerAsyncNamespacePipelineResponse, err error) {

	ns, id, _, returnTraces, err := h.preTriggerNamespacePipeline(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	operation, err := h.service.TriggerAsyncNamespacePipelineByID(ctx, ns, id, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerAsyncNamespacePipelineResponse{Operation: operation}, nil
}

// CreateNamespacePipelineRelease creates a pipeline release for a namespace.
func (h *PublicHandler) CreateNamespacePipelineRelease(ctx context.Context, req *pipelinepb.CreateNamespacePipelineReleaseRequest) (*pipelinepb.CreateNamespacePipelineReleaseResponse, error) {

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

	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	// TODO: We temporarily removed the release validation due to a malfunction
	// in the validation function. We'll add it back after we fix the validation
	// function.
	// _, err = h.service.ValidateNamespacePipelineByID(ctx, ns, pipelineID)
	// if err != nil {
	// 	return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }

	pbPipelineRelease, err := h.service.CreateNamespacePipelineRelease(ctx, ns, pipelineUID, req.GetRelease())
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

	return &pipelinepb.CreateNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

// ListNamespacePipelineReleases lists pipeline releases for a namespace.
func (h *PublicHandler) ListNamespacePipelineReleases(ctx context.Context, req *pipelinepb.ListNamespacePipelineReleasesRequest) (resp *pipelinepb.ListNamespacePipelineReleasesResponse, err error) {

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

	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListNamespacePipelineReleases(ctx, ns, pipelineUID, req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ListNamespacePipelineReleasesResponse{
		Releases:      pbPipelineReleases,
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil

}

// GetNamespacePipelineRelease gets a pipeline release for a namespace.
func (h *PublicHandler) GetNamespacePipelineRelease(ctx context.Context, req *pipelinepb.GetNamespacePipelineReleaseRequest) (resp *pipelinepb.GetNamespacePipelineReleaseResponse, err error) {

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

	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, pipelineUID, releaseID, req.GetView())
	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

// UpdateNamespacePipelineRelease updates a pipeline release for a namespace.
func (h *PublicHandler) UpdateNamespacePipelineRelease(ctx context.Context, req *pipelinepb.UpdateNamespacePipelineReleaseRequest) (resp *pipelinepb.UpdateNamespacePipelineReleaseResponse, err error) {

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

	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	getResp, err := h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{Name: req.GetRelease().GetName(), View: pipelinepb.Pipeline_VIEW_FULL.Enum()})
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
		return &pipelinepb.UpdateNamespacePipelineReleaseResponse{Release: getResp.GetRelease()}, nil
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

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseByID(ctx, ns, pipelineUID, releaseID, pbPipelineReleaseToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil
}

// RenameNamespacePipelineReleaseRequestInterface is the interface for the request to rename a pipeline release.
// DeleteNamespacePipelineRelease deletes a pipeline release for a namespace.
func (h *PublicHandler) DeleteNamespacePipelineRelease(ctx context.Context, req *pipelinepb.DeleteNamespacePipelineReleaseRequest) (*pipelinepb.DeleteNamespacePipelineReleaseResponse, error) {

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

	_, err = h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{
		Name: req.GetName(),
	})
	if err != nil {
		return nil, err
	}

	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineReleaseByID(ctx, ns, pipelineUID, releaseID); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteNamespacePipelineReleaseResponse{}, nil
}

func (h *PublicHandler) preTriggerNamespacePipelineRelease(ctx context.Context, req TriggerPipelineReleaseRequestInterface) (resource.Namespace, string, uuid.UUID, *pipelinepb.PipelineRelease, bool, error) {

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

	pipelineUID, err := h.service.GetNamespacePipelineUIDByID(ctx, ns, pipelineID)
	if err != nil {
		return ns, "", uuid.Nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, pipelineUID, releaseID, pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", uuid.Nil, nil, false, err
	}
	returnTraces := resourcex.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true"

	return ns, releaseID, pipelineUID, pbPipelineRelease, returnTraces, nil

}

// TriggerNamespacePipelineRelease triggers a pipeline release for a namespace.
func (h *PublicHandler) TriggerNamespacePipelineRelease(ctx context.Context, req *pipelinepb.TriggerNamespacePipelineReleaseRequest) (resp *pipelinepb.TriggerNamespacePipelineReleaseResponse, err error) {

	ns, releaseID, pipelineUID, _, returnTraces, err := h.preTriggerNamespacePipelineRelease(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	outputs, metadata, err := h.service.TriggerNamespacePipelineReleaseByID(ctx, ns, pipelineUID, releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerNamespacePipelineReleaseResponse{Outputs: outputs, Metadata: metadata}, nil
}

// TriggerAsyncNamespacePipelineRelease triggers an async pipeline release for a namespace.
func (h *PublicHandler) TriggerAsyncNamespacePipelineRelease(ctx context.Context, req *pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest) (resp *pipelinepb.TriggerAsyncNamespacePipelineReleaseResponse, err error) {

	ns, releaseID, pipelineUID, _, returnTraces, err := h.preTriggerNamespacePipelineRelease(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	operation, err := h.service.TriggerAsyncNamespacePipelineReleaseByID(ctx, ns, pipelineUID, releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerAsyncNamespacePipelineReleaseResponse{Operation: operation}, nil
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
