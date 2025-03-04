package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"
	"github.com/instill-ai/x/errmsg"
	"github.com/instill-ai/x/minio"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	resourcex "github.com/instill-ai/x/resource"
)

var preserveTags = []string{"featured", "feature"}

func (s *service) GetHubStats(ctx context.Context) (*pipelinepb.GetHubStatsResponse, error) {

	uidAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
	if err != nil {
		return &pipelinepb.GetHubStatsResponse{}, err
	}

	hubStats, err := s.repository.GetHubStats(uidAllowList)

	if err != nil {
		return &pipelinepb.GetHubStatsResponse{}, err
	}

	return &pipelinepb.GetHubStatsResponse{
		NumberOfPublicPipelines:   hubStats.NumberOfPublicPipelines,
		NumberOfFeaturedPipelines: hubStats.NumberOfFeaturedPipelines,
	}, nil
}

func (s *service) ListPipelines(ctx context.Context, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, visibility *pipelinepb.Pipeline_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*pipelinepb.Pipeline, int32, string, error) {

	uidAllowList := []uuid.UUID{}
	var err error

	// TODO: optimize the logic
	if visibility != nil && *visibility == pipelinepb.Pipeline_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == pipelinepb.Pipeline_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
	}

	presetOrgResp, err := s.mgmtPrivateServiceClient.GetOrganizationAdmin(ctx, &mgmtpb.GetOrganizationAdminRequest{OrganizationId: constant.PresetNamespaceID})
	if err != nil {
		return nil, 0, "", err
	}
	presetNamespaceUID := uuid.FromStringOrNil(presetOrgResp.Organization.Uid)
	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, int64(pageSize), pageToken, view <= pipelinepb.Pipeline_VIEW_BASIC, filter, uidAllowList, showDeleted, true, order, presetNamespaceUID)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelines, err := s.converter.ConvertPipelinesToPB(ctx, dbPipelines, view, true)
	return pbPipelines, int32(totalSize), nextPageToken, err

}

func (s *service) GetPipelineByUID(ctx context.Context, uid uuid.UUID, view pipelinepb.Pipeline_View) (*pipelinepb.Pipeline, error) {

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", uid, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, uid, view <= pipelinepb.Pipeline_VIEW_BASIC, true)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineToPB(ctx, dbPipeline, view, true, true)
}

func (s *service) CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, pbPipeline *pipelinepb.Pipeline) (*pipelinepb.Pipeline, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return nil, err
		}
		if !granted {
			return nil, errdomain.ErrUnauthorized
		}
	} else if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
		return nil, errdomain.ErrUnauthorized
	}

	dbPipeline, err := s.converter.ConvertPipelineToDB(ctx, ns, pbPipeline)
	if err != nil {
		return nil, err
	}
	if dbPipeline.Recipe != nil {
		if err := s.checkSecret(ctx, dbPipeline.Recipe.Component); err != nil {
			return nil, fmt.Errorf("checking referenced secrets: %w", err)
		}
	}

	dbPipeline.ShareCode = generateShareCode()

	if err := s.repository.CreateNamespacePipeline(ctx, dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, dbPipeline.ID, false, true)
	if err != nil {
		return nil, err
	}

	if err := s.configureRunOn(ctx, configureRunOnParams{
		Namespace:   ns,
		pipelineUID: dbPipeline.UID,
		releaseUID:  uuid.Nil,
		recipe:      dbCreatedPipeline.Recipe,
	}); err != nil {
		return nil, err
	}

	ownerType := string(ns.NsType)[0 : len(string(ns.NsType))-1]
	ownerUID := ns.NsUID
	err = s.aclClient.SetOwner(ctx, "pipeline", dbCreatedPipeline.UID, ownerType, ownerUID)
	if err != nil {
		return nil, err
	}
	// TODO: use OpenFGA as single source of truth
	err = s.aclClient.SetPipelinePermissionMap(ctx, dbCreatedPipeline)
	if err != nil {
		return nil, err
	}
	toCreatedTags := pbPipeline.GetTags()
	toBeCreatedTagNames := make([]string, 0, len(toCreatedTags))
	for _, tag := range toCreatedTags {
		tag = strings.ToLower(tag)
		if !slices.Contains(preserveTags, tag) {
			toBeCreatedTagNames = append(toBeCreatedTagNames, tag)
		}
	}

	if len(toBeCreatedTagNames) > 0 {
		err = s.repository.CreatePipelineTags(ctx, dbCreatedPipeline.UID, toBeCreatedTagNames)
		if err != nil {
			return nil, err
		}
		dbCreatedPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, dbPipeline.ID, false, true)
		if err != nil {
			return nil, err
		}
	}

	pipeline, err := s.converter.ConvertPipelineToPB(ctx, dbCreatedPipeline, pipelinepb.Pipeline_VIEW_FULL, false, true)
	if err != nil {
		return nil, err
	}
	pipeline.Permission = &pipelinepb.Permission{
		CanEdit:    true,
		CanTrigger: true,
	}
	return pipeline, nil
}

func (s *service) ListNamespacePipelines(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, visibility *pipelinepb.Pipeline_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*pipelinepb.Pipeline, int32, string, error) {

	ownerPermalink := ns.Permalink()

	uidAllowList := []uuid.UUID{}
	var err error

	// TODO: optimize the logic
	if visibility != nil && *visibility == pipelinepb.Pipeline_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == pipelinepb.Pipeline_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbPipelines, ps, pt, err := s.repository.ListNamespacePipelines(ctx, ownerPermalink, int64(pageSize), pageToken, view <= pipelinepb.Pipeline_VIEW_BASIC, filter, uidAllowList, showDeleted, true, order)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.converter.ConvertPipelinesToPB(ctx, dbPipelines, view, true)
	return pbPipelines, int32(ps), pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, filter filtering.Filter, showDeleted bool) ([]*pipelinepb.Pipeline, int32, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, int64(pageSize), pageToken, view <= pipelinepb.Pipeline_VIEW_BASIC, filter, showDeleted, true)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.converter.ConvertPipelinesToPB(ctx, dbPipelines, view, true)
	return pbPipelines, int32(ps), pt, err

}

func (s *service) GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, view pipelinepb.Pipeline_View) (*pipelinepb.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, view <= pipelinepb.Pipeline_VIEW_BASIC, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	return s.converter.ConvertPipelineToPB(ctx, dbPipeline, view, true, true)
}

func (s *service) GetNamespacePipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, id string) (uuid.UUID, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, true)
	if err != nil {
		return uuid.Nil, err
	}

	dbPipelineRelease, err := s.repository.GetLatestNamespacePipelineRelease(ctx, ownerPermalink, dbPipeline.UID, true)
	if err != nil {
		return uuid.Nil, err
	}

	return dbPipelineRelease.UID, nil
}

func (s *service) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view pipelinepb.Pipeline_View) (*pipelinepb.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, uid, view <= pipelinepb.Pipeline_VIEW_BASIC, true)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineToPB(ctx, dbPipeline, view, true, true)

}

