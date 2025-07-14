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

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	errorsx "github.com/instill-ai/x/errors"
	resourcex "github.com/instill-ai/x/resource"
)

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

// CreateUserPipeline creates a new pipeline for a user.
func (h *PublicHandler) CreateUserPipeline(ctx context.Context, req *pipelinepb.CreateUserPipelineRequest) (resp *pipelinepb.CreateUserPipelineResponse, err error) {
	r, err := h.CreateNamespacePipeline(ctx, &pipelinepb.CreateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Pipeline:    req.Pipeline,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.CreateUserPipelineResponse{Pipeline: r.Pipeline}, nil
}

// CreateOrganizationPipeline creates a new pipeline for an organization.
func (h *PublicHandler) CreateOrganizationPipeline(ctx context.Context, req *pipelinepb.CreateOrganizationPipelineRequest) (resp *pipelinepb.CreateOrganizationPipelineResponse, err error) {
	r, err := h.CreateNamespacePipeline(ctx, &pipelinepb.CreateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		Pipeline:    req.Pipeline,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.CreateOrganizationPipelineResponse{Pipeline: r.Pipeline}, nil
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

	// Return error if resource ID does not follow RFC-1034
	if err := checkfield.CheckResourceID(req.GetPipeline().GetId()); err != nil {
		return nil, fmt.Errorf("%w: invalid secret ID: %w", errorsx.ErrInvalidArgument, err)
	}

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)

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

// ListUserPipelines returns a paginated list of pipelines for a user.
func (h *PublicHandler) ListUserPipelines(ctx context.Context, req *pipelinepb.ListUserPipelinesRequest) (resp *pipelinepb.ListUserPipelinesResponse, err error) {
	r, err := h.ListNamespacePipelines(ctx, &pipelinepb.ListNamespacePipelinesRequest{
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
	return &pipelinepb.ListUserPipelinesResponse{
		Pipelines:     r.Pipelines,
		NextPageToken: r.NextPageToken,
		TotalSize:     r.TotalSize,
	}, nil
}

// ListOrganizationPipelines returns a paginated list of pipelines for an organization.
func (h *PublicHandler) ListOrganizationPipelines(ctx context.Context, req *pipelinepb.ListOrganizationPipelinesRequest) (resp *pipelinepb.ListOrganizationPipelinesResponse, err error) {
	r, err := h.ListNamespacePipelines(ctx, &pipelinepb.ListNamespacePipelinesRequest{
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
	return &pipelinepb.ListOrganizationPipelinesResponse{
		Pipelines:     r.Pipelines,
		NextPageToken: r.NextPageToken,
		TotalSize:     r.TotalSize,
	}, nil
}

// ListNamespacePipelines returns a paginated list of pipelines for a namespace.
func (h *PublicHandler) ListNamespacePipelines(ctx context.Context, req *pipelinepb.ListNamespacePipelinesRequest) (resp *pipelinepb.ListNamespacePipelinesResponse, err error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
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

// GetUserPipeline returns the details of a pipeline for a user.
func (h *PublicHandler) GetUserPipeline(ctx context.Context, req *pipelinepb.GetUserPipelineRequest) (resp *pipelinepb.GetUserPipelineResponse, err error) {
	r, err := h.GetNamespacePipeline(ctx, &pipelinepb.GetNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.GetUserPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

// GetOrganizationPipeline returns the details of a pipeline for an organization.
func (h *PublicHandler) GetOrganizationPipeline(ctx context.Context, req *pipelinepb.GetOrganizationPipelineRequest) (resp *pipelinepb.GetOrganizationPipelineResponse, err error) {
	r, err := h.GetNamespacePipeline(ctx, &pipelinepb.GetNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.GetOrganizationPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

// GetNamespacePipeline returns the details of a pipeline for a namespace.
func (h *PublicHandler) GetNamespacePipeline(ctx context.Context, req *pipelinepb.GetNamespacePipelineRequest) (*pipelinepb.GetNamespacePipelineResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, req.GetView())

	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

// UpdateUserPipeline updates a pipeline for a user.
func (h *PublicHandler) UpdateUserPipeline(ctx context.Context, req *pipelinepb.UpdateUserPipelineRequest) (resp *pipelinepb.UpdateUserPipelineResponse, err error) {
	r, err := h.UpdateNamespacePipeline(ctx, &pipelinepb.UpdateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Pipeline.Name, "/")[1],
		PipelineId:  strings.Split(req.Pipeline.Name, "/")[3],
		Pipeline:    req.Pipeline,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.UpdateUserPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

// UpdateOrganizationPipeline updates a pipeline for an organization.
func (h *PublicHandler) UpdateOrganizationPipeline(ctx context.Context, req *pipelinepb.UpdateOrganizationPipelineRequest) (resp *pipelinepb.UpdateOrganizationPipelineResponse, err error) {
	r, err := h.UpdateNamespacePipeline(ctx, &pipelinepb.UpdateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Pipeline.Name, "/")[1],
		PipelineId:  strings.Split(req.Pipeline.Name, "/")[3],
		Pipeline:    req.Pipeline,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.UpdateOrganizationPipelineResponse{
		Pipeline: r.Pipeline,
	}, nil
}

// UpdateNamespacePipeline updates a pipeline for a namespace.
func (h *PublicHandler) UpdateNamespacePipeline(ctx context.Context, req *pipelinepb.UpdateNamespacePipelineRequest) (*pipelinepb.UpdateNamespacePipelineResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
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
		return nil, errorsx.ErrUpdateMask
	}

	getResp, err := h.GetNamespacePipeline(ctx, &pipelinepb.GetNamespacePipelineRequest{NamespaceId: req.NamespaceId, PipelineId: req.PipelineId, View: pipelinepb.Pipeline_VIEW_RECIPE.Enum()})
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

	pbPipeline, err := h.service.UpdateNamespacePipelineByID(ctx, ns, req.PipelineId, pbPipelineToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

// DeleteUserPipeline deletes a pipeline for a user.
func (h *PublicHandler) DeleteUserPipeline(ctx context.Context, req *pipelinepb.DeleteUserPipelineRequest) (resp *pipelinepb.DeleteUserPipelineResponse, err error) {
	_, err = h.DeleteNamespacePipeline(ctx, &pipelinepb.DeleteNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.DeleteUserPipelineResponse{}, nil
}

// DeleteOrganizationPipeline deletes a pipeline for an organization.
func (h *PublicHandler) DeleteOrganizationPipeline(ctx context.Context, req *pipelinepb.DeleteOrganizationPipelineRequest) (resp *pipelinepb.DeleteOrganizationPipelineResponse, err error) {
	_, err = h.DeleteNamespacePipeline(ctx, &pipelinepb.DeleteNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.DeleteOrganizationPipelineResponse{}, nil
}

// DeleteNamespacePipeline deletes a pipeline for a namespace.
func (h *PublicHandler) DeleteNamespacePipeline(ctx context.Context, req *pipelinepb.DeleteNamespacePipelineRequest) (*pipelinepb.DeleteNamespacePipelineResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	_, err = h.GetNamespacePipeline(ctx, &pipelinepb.GetNamespacePipelineRequest{NamespaceId: req.NamespaceId, PipelineId: req.PipelineId})
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineByID(ctx, ns, req.PipelineId); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteNamespacePipelineResponse{}, nil
}

// LookUpPipeline returns the details of a pipeline.
func (h *PublicHandler) LookUpPipeline(ctx context.Context, req *pipelinepb.LookUpPipelineRequest) (*pipelinepb.LookUpPipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, lookUpPipelineRequiredFields); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	uid, err := resource.GetRscPermalinkUID(req.Permalink)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pbPipeline, err := h.service.GetPipelineByUID(ctx, uid, req.GetView())
	if err != nil {
		return nil, err
	}

	resp := pipelinepb.LookUpPipelineResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

// ValidateUserPipeline validates a pipeline for a user.
func (h *PublicHandler) ValidateUserPipeline(ctx context.Context, req *pipelinepb.ValidateUserPipelineRequest) (resp *pipelinepb.ValidateUserPipelineResponse, err error) {
	r, err := h.ValidateNamespacePipeline(ctx, &pipelinepb.ValidateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.ValidateUserPipelineResponse{Errors: r.Errors, Success: r.Success}, nil
}

// ValidateOrganizationPipeline validates a pipeline for an organization.
func (h *PublicHandler) ValidateOrganizationPipeline(ctx context.Context, req *pipelinepb.ValidateOrganizationPipelineRequest) (resp *pipelinepb.ValidateOrganizationPipelineResponse, err error) {
	r, err := h.ValidateNamespacePipeline(ctx, &pipelinepb.ValidateNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.ValidateOrganizationPipelineResponse{Errors: r.Errors, Success: r.Success}, nil
}

// ValidateNamespacePipeline validates a pipeline for a namespace.
func (h *PublicHandler) ValidateNamespacePipeline(ctx context.Context, req *pipelinepb.ValidateNamespacePipelineRequest) (*pipelinepb.ValidateNamespacePipelineResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	validationErrors, err := h.service.ValidateNamespacePipelineByID(ctx, ns, req.PipelineId)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	}

	return &pipelinepb.ValidateNamespacePipelineResponse{Errors: validationErrors, Success: len(validationErrors) == 0}, nil
}

// RenameUserPipeline renames a pipeline for a user.
func (h *PublicHandler) RenameUserPipeline(ctx context.Context, req *pipelinepb.RenameUserPipelineRequest) (resp *pipelinepb.RenameUserPipelineResponse, err error) {
	r, err := h.RenameNamespacePipeline(ctx, &pipelinepb.RenameNamespacePipelineRequest{
		NamespaceId:   strings.Split(req.Name, "/")[1],
		PipelineId:    strings.Split(req.Name, "/")[3],
		NewPipelineId: req.NewPipelineId,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.RenameUserPipelineResponse{Pipeline: r.Pipeline}, nil
}

// RenameOrganizationPipeline renames a pipeline for an organization.
func (h *PublicHandler) RenameOrganizationPipeline(ctx context.Context, req *pipelinepb.RenameOrganizationPipelineRequest) (resp *pipelinepb.RenameOrganizationPipelineResponse, err error) {
	r, err := h.RenameNamespacePipeline(ctx, &pipelinepb.RenameNamespacePipelineRequest{
		NamespaceId:   strings.Split(req.Name, "/")[1],
		PipelineId:    strings.Split(req.Name, "/")[3],
		NewPipelineId: req.NewPipelineId,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.RenameOrganizationPipelineResponse{Pipeline: r.Pipeline}, nil
}

// RenameNamespacePipeline renames a pipeline for a namespace.
func (h *PublicHandler) RenameNamespacePipeline(ctx context.Context, req *pipelinepb.RenameNamespacePipelineRequest) (*pipelinepb.RenameNamespacePipelineResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, renamePipelineRequiredFields); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
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

	pbPipeline, err := h.service.UpdateNamespacePipelineIDByID(ctx, ns, req.PipelineId, newID)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.RenameNamespacePipelineResponse{Pipeline: pbPipeline}, nil
}

// CloneNamespacePipeline clones a pipeline for a namespace.
func (h *PublicHandler) CloneNamespacePipeline(ctx context.Context, req *pipelinepb.CloneNamespacePipelineRequest) (*pipelinepb.CloneNamespacePipelineResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	_, err = h.service.CloneNamespacePipeline(
		ctx,
		ns,
		req.PipelineId,
		req.GetTargetNamespaceId(),
		req.GetTargetPipelineId(),
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

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}
	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	_, err = h.service.CloneNamespacePipelineRelease(
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

	id := req.GetPipelineId()
	ns, err := h.service.GetNamespaceByID(ctx, req.GetNamespaceId())
	if err != nil {
		return ns, id, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, id, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.GetPipelineId(), pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, id, nil, false, err
	}
	// _, err = h.service.ValidateNamespacePipelineByID(ctx, ns,  id)
	// if err != nil {
	// 	return ns, nil, id, nil, false, status.Error(codes.FailedPrecondition, fmt.Sprintf("[Pipeline Recipe Error] %+v", err.Error()))
	// }
	returnTraces := resourcex.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true"

	return ns, id, pbPipeline, returnTraces, nil

}

// TriggerUserPipeline triggers a pipeline for a user.
func (h *PublicHandler) TriggerUserPipeline(ctx context.Context, req *pipelinepb.TriggerUserPipelineRequest) (resp *pipelinepb.TriggerUserPipelineResponse, err error) {
	r, err := h.TriggerNamespacePipeline(ctx, &pipelinepb.TriggerNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerUserPipelineResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

// TriggerOrganizationPipeline triggers a pipeline for an organization.
func (h *PublicHandler) TriggerOrganizationPipeline(ctx context.Context, req *pipelinepb.TriggerOrganizationPipelineRequest) (resp *pipelinepb.TriggerOrganizationPipelineResponse, err error) {
	r, err := h.TriggerNamespacePipeline(ctx, &pipelinepb.TriggerNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerOrganizationPipelineResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
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

// TriggerAsyncUserPipeline triggers an async pipeline for a user.
func (h *PublicHandler) TriggerAsyncUserPipeline(ctx context.Context, req *pipelinepb.TriggerAsyncUserPipelineRequest) (resp *pipelinepb.TriggerAsyncUserPipelineResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipeline(ctx, &pipelinepb.TriggerAsyncNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerAsyncUserPipelineResponse{Operation: r.Operation}, nil
}

// TriggerAsyncOrganizationPipeline triggers an async pipeline for an organization.
func (h *PublicHandler) TriggerAsyncOrganizationPipeline(ctx context.Context, req *pipelinepb.TriggerAsyncOrganizationPipelineRequest) (resp *pipelinepb.TriggerAsyncOrganizationPipelineResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipeline(ctx, &pipelinepb.TriggerAsyncNamespacePipelineRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerAsyncOrganizationPipelineResponse{Operation: r.Operation}, nil
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

// CreateUserPipelineRelease creates a pipeline release for a user.
func (h *PublicHandler) CreateUserPipelineRelease(ctx context.Context, req *pipelinepb.CreateUserPipelineReleaseRequest) (resp *pipelinepb.CreateUserPipelineReleaseResponse, err error) {
	r, err := h.CreateNamespacePipelineRelease(ctx, &pipelinepb.CreateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PipelineId:  strings.Split(req.Parent, "/")[3],
		Release:     req.Release,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.CreateUserPipelineReleaseResponse{Release: r.Release}, nil
}

// CreateOrganizationPipelineRelease creates a pipeline release for an organization.
func (h *PublicHandler) CreateOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.CreateOrganizationPipelineReleaseRequest) (resp *pipelinepb.CreateOrganizationPipelineReleaseResponse, err error) {
	r, err := h.CreateNamespacePipelineRelease(ctx, &pipelinepb.CreateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Parent, "/")[1],
		PipelineId:  strings.Split(req.Parent, "/")[3],
		Release:     req.Release,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.CreateOrganizationPipelineReleaseResponse{Release: r.Release}, nil
}

// CreateNamespacePipelineRelease creates a pipeline release for a namespace.
func (h *PublicHandler) CreateNamespacePipelineRelease(ctx context.Context, req *pipelinepb.CreateNamespacePipelineReleaseRequest) (*pipelinepb.CreateNamespacePipelineReleaseResponse, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req.GetRelease(), append(releaseCreateRequiredFields, immutablePipelineFields...)); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	// Set all OUTPUT_ONLY fields to zero value on the requested payload pipeline resource
	if err := checkfield.CheckCreateOutputOnlyFields(req.GetRelease(), releaseOutputOnlyFields); err != nil {
		return nil, errorsx.ErrCheckOutputOnlyFields
	}

	// Return error if resource ID does not a semantic version
	if !semver.IsValid(req.GetRelease().GetId()) {
		return nil, errorsx.ErrSematicVersion
	}

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pipelinepb.Pipeline_VIEW_BASIC)
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

// ListUserPipelineReleases lists pipeline releases for a user.
func (h *PublicHandler) ListUserPipelineReleases(ctx context.Context, req *pipelinepb.ListUserPipelineReleasesRequest) (resp *pipelinepb.ListUserPipelineReleasesResponse, err error) {
	r, err := h.ListNamespacePipelineReleases(ctx, &pipelinepb.ListNamespacePipelineReleasesRequest{
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
	return &pipelinepb.ListUserPipelineReleasesResponse{Releases: r.Releases, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

// ListOrganizationPipelineReleases lists pipeline releases for an organization.
func (h *PublicHandler) ListOrganizationPipelineReleases(ctx context.Context, req *pipelinepb.ListOrganizationPipelineReleasesRequest) (resp *pipelinepb.ListOrganizationPipelineReleasesResponse, err error) {
	r, err := h.ListNamespacePipelineReleases(ctx, &pipelinepb.ListNamespacePipelineReleasesRequest{
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
	return &pipelinepb.ListOrganizationPipelineReleasesResponse{Releases: r.Releases, NextPageToken: r.NextPageToken, TotalSize: r.TotalSize}, nil
}

// ListNamespacePipelineReleases lists pipeline releases for a namespace.
func (h *PublicHandler) ListNamespacePipelineReleases(ctx context.Context, req *pipelinepb.ListNamespacePipelineReleasesRequest) (resp *pipelinepb.ListNamespacePipelineReleasesResponse, err error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
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

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineReleases, totalSize, nextPageToken, err := h.service.ListNamespacePipelineReleases(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.GetPageSize(), req.GetPageToken(), req.GetView(), filter, req.GetShowDeleted())
	if err != nil {
		return nil, err
	}

	return &pipelinepb.ListNamespacePipelineReleasesResponse{
		Releases:      pbPipelineReleases,
		TotalSize:     totalSize,
		NextPageToken: nextPageToken,
	}, nil

}

// GetUserPipelineRelease gets a pipeline release for a user.
func (h *PublicHandler) GetUserPipelineRelease(ctx context.Context, req *pipelinepb.GetUserPipelineReleaseRequest) (resp *pipelinepb.GetUserPipelineReleaseResponse, err error) {
	r, err := h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.GetUserPipelineReleaseResponse{Release: r.Release}, nil
}

// GetOrganizationPipelineRelease gets a pipeline release for an organization.
func (h *PublicHandler) GetOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.GetOrganizationPipelineReleaseRequest) (resp *pipelinepb.GetOrganizationPipelineReleaseResponse, err error) {
	r, err := h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		View:        req.View,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.GetOrganizationPipelineReleaseResponse{Release: r.Release}, nil
}

// GetNamespacePipelineRelease gets a pipeline release for a namespace.
func (h *PublicHandler) GetNamespacePipelineRelease(ctx context.Context, req *pipelinepb.GetNamespacePipelineReleaseRequest) (resp *pipelinepb.GetNamespacePipelineReleaseResponse, err error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, true); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.ReleaseId, req.GetView())
	if err != nil {
		return nil, err
	}

	return &pipelinepb.GetNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil

}

// UpdateUserPipelineRelease updates a pipeline release for a user.
func (h *PublicHandler) UpdateUserPipelineRelease(ctx context.Context, req *pipelinepb.UpdateUserPipelineReleaseRequest) (resp *pipelinepb.UpdateUserPipelineReleaseResponse, err error) {
	r, err := h.UpdateNamespacePipelineRelease(ctx, &pipelinepb.UpdateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Release.Name, "/")[1],
		PipelineId:  strings.Split(req.Release.Name, "/")[3],
		ReleaseId:   strings.Split(req.Release.Name, "/")[5],
		Release:     req.Release,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.UpdateUserPipelineReleaseResponse{Release: r.Release}, nil
}

// UpdateOrganizationPipelineRelease updates a pipeline release for an organization.
func (h *PublicHandler) UpdateOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.UpdateOrganizationPipelineReleaseRequest) (resp *pipelinepb.UpdateOrganizationPipelineReleaseResponse, err error) {
	r, err := h.UpdateNamespacePipelineRelease(ctx, &pipelinepb.UpdateNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Release.Name, "/")[1],
		PipelineId:  strings.Split(req.Release.Name, "/")[3],
		ReleaseId:   strings.Split(req.Release.Name, "/")[5],
		Release:     req.Release,
		UpdateMask:  req.UpdateMask,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.UpdateOrganizationPipelineReleaseResponse{Release: r.Release}, nil
}

// UpdateNamespacePipelineRelease updates a pipeline release for a namespace.
func (h *PublicHandler) UpdateNamespacePipelineRelease(ctx context.Context, req *pipelinepb.UpdateNamespacePipelineReleaseRequest) (resp *pipelinepb.UpdateNamespacePipelineReleaseResponse, err error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
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
		return nil, errorsx.ErrUpdateMask
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	getResp, err := h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{NamespaceId: req.NamespaceId, PipelineId: req.PipelineId, ReleaseId: req.ReleaseId, View: pipelinepb.Pipeline_VIEW_FULL.Enum()})
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

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.ReleaseId, pbPipelineReleaseToUpdate)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.UpdateNamespacePipelineReleaseResponse{Release: pbPipelineRelease}, nil
}

// RenameNamespacePipelineReleaseRequestInterface is the interface for the request to rename a pipeline release.
type RenameNamespacePipelineReleaseRequestInterface interface {
	GetName() string
	GetNewPipelineReleaseId() string
}

// RenameUserPipelineRelease renames a pipeline release for a user.
func (h *PublicHandler) RenameUserPipelineRelease(ctx context.Context, req *pipelinepb.RenameUserPipelineReleaseRequest) (resp *pipelinepb.RenameUserPipelineReleaseResponse, err error) {
	resp = &pipelinepb.RenameUserPipelineReleaseResponse{}
	resp.Release, err = h.renameNamespacePipelineRelease(ctx, req)
	return resp, err
}

// RenameOrganizationPipelineRelease renames a pipeline release for an organization.
func (h *PublicHandler) RenameOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.RenameOrganizationPipelineReleaseRequest) (resp *pipelinepb.RenameOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinepb.RenameOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.renameNamespacePipelineRelease(ctx, req)
	return resp, err
}

func (h *PublicHandler) renameNamespacePipelineRelease(ctx context.Context, req RenameNamespacePipelineReleaseRequestInterface) (release *pipelinepb.PipelineRelease, err error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, releaseRenameRequiredFields); err != nil {
		return nil, errorsx.ErrCheckRequiredFields
	}

	splits := strings.Split(req.GetName(), "/")
	namespaceID := splits[1]
	pipelineID := splits[3]
	releaseID := splits[5]
	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	newID := req.GetNewPipelineReleaseId()
	// Return error if resource ID does not a semantic version
	if !semver.IsValid(newID) {
		return nil, errorsx.ErrSematicVersion
	}

	pbPipelineRelease, err := h.service.UpdateNamespacePipelineReleaseIDByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, newID)
	if err != nil {
		return nil, err
	}

	return pbPipelineRelease, nil
}

// DeleteUserPipelineRelease deletes a pipeline release for a user.
func (h *PublicHandler) DeleteUserPipelineRelease(ctx context.Context, req *pipelinepb.DeleteUserPipelineReleaseRequest) (resp *pipelinepb.DeleteUserPipelineReleaseResponse, err error) {
	_, err = h.DeleteNamespacePipelineRelease(ctx, &pipelinepb.DeleteNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.DeleteUserPipelineReleaseResponse{}, nil
}

// DeleteOrganizationPipelineRelease deletes a pipeline release for an organization.
func (h *PublicHandler) DeleteOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.DeleteOrganizationPipelineReleaseRequest) (resp *pipelinepb.DeleteOrganizationPipelineReleaseResponse, err error) {
	_, err = h.DeleteNamespacePipelineRelease(ctx, &pipelinepb.DeleteNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.DeleteOrganizationPipelineReleaseResponse{}, nil
}

// DeleteNamespacePipelineRelease deletes a pipeline release for a namespace.
func (h *PublicHandler) DeleteNamespacePipelineRelease(ctx context.Context, req *pipelinepb.DeleteNamespacePipelineReleaseRequest) (*pipelinepb.DeleteNamespacePipelineReleaseResponse, error) {

	ns, err := h.service.GetNamespaceByID(ctx, req.NamespaceId)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	_, err = h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{
		NamespaceId: req.NamespaceId,
		PipelineId:  req.PipelineId,
		ReleaseId:   req.ReleaseId,
	})
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.PipelineId, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.DeleteNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), req.ReleaseId); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return nil, err
	}

	return &pipelinepb.DeleteNamespacePipelineReleaseResponse{}, nil
}

// RestoreNamespacePipelineReleaseRequestInterface is the interface for the request to restore a pipeline release.
type RestoreNamespacePipelineReleaseRequestInterface interface {
	GetName() string
}

// RestoreUserPipelineRelease restores a pipeline release for a user.
func (h *PublicHandler) RestoreUserPipelineRelease(ctx context.Context, req *pipelinepb.RestoreUserPipelineReleaseRequest) (resp *pipelinepb.RestoreUserPipelineReleaseResponse, err error) {
	resp = &pipelinepb.RestoreUserPipelineReleaseResponse{}
	resp.Release, err = h.restoreNamespacePipelineRelease(ctx, req)
	return resp, err
}

// RestoreOrganizationPipelineRelease restores a pipeline release for an organization.
func (h *PublicHandler) RestoreOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.RestoreOrganizationPipelineReleaseRequest) (resp *pipelinepb.RestoreOrganizationPipelineReleaseResponse, err error) {
	resp = &pipelinepb.RestoreOrganizationPipelineReleaseResponse{}
	resp.Release, err = h.restoreNamespacePipelineRelease(ctx, req)
	return resp, err
}

// RestoreNamespacePipelineRelease restores a pipeline release for a namespace.
func (h *PublicHandler) restoreNamespacePipelineRelease(ctx context.Context, req RestoreNamespacePipelineReleaseRequestInterface) (release *pipelinepb.PipelineRelease, err error) {

	splits := strings.Split(req.GetName(), "/")
	namespaceID := splits[1]
	pipelineID := splits[3]
	releaseID := splits[5]
	ns, err := h.service.GetNamespaceByID(ctx, namespaceID)
	if err != nil {
		return nil, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return nil, err
	}

	_, err = h.GetNamespacePipelineRelease(ctx, &pipelinepb.GetNamespacePipelineReleaseRequest{
		NamespaceId: namespaceID,
		PipelineId:  pipelineID,
		ReleaseId:   releaseID,
	})
	if err != nil {
		return nil, err
	}

	pipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, pipelineID, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	if err := h.service.RestoreNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID); err != nil {
		return nil, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pipeline.Uid), releaseID, pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return nil, err
	}

	return pbPipelineRelease, nil
}

func (h *PublicHandler) preTriggerNamespacePipelineRelease(ctx context.Context, req TriggerPipelineReleaseRequestInterface) (resource.Namespace, string, *pipelinepb.Pipeline, *pipelinepb.PipelineRelease, bool, error) {

	// Return error if REQUIRED fields are not provided in the requested payload pipeline resource
	if err := checkfield.CheckRequiredFields(req, triggerPipelineRequiredFields); err != nil {
		return resource.Namespace{}, "", nil, nil, false, errorsx.ErrCheckRequiredFields
	}

	ns, err := h.service.GetNamespaceByID(ctx, req.GetNamespaceId())
	if err != nil {
		return ns, "", nil, nil, false, err
	}
	if err := authenticateUser(ctx, false); err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipeline, err := h.service.GetNamespacePipelineByID(ctx, ns, req.GetPipelineId(), pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", nil, nil, false, err
	}

	pbPipelineRelease, err := h.service.GetNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), req.GetReleaseId(), pipelinepb.Pipeline_VIEW_FULL)
	if err != nil {
		return ns, "", nil, nil, false, err
	}
	returnTraces := resourcex.GetRequestSingleHeader(ctx, constant.HeaderReturnTracesKey) == "true"

	return ns, req.GetReleaseId(), pbPipeline, pbPipelineRelease, returnTraces, nil

}

// TriggerUserPipelineRelease triggers a pipeline release for a user.
func (h *PublicHandler) TriggerUserPipelineRelease(ctx context.Context, req *pipelinepb.TriggerUserPipelineReleaseRequest) (resp *pipelinepb.TriggerUserPipelineReleaseResponse, err error) {
	r, err := h.TriggerNamespacePipelineRelease(ctx, &pipelinepb.TriggerNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerUserPipelineReleaseResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

// TriggerOrganizationPipelineRelease triggers a pipeline release for an organization.
func (h *PublicHandler) TriggerOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.TriggerOrganizationPipelineReleaseRequest) (resp *pipelinepb.TriggerOrganizationPipelineReleaseResponse, err error) {
	r, err := h.TriggerNamespacePipelineRelease(ctx, &pipelinepb.TriggerNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerOrganizationPipelineReleaseResponse{Outputs: r.Outputs, Metadata: r.Metadata}, nil
}

// TriggerNamespacePipelineRelease triggers a pipeline release for a namespace.
func (h *PublicHandler) TriggerNamespacePipelineRelease(ctx context.Context, req *pipelinepb.TriggerNamespacePipelineReleaseRequest) (resp *pipelinepb.TriggerNamespacePipelineReleaseResponse, err error) {

	ns, releaseID, pbPipeline, _, returnTraces, err := h.preTriggerNamespacePipelineRelease(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	outputs, metadata, err := h.service.TriggerNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
	if err != nil {
		return nil, err
	}

	return &pipelinepb.TriggerNamespacePipelineReleaseResponse{Outputs: outputs, Metadata: metadata}, nil
}

// TriggerAsyncUserPipelineRelease triggers an async pipeline release for a user.
func (h *PublicHandler) TriggerAsyncUserPipelineRelease(ctx context.Context, req *pipelinepb.TriggerAsyncUserPipelineReleaseRequest) (resp *pipelinepb.TriggerAsyncUserPipelineReleaseResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipelineRelease(ctx, &pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerAsyncUserPipelineReleaseResponse{Operation: r.Operation}, nil
}

// TriggerAsyncOrganizationPipelineRelease triggers an async pipeline release for an organization.
func (h *PublicHandler) TriggerAsyncOrganizationPipelineRelease(ctx context.Context, req *pipelinepb.TriggerAsyncOrganizationPipelineReleaseRequest) (resp *pipelinepb.TriggerAsyncOrganizationPipelineReleaseResponse, err error) {
	r, err := h.TriggerAsyncNamespacePipelineRelease(ctx, &pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest{
		NamespaceId: strings.Split(req.Name, "/")[1],
		PipelineId:  strings.Split(req.Name, "/")[3],
		ReleaseId:   strings.Split(req.Name, "/")[5],
		Inputs:      req.Inputs,
		Data:        req.Data,
	})
	if err != nil {
		return nil, err
	}
	return &pipelinepb.TriggerAsyncOrganizationPipelineReleaseResponse{Operation: r.Operation}, nil
}

// TriggerAsyncNamespacePipelineRelease triggers an async pipeline release for a namespace.
func (h *PublicHandler) TriggerAsyncNamespacePipelineRelease(ctx context.Context, req *pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest) (resp *pipelinepb.TriggerAsyncNamespacePipelineReleaseResponse, err error) {

	ns, releaseID, pbPipeline, _, returnTraces, err := h.preTriggerNamespacePipelineRelease(ctx, req)
	if err != nil {
		return nil, err
	}

	logUUID, _ := uuid.NewV4()
	operation, err := h.service.TriggerAsyncNamespacePipelineReleaseByID(ctx, ns, uuid.FromStringOrNil(pbPipeline.Uid), releaseID, mergeInputsIntoData(req.GetInputs(), req.GetData()), logUUID.String(), returnTraces)
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