type configureRunOnParams struct {
	resource.Namespace
	pipelineUID uuid.UUID
	releaseUID  uuid.UUID
	recipe      *datamodel.Recipe
}

func (s *service) marshalEventSettings(ctx context.Context, ns resource.Namespace, config, setup any) (format.Value, format.Value, error) {
	marshaler := data.NewMarshaler()
	cfg, err := marshaler.Marshal(config)
	if err != nil {
		return nil, nil, err
	}
	var st format.Value
	if setup != nil {
		st, err = marshaler.Marshal(setup)
		if err != nil {
			return nil, nil, err
		}
		if connRef, ok := st.(format.ReferenceString); ok {
			connID, err := recipe.ConnectionIDFromReference(connRef.String())
			if err != nil {
				return nil, nil, err
			}

			conn, err := s.repository.GetNamespaceConnectionByID(ctx, ns.NsUID, connID)
			if err != nil {
				if errors.Is(err, errdomain.ErrNotFound) {
					err = errmsg.AddMessage(err, fmt.Sprintf("Connection %s doesn't exist.", connID))
				}
				return nil, nil, err
			}

			var s map[string]any
			if err := json.Unmarshal(conn.Setup, &s); err != nil {
				return nil, nil, err
			}

			setupVal, err := data.NewValue(s)
			if err != nil {
				return nil, nil, err
			}
			st = setupVal
		}
	}
	return cfg, st, nil
}

func (s *service) configureRunOn(ctx context.Context, params configureRunOnParams) error {

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, params.pipelineUID, false, true)
	if err != nil {
		return err
	}

	switch {

	// Case 1: Update unversioned pipeline and there is no existing release
	case params.releaseUID == uuid.Nil && len(dbPipeline.Releases) == 0:
		if err := s.clearRunOn(ctx, params); err != nil {
			return err
		}
		if err := s.updateRunOn(ctx, params); err != nil {
			return err
		}
		return nil

	// Case 2: Update versioned pipeline
	case params.releaseUID != uuid.Nil:
		if err := s.clearRunOn(ctx, params); err != nil {
			return err
		}
		if err := s.updateRunOn(ctx, params); err != nil {
			return err
		}
		return nil

	// Case 3: Update unversioned pipeline but there are existing releases
	default:
		// Do nothing
		return nil
	}

}

func (s *service) clearRunOn(ctx context.Context, params configureRunOnParams) error {
	// Unregister all webhooks from vendor
	runOn, err := s.repository.ListPipelineRunOns(ctx, params.pipelineUID)
	if err != nil {
		return err
	}

	for _, r := range runOn.PipelineRunOns {
		var origCfg any
		var origSetup any
		if r.Config != nil {
			err = json.Unmarshal(r.Config, &origCfg)
			if err != nil {
				return err
			}
		}
		if r.Setup != nil {
			err = json.Unmarshal(r.Setup, &origSetup)
			if err != nil {
				return err
			}
		}
		cfg, setup, err := s.marshalEventSettings(ctx, params.Namespace, origCfg, origSetup)
		if err == nil {
			identifier := base.Identifier{}
			err = json.Unmarshal(r.Identifier, &identifier)
			if err != nil {
				return err
			}
			err = s.component.UnregisterEvent(ctx, r.RunOnType, &base.UnregisterEventSettings{
				EventSettings: base.EventSettings{
					Config: cfg,
					Setup:  setup,
				},
			}, []base.Identifier{identifier})
			if err != nil {
				return err
			}
			err = s.repository.DeletePipelineRunOn(ctx, r.UID)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (s *service) updateRunOn(ctx context.Context, params configureRunOnParams) error {

	if params.recipe == nil {
		return nil
	}

	// Register all events from vendor
	if params.recipe != nil && len(params.recipe.On) > 0 {
		for eventID, v := range params.recipe.On {
			if v != nil {
				cfg, setup, err := s.marshalEventSettings(ctx, params.Namespace, v.Config, v.Setup)
				if err == nil {
					registrationUID := params.pipelineUID
					if params.releaseUID != uuid.Nil {
						registrationUID = params.releaseUID
					}
					identifiers, err := s.component.RegisterEvent(ctx, v.Type, &base.RegisterEventSettings{
						EventSettings: base.EventSettings{
							Config: cfg,
							Setup:  setup,
						},
						RegistrationUID: registrationUID,
					})
					if err != nil {
						return err
					}
					for _, identifier := range identifiers {
						jsonIdentifier, err := json.Marshal(identifier)
						if err != nil {
							return err
						}
						jsonConfig, err := json.Marshal(v.Config)
						if err != nil {
							return err
						}
						jsonSetup, err := json.Marshal(v.Setup)
						if err != nil {
							return err
						}
						err = s.repository.CreatePipelineRunOn(ctx, &datamodel.PipelineRunOn{
							RunOnType:   v.Type,
							EventID:     eventID,
							Identifier:  jsonIdentifier,
							PipelineUID: params.pipelineUID,
							ReleaseUID:  params.releaseUID,
							Config:      jsonConfig,
							Setup:       jsonSetup,
						})
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func (s *service) UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, toUpdPipeline *pipelinepb.Pipeline) (*pipelinepb.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.converter.ConvertPipelineToDB(ctx, ns, toUpdPipeline)
	if err != nil {
		return nil, err
	}

	if dbPipeline.Recipe != nil {
		if err := s.checkSecret(ctx, dbPipeline.Recipe.Component); err != nil {
			return nil, fmt.Errorf("checking referenced secrets: %w", err)
		}
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrUnauthorized
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, false); existingPipeline == nil {
		return nil, err
	}

	if existingPipeline.ShareCode == "" {
		dbPipeline.ShareCode = generateShareCode()
	}

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, dbPipeline.UID, dbPipeline); err != nil {
		return nil, err
	}

	toUpdTags := toUpdPipeline.GetTags()
	for i := range toUpdTags {
		toUpdTags[i] = strings.ToLower(toUpdTags[i])
	}
	currentTags := existingPipeline.TagNames()
	for i := range currentTags {
		currentTags[i] = strings.ToLower(currentTags[i])
	}

	toBeCreatedTagNames := make([]string, 0, len(toUpdTags))
	for _, tag := range toUpdTags {
		if !slices.Contains(currentTags, tag) && !slices.Contains(preserveTags, tag) {
			toBeCreatedTagNames = append(toBeCreatedTagNames, tag)
		}
	}

	toBeDeletedTagNames := make([]string, 0, len(toUpdTags))
	for _, tag := range currentTags {
		if !slices.Contains(toUpdTags, tag) && !slices.Contains(preserveTags, tag) {
			toBeDeletedTagNames = append(toBeDeletedTagNames, tag)
		}
	}
	if len(toBeDeletedTagNames) > 0 {
		err = s.repository.DeletePipelineTags(ctx, existingPipeline.UID, toBeDeletedTagNames)
		if err != nil {
			return nil, err
		}
	}
	if len(toBeCreatedTagNames) > 0 {
		err = s.repository.CreatePipelineTags(ctx, existingPipeline.UID, toBeCreatedTagNames)
		if err != nil {
			return nil, err
		}
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, toUpdPipeline.Id, false, true)
	if err != nil {
		return nil, err
	}

	if err := s.configureRunOn(ctx, configureRunOnParams{
		Namespace:   ns,
		pipelineUID: dbPipeline.UID,
		releaseUID:  uuid.Nil,
		recipe:      dbPipeline.Recipe,
	}); err != nil {
		return nil, err
	}

	oldSharing, _ := json.Marshal(existingPipeline.Sharing)
	newSharing, _ := json.Marshal(dbPipeline.Sharing)

	// TODO: use OpenFGA as single source of truth
	if string(oldSharing) != string(newSharing) {
		err = s.aclClient.SetPipelinePermissionMap(ctx, dbPipeline)
		if err != nil {
			return nil, err
		}
	}

	dbPipelineUpdated, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, err
	}
	pipeline, err := s.converter.ConvertPipelineToPB(ctx, dbPipelineUpdated, pipelinepb.Pipeline_VIEW_FULL, false, true)
	if err != nil {
		return nil, err
	}
	pipeline.Permission = &pipelinepb.Permission{
		CanEdit:    true,
		CanTrigger: true,
	}
	return pipeline, nil
}

func (s *service) DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) error {
	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrUnauthorized
	}

	// TODO: pagination
	pipelineReleases, _, _, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, dbPipeline.UID, 1000, "", false, filtering.Filter{}, false, false)
	if err != nil {
		return err
	}

	ch := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(pipelineReleases))

	for idx := range pipelineReleases {
		go func(r *datamodel.PipelineRelease) {
			defer wg.Done()
			err := s.DeleteNamespacePipelineReleaseByID(ctx, ns, dbPipeline.UID, r.ID)
			ch <- err
		}(pipelineReleases[idx])
	}
	for range pipelineReleases {
		err = <-ch
		if err != nil {
			return err
		}
	}

	err = s.aclClient.Purge(ctx, "pipeline", dbPipeline.UID)
	if err != nil {
		return err
	}
	return s.repository.DeleteNamespacePipelineByID(ctx, ownerPermalink, id)
}

func (s *service) generateCloneTargetNamespace(ctx context.Context, targetNamespace string) (resource.Namespace, error) {

	resp, err := s.mgmtPrivateServiceClient.CheckNamespaceAdmin(ctx, &mgmtpb.CheckNamespaceAdminRequest{
		Id: targetNamespace,
	})
	if err != nil {
		return resource.Namespace{}, err
	}

	var targetNS resource.Namespace
	if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_USER {
		targetNS = resource.Namespace{
			NsType: resource.User,
			NsID:   targetNamespace,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}
	} else if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION {
		targetNS = resource.Namespace{
			NsType: resource.Organization,
			NsID:   targetNamespace,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}
	} else {
		return resource.Namespace{}, errdomain.ErrInvalidCloneTarget
	}

	return targetNS, nil
}

func (s *service) CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, id, targetNamespaceID, targetPipelineID, description string, sharing *pipelinepb.Sharing) (*pipelinepb.Pipeline, error) {
	sourcePipeline, err := s.GetNamespacePipelineByID(ctx, ns, id, pipelinepb.Pipeline_VIEW_RECIPE)
	if err != nil {
		return nil, err
	}
	targetNS, err := s.generateCloneTargetNamespace(ctx, targetNamespaceID)
	if err != nil {
		return nil, err
	}

	newPipeline := &pipelinepb.Pipeline{
		Id:          targetPipelineID,
		Description: &description,
		Sharing:     sharing,
		RawRecipe:   sourcePipeline.RawRecipe,
		Metadata:    sourcePipeline.Metadata,
	}

	pipeline, err := s.CreateNamespacePipeline(ctx, targetNS, newPipeline)
	if err != nil {
		return nil, err
	}
	err = s.repository.AddPipelineClones(ctx, uuid.FromStringOrNil(sourcePipeline.Uid))
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

func (s *service) CloneNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id, targetNamespaceID, targetPipelineID, description string, sharing *pipelinepb.Sharing) (*pipelinepb.Pipeline, error) {
	sourcePipelineRelease, err := s.GetNamespacePipelineReleaseByID(ctx, ns, pipelineUID, id, pipelinepb.Pipeline_VIEW_RECIPE)
	if err != nil {
		return nil, err
	}
	targetNS, err := s.generateCloneTargetNamespace(ctx, targetNamespaceID)
	if err != nil {
		return nil, err
	}

	newPipeline := &pipelinepb.Pipeline{
		Id:          targetPipelineID,
		Description: &description,
		Sharing:     sharing,
		RawRecipe:   sourcePipelineRelease.RawRecipe,
		Metadata:    sourcePipelineRelease.Metadata,
	}

	pipeline, err := s.CreateNamespacePipeline(ctx, targetNS, newPipeline)
	if err != nil {
		return nil, err
	}
	err = s.repository.AddPipelineClones(ctx, pipelineUID)
	if err != nil {
		return nil, err
	}
	return pipeline, nil
}

func (s *service) ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) ([]*pipelinepb.ErrPipelineValidation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrUnauthorized
	}

	validateErrs, err := s.checkRecipe(dbPipeline.Recipe)
	if err != nil {
		return nil, err
	}

	return validateErrs, nil

}

func (s *service) UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, id string, newID string) (*pipelinepb.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	// Validation: Pipeline existence
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrUnauthorized
	}

	if err := s.repository.UpdateNamespacePipelineIDByID(ctx, ownerPermalink, id, newID); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, newID, false, true)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineToPB(ctx, dbPipeline, pipelinepb.Pipeline_VIEW_FULL, true, true)
}

// preTriggerPipeline does the following:
//  1. Upload pipeline input data to minio if the data is blob data.
//  2. New workflow memory.
//     a. Set the default values for the variables for memory data and
//     uploading pipeline data.
//     b. Set the data with data.Value for the memory data, which will be
//     used for pipeline running.
//     c. Upload "uploading pipeline data" to minio for pipeline run logger.
//  3. Map the settings in recipe to the format in workflow memory.
//  4. Enable the streaming mode when the header contains "text/event-stream"
//
// We upload User Input Data by `uploadBlobAndGetDownloadURL`, which exposes
// the public URL because it will be used by `console` & external users.
// We upload Pipeline Input Data by `uploadPipelineRunInputsToMinio`, which
// does not expose the public URL. The URL will be used by pipeline run logger.
func (s *service) preTriggerPipeline(
	ctx context.Context,
	requester resource.Namespace,
	r *datamodel.Recipe,
	pipelineTriggerID string,
	pipelineData []*pipelinepb.TriggerData,
	expiryRule minio.ExpiryRule,
) error {
	batchSize := len(pipelineData)
	if batchSize > constant.MaxBatchSize {
		return ErrExceedMaxBatchSize
	}

	typeMap := map[string]string{}
	defaultValueMap := map[string]any{}

	for k, v := range r.Variable {
		typeMap[k] = v.Type
		defaultValueMap[k] = v.Default
	}

	errors := []string{}

	for idx, data := range pipelineData {
		vars := data.Variable
		b, err := protojson.Marshal(vars)
		if err != nil {
			errors = append(errors, fmt.Sprintf("inputs[%d]: data error", idx))
			continue
		}
		var i any
		if err := json.Unmarshal(b, &i); err != nil {
			errors = append(errors, fmt.Sprintf("inputs[%d]: data error", idx))
			continue
		}

		m := i.(map[string]any)
		for k := range m {
			switch str := m[k].(type) {
			case string:
				if isUnstructuredType(typeMap[k]) {
					// Skip the base64 decoding if the string is a URL.
					if strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://") {
						continue
					}
					downloadURL, err := s.uploadBlobAndGetDownloadURL(ctx, str, requester, expiryRule)
					if err != nil {
						return fmt.Errorf("upload blob and get download url: %w", err)
					}
					vars.Fields[k] = structpb.NewStringValue(downloadURL)
				}
			case []string:
				if isUnstructuredType(typeMap[k]) {
					for idx := range str {
						// Skip the base64 decoding if the string is a URL
						if strings.HasPrefix(str[idx], "http://") || strings.HasPrefix(str[idx], "https://") {
							continue
						}
						downloadURL, err := s.uploadBlobAndGetDownloadURL(ctx, str[idx], requester, expiryRule)
						if err != nil {
							return fmt.Errorf("upload blob and get download url: %w", err)
						}
						vars.Fields[k] = structpb.NewStringValue(downloadURL)

					}
				}
			}
		}

	}

	if len(errors) > 0 {
		return fmt.Errorf("[Pipeline Trigger Data Error] %s", strings.Join(errors, "; "))
	}

	wfm, err := s.memory.NewWorkflowMemory(ctx, pipelineTriggerID, nil, len(pipelineData))
	if err != nil {
		return err
	}

	types := map[string][]string{}
	for k, v := range typeMap {
		types[k] = []string{v}
	}

	uploadingPipelineData := make([]map[string]any, len(pipelineData))
	for idx := range uploadingPipelineData {
		uploadingPipelineData[idx] = make(map[string]any)
	}

	// TODO(huitang): implement a structpb to format.Value converter
	for idx, d := range pipelineData {

		variable := data.Map{}
		for k := range typeMap {
			v := d.Variable.Fields[k]
			if _, ok := typeMap[k]; !ok {
				continue
			}

			if v == nil {
				// If the field is required but no value is provided, return an error.
				if r.Variable[k].Required {
					return fmt.Errorf("missing required variable: %s", k)
				}

				// If the field has no value and no default value is specified,
				// represent it as null. A null value indicates that the field
				// is missing and should be handled as such by components.
				if d, ok := defaultValueMap[k]; !ok || d == nil {
					variable[k] = data.NewNull()
					uploadingPipelineData[idx][k] = nil
					continue
				}
			}

			switch typeMap[k] {
			case "boolean":
				if v == nil {
					variable[k] = data.NewBoolean(defaultValueMap[k].(bool))
					uploadingPipelineData[idx][k] = defaultValueMap[k].(bool)
				} else {
					if _, ok := v.Kind.(*structpb.Value_BoolValue); !ok {
						return fmt.Errorf("%w: invalid boolean value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k] = data.NewBoolean(v.GetBoolValue())
					uploadingPipelineData[idx][k] = v.GetBoolValue()
				}
			case "array:boolean":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					uploadingDataArray := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i] = data.NewBoolean(val.(bool))
						uploadingDataArray[i] = val.(bool)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					uploadingDataArray := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_BoolValue); !ok {
							return fmt.Errorf("%w: invalid boolean value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i] = data.NewBoolean(val.GetBoolValue())
						uploadingDataArray[i] = val.GetBoolValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				}
			case "string":
				if v == nil {
					variable[k] = data.NewString(defaultValueMap[k].(string))
					uploadingPipelineData[idx][k] = defaultValueMap[k].(string)
				} else {
					if _, ok := v.Kind.(*structpb.Value_StringValue); !ok {
						return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k] = data.NewString(v.GetStringValue())
					uploadingPipelineData[idx][k] = v.GetStringValue()
				}
			case "array:string":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					uploadingDataArray := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i] = data.NewString(val.(string))
						uploadingDataArray[i] = val.(string)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					uploadingDataArray := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_StringValue); !ok {
							return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i] = data.NewString(val.GetStringValue())
						uploadingDataArray[i] = val.GetStringValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				}
			case "integer":
				if v == nil {
					variable[k] = data.NewNumberFromFloat(defaultValueMap[k].(float64))
					uploadingPipelineData[idx][k] = defaultValueMap[k].(float64)
				} else {
					if _, ok := v.Kind.(*structpb.Value_NumberValue); !ok {
						return fmt.Errorf("%w: invalid number value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k] = data.NewNumberFromFloat(v.GetNumberValue())
					uploadingPipelineData[idx][k] = v.GetNumberValue()
				}
			case "array:integer":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					uploadingDataArray := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i] = data.NewNumberFromFloat(val.(float64))
						uploadingDataArray[i] = val.(float64)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					uploadingDataArray := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_NumberValue); !ok {
							return fmt.Errorf("%w: invalid number value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i] = data.NewNumberFromFloat(val.GetNumberValue())
						uploadingDataArray[i] = val.GetNumberValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				}
			case "number":
				if v == nil {

					// TODO: this is a temporary solution to handle the default
					// value of integer type, we will implement a better
					// solution for conversion JSON betweeen instill format
					switch num := defaultValueMap[k].(type) {
					case int:
						variable[k] = data.NewNumberFromFloat(float64(num))
						uploadingPipelineData[idx][k] = float64(num)
					case float64:
						variable[k] = data.NewNumberFromFloat(num)
						uploadingPipelineData[idx][k] = num
					default:
						return fmt.Errorf("invalid number value: %v", defaultValueMap[k])
					}

				} else {
					if _, ok := v.Kind.(*structpb.Value_NumberValue); !ok {
						return fmt.Errorf("%w: invalid number value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k] = data.NewNumberFromFloat(v.GetNumberValue())
					uploadingPipelineData[idx][k] = v.GetNumberValue()
				}
			case "array:number":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					uploadingDataArray := make([]any, len(defaultValueMap[k].([]any)))

					// TODO: this is a temporary solution to handle the default
					// value of integer type, we will implement a better
					// solution for conversion JSON betweeen instill type
					for i, val := range defaultValueMap[k].([]any) {
						switch num := val.(type) {
						case int:
							array[i] = data.NewNumberFromFloat(float64(num))
							uploadingDataArray[i] = float64(num)
						case float64:
							array[i] = data.NewNumberFromFloat(num)
							uploadingDataArray[i] = num
						default:
							return fmt.Errorf("invalid number value: %v", val)
						}
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					uploadingDataArray := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_NumberValue); !ok {
							return fmt.Errorf("%w: invalid number value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i] = data.NewNumberFromFloat(val.GetNumberValue())
						uploadingDataArray[i] = val.GetNumberValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = uploadingDataArray
				}
			case "image", "image/*":
				if v == nil {
					variable[k], err = data.NewImageFromURL(ctx, s.binaryFetcher, defaultValueMap[k].(string))
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = defaultValueMap[k].(string)
				} else {
					if _, ok := v.Kind.(*structpb.Value_StringValue); !ok {
						return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k], err = data.NewImageFromURL(ctx, s.binaryFetcher, v.GetStringValue())
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = v.GetStringValue()
				}
			case "array:image", "array:image/*":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					arrayWithURL := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i], err = data.NewImageFromURL(ctx, s.binaryFetcher, val.(string))
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.(string)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					arrayWithURL := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_StringValue); !ok {
							return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i], err = data.NewImageFromURL(ctx, s.binaryFetcher, val.GetStringValue())
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.GetStringValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				}
			case "audio", "audio/*":
				if v == nil {
					variable[k], err = data.NewAudioFromURL(ctx, s.binaryFetcher, defaultValueMap[k].(string))
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = defaultValueMap[k].(string)
				} else {
					if _, ok := v.Kind.(*structpb.Value_StringValue); !ok {
						return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k], err = data.NewAudioFromURL(ctx, s.binaryFetcher, v.GetStringValue())
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = v.GetStringValue()
				}
			case "array:audio", "array:audio/*":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					arrayWithURL := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i], err = data.NewAudioFromURL(ctx, s.binaryFetcher, val.(string))
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.(string)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					arrayWithURL := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_StringValue); !ok {
							return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i], err = data.NewAudioFromURL(ctx, s.binaryFetcher, val.GetStringValue())
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.GetStringValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				}
			case "video", "video/*":
				if v == nil {
					variable[k], err = data.NewVideoFromURL(ctx, s.binaryFetcher, defaultValueMap[k].(string))
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = defaultValueMap[k].(string)
				} else {
					if _, ok := v.Kind.(*structpb.Value_StringValue); !ok {
						return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k], err = data.NewVideoFromURL(ctx, s.binaryFetcher, v.GetStringValue())
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = v.GetStringValue()
				}
			case "array:video", "array:video/*":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					arrayWithURL := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i], err = data.NewVideoFromURL(ctx, s.binaryFetcher, val.(string))
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.(string)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					arrayWithURL := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_StringValue); !ok {
							return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i], err = data.NewVideoFromURL(ctx, s.binaryFetcher, val.GetStringValue())
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.GetStringValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				}

			case "document":
				if v == nil {
					variable[k], err = data.NewDocumentFromURL(ctx, s.binaryFetcher, defaultValueMap[k].(string))
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = defaultValueMap[k].(string)
				} else {
					if _, ok := v.Kind.(*structpb.Value_StringValue); !ok {
						return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k], err = data.NewDocumentFromURL(ctx, s.binaryFetcher, v.GetStringValue())
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = v.GetStringValue()
				}
			case "array:document":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					arrayWithURL := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i], err = data.NewDocumentFromURL(ctx, s.binaryFetcher, val.(string))
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.(string)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					arrayWithURL := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_StringValue); !ok {
							return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i], err = data.NewDocumentFromURL(ctx, s.binaryFetcher, val.GetStringValue())
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.GetStringValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				}
			case "file", "*/*":
				if v == nil {
					variable[k], err = data.NewBinaryFromURL(ctx, s.binaryFetcher, defaultValueMap[k].(string))
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = defaultValueMap[k].(string)
				} else {
					if _, ok := v.Kind.(*structpb.Value_StringValue); !ok {
						return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, v)
					}
					variable[k], err = data.NewBinaryFromURL(ctx, s.binaryFetcher, v.GetStringValue())
					if err != nil {
						return err
					}
					uploadingPipelineData[idx][k] = v.GetStringValue()
				}
			case "array:file", "array:*/*":
				if v == nil {
					array := make(data.Array, len(defaultValueMap[k].([]any)))
					arrayWithURL := make([]any, len(defaultValueMap[k].([]any)))
					for i, val := range defaultValueMap[k].([]any) {
						array[i], err = data.NewBinaryFromURL(ctx, s.binaryFetcher, val.(string))
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.(string)
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				} else {
					array := make(data.Array, len(v.GetListValue().Values))
					arrayWithURL := make([]any, len(v.GetListValue().Values))
					for i, val := range v.GetListValue().Values {
						if _, ok := val.Kind.(*structpb.Value_StringValue); !ok {
							return fmt.Errorf("%w: invalid string value: %v", errdomain.ErrInvalidArgument, val)
						}
						array[i], err = data.NewBinaryFromURL(ctx, s.binaryFetcher, val.GetStringValue())
						if err != nil {
							return err
						}
						arrayWithURL[i] = val.GetStringValue()
					}
					variable[k] = array
					uploadingPipelineData[idx][k] = arrayWithURL
				}
			case "semi-structured/*", "semi-structured/json", "json":

				if v == nil {
					jv, err := data.NewJSONValue(defaultValueMap[k])
					if err != nil {
						return err
					}
					variable[k] = jv
					uploadingPipelineData[idx][k] = jv
				} else {
					switch v.Kind.(type) {
					case *structpb.Value_StructValue:
						j := map[string]any{}
						b, err := protojson.Marshal(v)
						if err != nil {
							return err
						}
						err = json.Unmarshal(b, &j)
						if err != nil {
							return err
						}
						jv, err := data.NewJSONValue(j)
						if err != nil {
							return err
						}
						variable[k] = jv
						uploadingPipelineData[idx][k] = jv
					case *structpb.Value_ListValue:
						j := []any{}
						b, err := protojson.Marshal(v)
						if err != nil {
							return err
						}
						err = json.Unmarshal(b, &j)
						if err != nil {
							return err
						}
						jv, err := data.NewJSONValue(j)
						if err != nil {
							return err
						}
						variable[k] = jv
						uploadingPipelineData[idx][k] = jv
					}
				}

			}
			if err != nil {
				return err
			}
		}

		err = s.uploadPipelineRunInputsToMinio(ctx, uploadPipelineRunInputsToMinioParam{
			pipelineTriggerID: pipelineTriggerID,
			expiryRule:        expiryRule,
			pipelineData:      uploadingPipelineData,
		})
		if err != nil {
			return fmt.Errorf("pipeline run inputs to minio: %w", err)
		}

		err = wfm.Set(ctx, idx, constant.SegVariable, variable)
		if err != nil {
			return err
		}

		// Each batch may overwrite the secret and connection references in the
		// recipe by providing a new value (secret) or reference (connection)
		// in the trigger data. For secrets, the value will be read from the
		// trigger data instead of from the namespace's secrets. In the case of
		// connections, this works as aliasing an existing connection with the
		// ID defined in the recipe.
		//
		// This is useful for parametrizing pipeline triggers and for
		// triggering pipelines owned by other namespaces (which might
		// reference secrets or connections that exist in their namespaces).
		secret := data.Map{}
		for k, v := range d.Secret {
			secret[k] = data.NewString(v)
		}
		err = wfm.Set(ctx, idx, constant.SegSecret, secret)
		if err != nil {
			return err
		}

		connRefs := data.Map{}
		for k, v := range d.ConnectionReferences {
			connRefs[k] = data.NewString(v)
		}

		err = wfm.Set(ctx, idx, constant.SegConnection, connRefs)
		if err != nil {
			return err
		}

	}
	isStreaming := resource.GetRequestSingleHeader(ctx, constant.HeaderAccept) == "text/event-stream"
	if isStreaming {
		wfm.EnableStreaming()
	}
	return nil
}

func (s *service) CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pipelineRelease *pipelinepb.PipelineRelease) (*pipelinepb.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, false)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrUnauthorized
	}

	dbPipelineReleaseToCreate, err := s.converter.ConvertPipelineReleaseToDB(ctx, pipelineUID, pipelineRelease)
	if err != nil {
		return nil, err
	}

	if dbPipelineReleaseToCreate.RecipeYAML == "" {
		dbPipelineReleaseToCreate.RecipeYAML = dbPipeline.RecipeYAML
	}
	if dbPipelineReleaseToCreate.Metadata == nil {
		dbPipelineReleaseToCreate.Metadata = dbPipeline.Metadata
	}

	if err := s.repository.CreateNamespacePipelineRelease(ctx, ownerPermalink, pipelineUID, dbPipelineReleaseToCreate); err != nil {
		return nil, err
	}

	dbCreatedPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, pipelineRelease.Id, false)
	if err != nil {
		return nil, err
	}

	if err := s.configureRunOn(ctx, configureRunOnParams{
		Namespace:   ns,
		pipelineUID: dbPipeline.UID,
		releaseUID:  dbCreatedPipelineRelease.UID,
		recipe:      dbCreatedPipelineRelease.Recipe,
	}); err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineReleaseToPB(ctx, dbPipeline, dbCreatedPipelineRelease, pipelinepb.Pipeline_VIEW_FULL)

}
func (s *service) ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, filter filtering.Filter, showDeleted bool) ([]*pipelinepb.PipelineRelease, int32, string, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, 0, "", errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, 0, "", err
	} else if !granted {
		return nil, 0, "", errdomain.ErrNotFound
	}

	dbPipelineReleases, ps, pt, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, pipelineUID, int64(pageSize), pageToken, view <= pipelinepb.Pipeline_VIEW_BASIC, filter, showDeleted, true)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelineReleases, err := s.converter.ConvertPipelineReleasesToPB(ctx, dbPipeline, dbPipelineReleases, view)
	return pbPipelineReleases, int32(ps), pt, err
}

func (s *service) GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, view pipelinepb.Pipeline_View) (*pipelinepb.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, view <= pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineReleaseToPB(ctx, dbPipeline, dbPipelineRelease, view)

}

func (s *service) UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, toUpdPipeline *pipelinepb.PipelineRelease) (*pipelinepb.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrUnauthorized
	}

	if _, err := s.GetNamespacePipelineReleaseByID(ctx, ns, pipelineUID, id, pipelinepb.Pipeline_VIEW_BASIC); err != nil {
		return nil, err
	}

	pbPipelineReleaseToUpdate, err := s.converter.ConvertPipelineReleaseToDB(ctx, pipelineUID, toUpdPipeline)
	if err != nil {
		return nil, err
	}
	if err := s.repository.UpdateNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, pbPipelineReleaseToUpdate); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, toUpdPipeline.Id, false)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineReleaseToPB(ctx, dbPipeline, dbPipelineRelease, pipelinepb.Pipeline_VIEW_FULL)
}

func (s *service) UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, newID string) (*pipelinepb.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, errdomain.ErrUnauthorized
	}

	// Validation: Pipeline existence
	_, err = s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, true)
	if err != nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespacePipelineReleaseIDByID(ctx, ownerPermalink, pipelineUID, id, newID); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, newID, false)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineReleaseToPB(ctx, dbPipeline, dbPipelineRelease, pipelinepb.Pipeline_VIEW_FULL)
}

func (s *service) DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrUnauthorized
	}

	return s.repository.DeleteNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id)
}

func (s *service) RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error {
	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, pipelineUID, pipelinepb.Pipeline_VIEW_BASIC)
	if err != nil {
		return errdomain.ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", uuid.FromStringOrNil(pipeline.GetUid()), "admin"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", uuid.FromStringOrNil(pipeline.GetUid()), "admin"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrUnauthorized
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, err = s.repository.GetPipelineByUID(ctx, pipelineUID, false, true); err != nil {
		return err
	}
	existingPipeline.Recipe = dbPipelineRelease.Recipe

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, existingPipeline.UID, existingPipeline); err != nil {
		return fmt.Errorf("updating pipeline: %w", err)
	}

	return nil
}

// TODO: share the code with worker/workflow.go
func (s *service) triggerPipeline(
	ctx context.Context,
	triggerParams triggerParams,
	returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	logger, _ := logger.GetZapLogger(ctx)

	defer func() {
		_ = s.memory.PurgeWorkflowMemory(ctx, triggerParams.pipelineTriggerID)
	}()

	workflowOptions := client.StartWorkflowOptions{
		ID:                       triggerParams.pipelineTriggerID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	requester, err := s.GetNamespaceByUID(ctx, triggerParams.requesterUID)
	if err != nil {
		return nil, nil, err
	}
	user, err := s.GetNamespaceByUID(ctx, triggerParams.userUID)
	if err != nil {
		return nil, nil, err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			TriggerFromAPI: true,
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    triggerParams.pipelineTriggerID,
				PipelineID:           triggerParams.pipelineID,
				PipelineUID:          triggerParams.pipelineUID,
				PipelineReleaseID:    triggerParams.pipelineReleaseID,
				PipelineReleaseUID:   triggerParams.pipelineReleaseUID,
				PipelineOwner:        triggerParams.ns,
				PipelineUserUID:      user.NsUID,
				PipelineRequesterUID: requester.NsUID,
				PipelineRequesterID:  requester.NsID,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
				ExpiryRule:           triggerParams.expiryRule,
			},
			Mode:      mgmtpb.Mode_MODE_SYNC,
			WorkerUID: s.workerUID,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, nil, err
	}

	if err := we.Get(ctx, nil); err != nil {
		// Note: We categorize all pipeline trigger errors as ErrTriggerFail
		// and mark the code as 400 InvalidArgument for now.
		// We should further categorize them into InvalidArgument or
		// PreconditionFailed or InternalError in the future.
		err = fmt.Errorf("%w:%w", ErrTriggerFail, err)

		var applicationErr *temporal.ApplicationError
		if errors.As(err, &applicationErr) && applicationErr.Message() != "" {
			err = errmsg.AddMessage(err, applicationErr.Message())
		}

		return nil, nil, err
	}

	return s.getOutputsAndMetadata(ctx, triggerParams.pipelineTriggerID, returnTraces)
}

type triggerParams struct {
	ns                 resource.Namespace
	pipelineID         string
	pipelineUID        uuid.UUID
	pipelineReleaseID  string
	pipelineReleaseUID uuid.UUID
	pipelineTriggerID  string
	requesterUID       uuid.UUID
	userUID            uuid.UUID
	expiryRule         minio.ExpiryRule
}

func (s *service) triggerAsyncPipeline(ctx context.Context, params triggerParams) (*longrunningpb.Operation, error) {

	defer func() {
		go func() {
			// We only retain the memory for a maximum of 60 minutes.
			time.Sleep(60 * time.Minute)
			_ = s.memory.PurgeWorkflowMemory(ctx, params.pipelineTriggerID)
		}()
	}()

	logger, _ := logger.GetZapLogger(ctx)
	logger = logger.With(zap.String("triggerID", params.pipelineTriggerID))

	workflowOptions := client.StartWorkflowOptions{
		ID:                       params.pipelineTriggerID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	requester, err := s.GetNamespaceByUID(ctx, params.requesterUID)
	if err != nil {
		return nil, err
	}
	user, err := s.GetNamespaceByUID(ctx, params.userUID)
	if err != nil {
		return nil, err
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    params.pipelineTriggerID,
				PipelineID:           params.pipelineID,
				PipelineUID:          params.pipelineUID,
				PipelineReleaseID:    params.pipelineReleaseID,
				PipelineReleaseUID:   params.pipelineReleaseUID,
				PipelineOwner:        params.ns,
				PipelineUserUID:      user.NsUID,
				PipelineRequesterUID: requester.NsUID,
				PipelineRequesterID:  requester.NsID,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
				ExpiryRule:           params.expiryRule,
			},
			Mode:           mgmtpb.Mode_MODE_ASYNC,
			TriggerFromAPI: true,
			WorkerUID:      s.workerUID,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with workflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	// wait for trigger ends in goroutine and upload outputs
	utils.GoSafe(func() {
		defer func() {
			if err := s.memory.PurgeWorkflowMemory(ctx, params.pipelineTriggerID); err != nil {
				logger.Error("Couldn't purge workflow memory", zap.Error(err))
			}
		}()

		subCtx := context.Background()
		err = we.Get(subCtx, nil)
		if err != nil {
			err = fmt.Errorf("%w:%w", ErrTriggerFail, err)

			var applicationErr *temporal.ApplicationError
			if errors.As(err, &applicationErr) && applicationErr.Message() != "" {
				err = errmsg.AddMessage(err, applicationErr.Message())
			}
			logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))

			run, repoErr := s.repository.GetPipelineRunByUID(subCtx, uuid.FromStringOrNil(params.pipelineTriggerID))
			if repoErr != nil {
				logger.Error("failed to log pipeline run error", zap.Error(err), zap.Error(repoErr))
				return
			}

			s.logPipelineRunError(subCtx, params.pipelineTriggerID, err, run.StartedTime)
			return
		}
	})

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", params.pipelineTriggerID),
		Done: false,
	}, nil

}

func (s *service) getOutputsAndMetadata(ctx context.Context, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	wfm, err := s.memory.GetWorkflowMemory(ctx, pipelineTriggerID)
	if err != nil {
		return nil, nil, err
	}

	pipelineOutputs := make([]*structpb.Struct, wfm.GetBatchSize())

	for idx := range wfm.GetBatchSize() {
		output, err := wfm.Get(ctx, idx, constant.SegOutput)
		if err != nil {
			return nil, nil, err
		}
		outputStruct, err := output.ToStructValue()
		if err != nil {
			return nil, nil, err
		}
		pipelineOutputs[idx] = outputStruct.GetStructValue()
	}

	var metadata *pipelinepb.TriggerMetadata

	traces, err := recipe.GenerateTraces(ctx, wfm, returnTraces)
	if err != nil {
		return nil, nil, err
	}
	metadata = &pipelinepb.TriggerMetadata{
		Traces: traces,
	}

	return pipelineOutputs, metadata, nil
}

// checkRequesterPermission validates that the authenticated user can make
// requests on behalf of the resource identified by the requester UID.
func (s *service) checkRequesterPermission(ctx context.Context, pipeline *datamodel.Pipeline) error {
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	if authType != "user" {
		// Only authenticated users can switch namespaces.
		return errdomain.ErrUnauthorized
	}

	requester := resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey)
	authenticatedUser := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if requester == "" || authenticatedUser == requester {
		// Request doesn't contain impersonation.
		return nil
	}

	// The only impersonation that's currently implemented is switching to an
	// organization namespace.
	isMember, err := s.aclClient.CheckPermission(ctx, "organization", uuid.FromStringOrNil(requester), "member")
	if err != nil {
		return errmsg.AddMessage(
			fmt.Errorf("checking organization membership: %w", err),
			"Couldn't check organization membership.",
		)
	}

	if !isMember {
		return fmt.Errorf("authenticated user doesn't belong to requester organization: %w", errdomain.ErrUnauthorized)
	}

	if pipeline.IsPublic() {
		// Public pipelines can be always be triggered as an organization.
		return nil
	}

	if pipeline.OwnerUID().String() == requester {
		// Organizations can trigger their private pipelines.
		return nil
	}

	// Organizations can only trigger external private pipelines through a
	// shareable link.
	canTrigger, err := s.aclClient.CheckLinkPermission(ctx, "pipeline", pipeline.UID, "executor")
	if err != nil {
		return errmsg.AddMessage(
			fmt.Errorf("checking shareable link permissions: %w", err),
			"Couldn't validate shareable link.",
		)
	}

	if !canTrigger {
		return fmt.Errorf("organization can't trigger private external pipeline: %w", errdomain.ErrUnauthorized)
	}

	return nil
}

func (s *service) checkTriggerPermission(ctx context.Context, pipeline *datamodel.Pipeline) (err error) {
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", pipeline.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", pipeline.UID, "executor"); err != nil {
		return err
	} else if !granted {
		return errdomain.ErrUnauthorized
	}

	// For now, impersonation is only implemented for pipeline triggers. When
	// this is used in other entrypoints, the requester permission should be
	// checked at a higher level (e.g. handler or middleware).
	if err := s.checkRequesterPermission(ctx, pipeline); err != nil {
		return fmt.Errorf("checking requester permission: %w", err)
	}
	return nil
}

func (s *service) TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {
	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, nil, errdomain.ErrNotFound
	}
	pipelineUID := dbPipeline.UID

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	requester, err := s.GetNamespaceByUID(ctx, requesterUID)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching requester namespace: %w", err)
	}

	pipelineRun := s.logPipelineRunStart(ctx, logPipelineRunStartParams{
		pipelineTriggerID: pipelineTriggerID,
		pipelineUID:       pipelineUID,
		pipelineReleaseID: defaultPipelineReleaseID,
		requesterUID:      requesterUID,
		userUID:           userUID,
	})
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, nil, fmt.Errorf("check trigger permission error: %w", err)
	}

	expiryRule, err := s.retentionHandler.GetExpiryRuleByNamespace(ctx, requesterUID)
	if err != nil {
		return nil, nil, fmt.Errorf("accessing expiry rule: %w", err)
	}

	err = s.preTriggerPipeline(ctx, requester, dbPipeline.Recipe, pipelineTriggerID, data, expiryRule)
	if err != nil {
		return nil, nil, err
	}

	outputs, triggerMetadata, err := s.triggerPipeline(ctx, triggerParams{
		ns:                ns,
		pipelineID:        dbPipeline.ID,
		pipelineUID:       pipelineUID,
		pipelineTriggerID: pipelineTriggerID,
		requesterUID:      requesterUID,
		userUID:           userUID,
		expiryRule:        expiryRule,
	}, returnTraces)
	if err != nil {
		return nil, nil, err
	}
	return outputs, triggerMetadata, nil
}

func (s *service) TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	requester, err := s.GetNamespaceByUID(ctx, requesterUID)
	if err != nil {
		return nil, fmt.Errorf("fetching requester namespace: %w", err)
	}

	pipelineRun := s.logPipelineRunStart(ctx, logPipelineRunStartParams{
		pipelineTriggerID: pipelineTriggerID,
		pipelineUID:       dbPipeline.UID,
		pipelineReleaseID: defaultPipelineReleaseID,
		requesterUID:      requesterUID,
		userUID:           userUID,
	})
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, err
	}

	expiryRule, err := s.retentionHandler.GetExpiryRuleByNamespace(ctx, requesterUID)
	if err != nil {
		return nil, fmt.Errorf("accessing expiry rule: %w", err)
	}

	err = s.preTriggerPipeline(ctx, requester, dbPipeline.Recipe, pipelineTriggerID, data, expiryRule)
	if err != nil {
		return nil, err
	}
	operation, err := s.triggerAsyncPipeline(ctx, triggerParams{
		ns:                ns,
		pipelineID:        dbPipeline.ID,
		pipelineUID:       dbPipeline.UID,
		pipelineTriggerID: pipelineTriggerID,
		requesterUID:      requesterUID,
		userUID:           userUID,
		expiryRule:        expiryRule,
	})
	if err != nil {
		return nil, err
	}
	return operation, nil

}

func (s *service) TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, true)
	if err != nil {
		return nil, nil, errdomain.ErrNotFound
	}
	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	requester, err := s.GetNamespaceByUID(ctx, requesterUID)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching requester namespace: %w", err)
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, nil, err
	}

	pipelineRun := s.logPipelineRunStart(ctx, logPipelineRunStartParams{
		pipelineTriggerID: pipelineTriggerID,
		pipelineUID:       pipelineUID,
		pipelineReleaseID: dbPipelineRelease.ID,
		requesterUID:      requesterUID,
		userUID:           userUID,
	})
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, nil, err
	}

	expiryRule, err := s.retentionHandler.GetExpiryRuleByNamespace(ctx, requesterUID)
	if err != nil {
		return nil, nil, fmt.Errorf("accessing expiry rule: %w", err)
	}

	err = s.preTriggerPipeline(ctx, requester, dbPipelineRelease.Recipe, pipelineTriggerID, data, expiryRule)
	if err != nil {
		return nil, nil, err
	}

	outputs, triggerMetadata, err := s.triggerPipeline(ctx, triggerParams{
		ns:                 ns,
		pipelineID:         dbPipeline.ID,
		pipelineUID:        dbPipeline.UID,
		pipelineReleaseID:  dbPipelineRelease.ID,
		pipelineReleaseUID: dbPipelineRelease.UID,
		pipelineTriggerID:  pipelineTriggerID,
		requesterUID:       requesterUID,
		userUID:            userUID,
		expiryRule:         expiryRule,
	}, returnTraces)
	if err != nil {
		return nil, nil, err
	}
	return outputs, triggerMetadata, nil
}

func (s *service) TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	requesterUID, userUID := resourcex.GetRequesterUIDAndUserUID(ctx)
	requester, err := s.GetNamespaceByUID(ctx, requesterUID)
	if err != nil {
		return nil, fmt.Errorf("fetching requester namespace: %w", err)
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, err
	}

	pipelineRun := s.logPipelineRunStart(ctx, logPipelineRunStartParams{
		pipelineTriggerID: pipelineTriggerID,
		pipelineUID:       pipelineUID,
		pipelineReleaseID: dbPipelineRelease.ID,
		requesterUID:      requesterUID,
		userUID:           userUID,
	})
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, err
	}

	expiryRule, err := s.retentionHandler.GetExpiryRuleByNamespace(ctx, requesterUID)
	if err != nil {
		return nil, fmt.Errorf("accessing expiry rule: %w", err)
	}

	err = s.preTriggerPipeline(ctx, requester, dbPipelineRelease.Recipe, pipelineTriggerID, data, expiryRule)
	if err != nil {
		return nil, err
	}
	operation, err := s.triggerAsyncPipeline(ctx, triggerParams{
		ns:                 ns,
		pipelineID:         dbPipeline.ID,
		pipelineUID:        dbPipeline.UID,
		pipelineReleaseID:  dbPipelineRelease.ID,
		pipelineReleaseUID: dbPipelineRelease.UID,
		pipelineTriggerID:  pipelineTriggerID,
		requesterUID:       requesterUID,
		userUID:            userUID,
		expiryRule:         expiryRule,
	})
	if err != nil {
		return nil, err
	}
	return operation, nil
}

func (s *service) GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")

	if err != nil {
		return nil, err
	}
	return s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) getOperationFromWorkflowInfo(ctx context.Context, workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:

		pipelineTriggerID := workflowExecutionInfo.Execution.WorkflowId
		defer func() {
			_ = s.memory.PurgeWorkflowMemory(ctx, pipelineTriggerID)
		}()

		outputs, metadata, err := s.getOutputsAndMetadata(ctx, pipelineTriggerID, true)
		if err != nil {
			return nil, err
		}

		pipelineResp := &pipelinepb.TriggerNamespacePipelineResponse{
			Outputs:  outputs,
			Metadata: metadata,
		}

		resp, err := anypb.New(pipelineResp)
		if err != nil {
			return nil, err
		}
		resp.TypeUrl = "buf.build/instill-ai/protobufs/pipeline.pipeline.v1beta.TriggerNamespacePipelineResponse"
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Response{
				Response: resp,
			},
		}

	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
	case enums.WORKFLOW_EXECUTION_STATUS_CONTINUED_AS_NEW:
		operation = longrunningpb.Operation{
			Done: false,
			Result: &longrunningpb.Operation_Response{
				Response: &anypb.Any{},
			},
		}
	default:
		operation = longrunningpb.Operation{
			Done: true,
			Result: &longrunningpb.Operation_Error{
				Error: &rpcstatus.Status{
					Code:    int32(workflowExecutionInfo.Status),
					Details: []*anypb.Any{},
					Message: "",
				},
			},
		}
	}

	operation.Name = fmt.Sprintf("operations/%s", workflowExecutionInfo.Execution.WorkflowId)
	return &operation, nil
}
