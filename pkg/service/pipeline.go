package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/PaesslerAG/jsonpath"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcstatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/pipeline-backend/pkg/component/base"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
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
	if err := s.setSchedulePipeline(ctx, ns, dbPipeline.ID, "", dbPipeline.UID, uuid.Nil, dbPipeline.Recipe); err != nil {
		return nil, err
	}

	if err := s.repository.CreateNamespacePipeline(ctx, dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, dbPipeline.ID, false, true)
	if err != nil {
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

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, uid, view <= pipelinepb.Pipeline_VIEW_BASIC, true)
	if err != nil {
		return nil, err
	}

	return s.converter.ConvertPipelineToPB(ctx, dbPipeline, view, true, true)

}

func (s *service) setSchedulePipeline(ctx context.Context, ns resource.Namespace, pipelineID, pipelineReleaseID string, pipelineUID, releaseUID uuid.UUID, recipe *datamodel.Recipe) error {
	// TODO This check could be removed, as the receiver should be initialized
	// at this point. However, some tests depend on it, so we would need to
	// either mock this interface or (better) communicate with Temporal through
	// our own interface.
	if s.temporalClient == nil {
		return nil
	}

	crons := []string{}
	if recipe != nil && recipe.On != nil && recipe.On.Schedule != nil {
		for _, v := range recipe.On.Schedule {
			crons = append(crons, v.Cron)
		}
	}

	scheduleID := fmt.Sprintf("%s_%s_schedule", pipelineUID, releaseUID)

	handle := s.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	_ = handle.Delete(ctx)

	if len(crons) > 0 {

		param := &worker.SchedulePipelineWorkflowParam{
			Namespace:          ns,
			PipelineID:         pipelineID,
			PipelineUID:        pipelineUID,
			PipelineReleaseID:  pipelineReleaseID,
			PipelineReleaseUID: releaseUID,
		}
		_, err := s.temporalClient.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: scheduleID,
			Spec: client.ScheduleSpec{
				CronExpressions: crons,
			},
			Action: &client.ScheduleWorkflowAction{
				Args:      []any{param},
				ID:        scheduleID,
				Workflow:  "SchedulePipelineWorkflow",
				TaskQueue: worker.TaskQueue,
				RetryPolicy: &temporal.RetryPolicy{
					MaximumAttempts: 1,
				},
			},
		})

		if err != nil {
			return err
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
	if existingPipeline, _ = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, false); existingPipeline == nil {
		return nil, err
	}

	if existingPipeline.ShareCode == "" {
		dbPipeline.ShareCode = generateShareCode()
	}

	if err := s.setSchedulePipeline(ctx, ns, dbPipeline.ID, "", dbPipeline.UID, uuid.Nil, dbPipeline.Recipe); err != nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, dbPipeline.UID, dbPipeline); err != nil {
		return nil, err
	}

	toUpdTags := toUpdPipeline.GetTags()

	currentTags := existingPipeline.TagNames()
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

func (s *service) preTriggerPipeline(ctx context.Context, ns resource.Namespace, r *datamodel.Recipe, pipelineTriggerID string, pipelineData []*pipelinepb.TriggerData) error {
	batchSize := len(pipelineData)
	if batchSize > constant.MaxBatchSize {
		return ErrExceedMaxBatchSize
	}

	var metadata []byte

	instillFormatMap := map[string]string{}

	schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	schStruct.Fields["type"] = structpb.NewStringValue("object")
	for k, v := range r.Variable {
		v.InstillFormat = utils.ConvertInstillFormat(v.InstillFormat)
		instillFormatMap[k] = v.InstillFormat
	}

	b, _ := json.Marshal(r.Variable)
	properties := &structpb.Struct{}
	_ = protojson.Unmarshal(b, properties)
	schStruct.Fields["properties"] = structpb.NewStructValue(properties)
	err := componentbase.CompileInstillAcceptFormats(schStruct)
	if err != nil {
		return err
	}
	err = componentbase.CompileInstillFormat(schStruct)
	if err != nil {
		return err
	}
	metadata, err = protojson.Marshal(schStruct)
	if err != nil {
		return err
	}

	c := jsonschema.NewCompiler()
	c.RegisterExtension("instillAcceptFormats", componentbase.InstillAcceptFormatsMeta, componentbase.InstillAcceptFormatsCompiler{})
	c.RegisterExtension("instillFormat", componentbase.InstillFormatMeta, componentbase.InstillFormatCompiler{})

	if err := c.AddResource("schema.json", strings.NewReader(string(metadata))); err != nil {
		return err
	}

	sch, err := c.Compile("schema.json")

	if err != nil {
		return err
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
			switch s := m[k].(type) {
			case string:
				if instillFormatMap[k] != "string" {
					if !strings.HasPrefix(s, "data:") {
						b, err := base64.StdEncoding.DecodeString(s)
						if err != nil {
							return fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
						}
						mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]
						vars.Fields[k] = structpb.NewStringValue(fmt.Sprintf("data:%s;base64,%s", mimeType, s))
					}
				}
			case []string:
				if instillFormatMap[k] != "array:string" {
					for idx := range s {
						if !strings.HasPrefix(s[idx], "data:") {
							b, err := base64.StdEncoding.DecodeString(s[idx])
							if err != nil {
								return fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
							}
							mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]
							vars.Fields[k].GetListValue().GetValues()[idx] = structpb.NewStringValue(fmt.Sprintf("data:%s;base64,%s", mimeType, s[idx]))
						}

					}
				}
			}
		}

		if err = sch.Validate(m); err != nil {
			e := err.(*jsonschema.ValidationError)

			for _, valErr := range e.DetailedOutput().Errors {
				inputPath := fmt.Sprintf("%s/%d", "inputs", idx)
				componentbase.FormatErrors(inputPath, valErr, &errors)
				for _, subValErr := range valErr.Errors {
					componentbase.FormatErrors(inputPath, subValErr, &errors)
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

	formats := map[string][]string{}
	for k, v := range instillFormatMap {
		formats[k] = []string{v}
	}

	for idx, d := range pipelineData {

		// TODO: refactor array parser
		variable := data.NewMap(nil)
		for k, v := range d.Variable.Fields {
			if _, ok := instillFormatMap[k]; !ok {
				continue
			}
			switch instillFormatMap[k] {
			case "boolean":
				variable.Fields[k] = data.NewBoolean(v.GetBoolValue())
			case "array:boolean":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx] = data.NewBoolean(val.GetBoolValue())
				}
				variable.Fields[k] = array
			case "string":
				variable.Fields[k] = data.NewString(v.GetStringValue())
			case "array:string":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx] = data.NewString(val.GetStringValue())
				}
				variable.Fields[k] = array
			case "integer":
				variable.Fields[k] = data.NewNumberFromFloat(v.GetNumberValue())
			case "array:integer":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx] = data.NewNumberFromFloat(val.GetNumberValue())
				}
				variable.Fields[k] = array
			case "number":
				variable.Fields[k] = data.NewNumberFromFloat(v.GetNumberValue())
			case "array:number":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx] = data.NewNumberFromFloat(val.GetNumberValue())
				}
				variable.Fields[k] = array
			case "image", "image/*":
				variable.Fields[k], err = data.NewImageFromURL(v.GetStringValue())
				if err != nil {
					return err
				}
			case "array:image", "array:image/*":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx], err = data.NewImageFromURL(val.GetStringValue())
					if err != nil {
						return err
					}
				}
				variable.Fields[k] = array
			case "audio", "audio/*":
				variable.Fields[k], err = data.NewAudioFromURL(v.GetStringValue())
				if err != nil {
					return err
				}
			case "array:audio", "array:audio/*":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx], err = data.NewAudioFromURL(val.GetStringValue())
					if err != nil {
						return err
					}
				}
				variable.Fields[k] = array
			case "video", "video/*":
				variable.Fields[k], err = data.NewVideoFromURL(v.GetStringValue())
				if err != nil {
					return err
				}
			case "array:video", "array:video/*":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))

				for idx, val := range v.GetListValue().Values {
					array.Values[idx], err = data.NewVideoFromURL(val.GetStringValue())
					if err != nil {
						return err
					}
				}
				variable.Fields[k] = array
			case "document", "file", "*/*":
				variable.Fields[k], err = data.NewDocumentFromURL(v.GetStringValue())
				if err != nil {
					return err
				}
			case "array:document", "array:file", "array:*/*":
				array := data.NewArray(make([]data.Value, len(v.GetListValue().Values)))
				for idx, val := range v.GetListValue().Values {
					array.Values[idx], err = data.NewDocumentFromURL(val.GetStringValue())
					if err != nil {
						return err
					}
				}
				variable.Fields[k] = array
			case "semi-structured/*", "semi-structured/json", "json":

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
					variable.Fields[k] = jv
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
					variable.Fields[k] = jv
				}

			}
			if err != nil {
				return err
			}
		}
		err = wfm.Set(ctx, idx, constant.SegVariable, variable)
		if err != nil {
			return err
		}

		secret := data.NewMap(nil)
		for k, v := range d.Secret {
			secret.Fields[k] = data.NewString(v)
		}
		err = wfm.Set(ctx, idx, constant.SegSecret, secret)
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

	dbPipelineReleaseToCreate.RecipeYAML = dbPipeline.RecipeYAML
	dbPipelineReleaseToCreate.Metadata = dbPipeline.Metadata

	if err := s.repository.CreateNamespacePipelineRelease(ctx, ownerPermalink, pipelineUID, dbPipelineReleaseToCreate); err != nil {
		return nil, err
	}

	dbCreatedPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, pipelineRelease.Id, false)
	if err != nil {
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
	if existingPipeline, err = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUID, false, true); err != nil {
		return err
	}
	existingPipeline.Recipe = dbPipelineRelease.Recipe

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, existingPipeline.UID, existingPipeline); err != nil {
		return err
	}

	return nil
}

// TODO: share the code with worker/workflow.go
func (s *service) triggerPipeline(
	ctx context.Context,
	ns resource.Namespace,
	r *datamodel.Recipe,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineData []*pipelinepb.TriggerData,
	pipelineTriggerID string,
	returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	logger, _ := logger.GetZapLogger(ctx)

	defer func() {
		_ = s.memory.PurgeWorkflowMemory(ctx, pipelineTriggerID)
	}()
	err := s.preTriggerPipeline(ctx, ns, r, pipelineTriggerID, pipelineData)
	if err != nil {
		return nil, nil, err
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	requesterUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	if requesterUID.IsNil() {
		requesterUID = userUID
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			TriggerFromAPI: true,
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    pipelineTriggerID,
				PipelineID:           pipelineID,
				PipelineUID:          pipelineUID,
				PipelineReleaseID:    pipelineReleaseID,
				PipelineReleaseUID:   pipelineReleaseUID,
				PipelineOwnerType:    ns.NsType,
				PipelineOwnerUID:     ns.NsUID,
				PipelineUserUID:      userUID,
				PipelineRequesterUID: requesterUID,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
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

	return s.getOutputsAndMetadata(ctx, pipelineTriggerID, returnTraces)
}

func (s *service) triggerAsyncPipeline(
	ctx context.Context,
	ns resource.Namespace,
	r *datamodel.Recipe,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineData []*pipelinepb.TriggerData,
	pipelineTriggerID string,
	returnTraces bool) (*longrunningpb.Operation, error) {

	defer func() {
		go func() {
			// We only retain the memory for a maximum of 60 minutes.
			time.Sleep(60 * time.Minute)
			_ = s.memory.PurgeWorkflowMemory(ctx, pipelineTriggerID)
		}()
	}()
	err := s.preTriggerPipeline(ctx, ns, r, pipelineTriggerID, pipelineData)
	if err != nil {
		return nil, err
	}

	logger, _ := logger.GetZapLogger(ctx)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerID,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
	}

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	requesterUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
	if requesterUID.IsNil() {
		requesterUID = userUID
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    pipelineTriggerID,
				PipelineID:           pipelineID,
				PipelineUID:          pipelineUID,
				PipelineReleaseID:    pipelineReleaseID,
				PipelineReleaseUID:   pipelineReleaseUID,
				PipelineOwnerType:    ns.NsType,
				PipelineOwnerUID:     ns.NsUID,
				PipelineUserUID:      userUID,
				PipelineRequesterUID: requesterUID,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
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
		subCtx := context.Background()

		err = we.Get(subCtx, nil)
		if err != nil {
			err = fmt.Errorf("%w:%w", ErrTriggerFail, err)

			var applicationErr *temporal.ApplicationError
			if errors.As(err, &applicationErr) && applicationErr.Message() != "" {
				err = errmsg.AddMessage(err, applicationErr.Message())
			}
			logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))

			run, repoErr := s.repository.GetPipelineRunByUID(subCtx, uuid.FromStringOrNil(pipelineTriggerID))
			if repoErr != nil {
				logger.Error("failed to log pipeline run error", zap.Error(err), zap.Error(repoErr))
				return
			}

			s.logPipelineRunError(subCtx, pipelineTriggerID, err, run.StartedTime)
			return
		}
	})

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", pipelineTriggerID),
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

func (s *service) CheckPipelineEventCode(ctx context.Context, ns resource.Namespace, id string, code string) (bool, error) {
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ns.Permalink(), id, false, true)
	if err != nil {
		return false, errdomain.ErrNotFound
	}

	return dbPipeline.ShareCode == code, nil
}

func (s *service) HandleNamespacePipelineEventByID(ctx context.Context, ns resource.Namespace, id string, eventID string, data *structpb.Struct, pipelineTriggerID string) (*structpb.Struct, error) {

	var targetType string
	ownerPermalink := ns.Permalink()
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	// TODO: In the webhook event, the request is sent by a system user, not an
	// end user. It doesn't include the user UID or requester UID. For now,
	// we'll use the namespace as the user ID and requester UID.
	// A proper authentication mechanism for system users needs to be designed.
	md, _ := metadata.FromIncomingContext(ctx)
	md.Set(constant.HeaderUserUIDKey, ns.NsUID.String())
	md.Set(constant.HeaderRequesterUIDKey, ns.NsUID.String())
	ctx = metadata.NewIncomingContext(ctx, md)

	pipelineRun := s.logPipelineRunStart(ctx, pipelineTriggerID, dbPipeline.UID, defaultPipelineReleaseID)
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if e, ok := dbPipeline.Recipe.On.Event[eventID]; ok {

		targetType = e.Type
	} else {
		return nil, fmt.Errorf("eventID not correct")
	}

	isVerificationEvent, out, err := s.component.HandleVerificationEvent(targetType, md, data, nil)
	if err != nil {
		return nil, err
	}
	if isVerificationEvent {
		return out, nil
	}

	d := pipelinepb.TriggerData{
		Variable: &structpb.Struct{Fields: make(map[string]*structpb.Value)},
	}

	jsonInput := map[string]any{}
	b, err := protojson.Marshal(data)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &jsonInput)
	if err != nil {
		return nil, err
	}

	for key, v := range dbPipeline.Recipe.Variable {
		for _, l := range v.Listen {
			l := l[2 : len(l)-1]
			s := strings.Split(l, ".")
			if s[0] != "on" || s[1] != "event" {
				return nil, fmt.Errorf("cannot listen to data outside of `on.event`")
			}
			if eventID == s[2] {
				path := strings.Join(s[4:], ".")
				res, err := jsonpath.Get(fmt.Sprintf("$.%s", path), jsonInput)
				if err != nil {
					return nil, err
				}
				d.Variable.Fields[key], err = structpb.NewValue(res)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	triggerData := []*pipelinepb.TriggerData{&d}

	_, err = s.triggerAsyncPipeline(ctx, ns, dbPipeline.Recipe, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, triggerData, pipelineTriggerID, false)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *service) TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {
	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, nil, errdomain.ErrNotFound
	}
	pipelineUID := dbPipeline.UID

	pipelineRun := s.logPipelineRunStart(ctx, pipelineTriggerID, pipelineUID, defaultPipelineReleaseID)
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, nil, fmt.Errorf("check trigger permission error: %w", err)
	}

	outputs, triggerMetadata, err := s.triggerPipeline(ctx, ns, dbPipeline.Recipe, dbPipeline.ID, pipelineUID, "", uuid.Nil, data, pipelineTriggerID, returnTraces)
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

	pipelineRun := s.logPipelineRunStart(ctx, pipelineTriggerID, dbPipeline.UID, defaultPipelineReleaseID)
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, err
	}

	operation, err := s.triggerAsyncPipeline(ctx, ns, dbPipeline.Recipe, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, data, pipelineTriggerID, returnTraces)
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

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, nil, err
	}

	pipelineRun := s.logPipelineRunStart(ctx, pipelineTriggerID, pipelineUID, dbPipelineRelease.ID)
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, nil, err
	}

	outputs, triggerMetadata, err := s.triggerPipeline(ctx, ns, dbPipelineRelease.Recipe, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, data, pipelineTriggerID, returnTraces)
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

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, err
	}

	pipelineRun := s.logPipelineRunStart(ctx, pipelineTriggerID, pipelineUID, dbPipelineRelease.ID)
	defer func() {
		if err != nil {
			s.logPipelineRunError(ctx, pipelineTriggerID, err, pipelineRun.StartedTime)
		}
	}()

	if err = s.checkTriggerPermission(ctx, dbPipeline); err != nil {
		return nil, err
	}

	operation, err := s.triggerAsyncPipeline(ctx, ns, dbPipelineRelease.Recipe, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, data, pipelineTriggerID, returnTraces)
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
		resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1beta.TriggerNamespacePipelineResponse"
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

func (s *service) ListPipelineRuns(ctx context.Context, req *pipelinepb.ListPipelineRunsRequest, filter filtering.Filter) (*pipelinepb.ListPipelineRunsResponse, error) {
	ns, err := s.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	log, _ := logger.GetZapLogger(ctx)

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ns.Permalink(), req.GetPipelineId(), true, false)
	if err != nil {
		return nil, err
	}

	requesterUID, _ := utils.GetRequesterUIDAndUserUID(ctx)
	page := s.pageInRange(req.GetPage())
	pageSize := s.pageSizeInRange(req.GetPageSize())

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}

	isOwner := dbPipeline.OwnerUID().String() == requesterUID

	pipelineRuns, totalCount, err := s.repository.GetPaginatedPipelineRunsWithPermissions(ctx, requesterUID, dbPipeline.UID.String(),
		page, pageSize, filter, orderBy, isOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline runs: %w", err)
	}

	var referenceIDs []string
	for _, pipelineRun := range pipelineRuns {
		if CanViewPrivateData(pipelineRun.Namespace, requesterUID) {
			for _, input := range pipelineRun.Inputs {
				referenceIDs = append(referenceIDs, input.Name)
			}
			for _, output := range pipelineRun.Outputs {
				referenceIDs = append(referenceIDs, output.Name)
			}
			for _, reference := range pipelineRun.RecipeSnapshot {
				referenceIDs = append(referenceIDs, reference.Name)
			}
		}
	}

	log.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		log.Error("failed to get files from minio", zap.Error(err))
	}

	metadataMap := make(map[string][]byte)
	for _, content := range fileContents {
		metadataMap[content.Name] = content.Content
	}

	requesterIDMap := make(map[string]struct{})
	for _, pipelineRun := range pipelineRuns {
		requesterIDMap[pipelineRun.TriggeredBy] = struct{}{}
	}

	runnerMap := make(map[string]*string)
	for requesterID := range requesterIDMap {
		runner, err := s.mgmtPrivateServiceClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{Uid: requesterID})
		if err != nil {
			return nil, err
		}
		log.Info("CheckNamespaceByUIDAdmin finished", zap.String("RequesterUID", requesterID), zap.String("runnerId", runner.Id))
		runnerMap[requesterID] = &runner.Id
	}

	// Convert datamodel.PipelineRun to pb.PipelineRun
	pbPipelineRuns := make([]*pipelinepb.PipelineRun, len(pipelineRuns))
	for i, run := range pipelineRuns {
		pbRun, err := s.convertPipelineRunToPB(run)
		if err != nil {
			return nil, fmt.Errorf("failed to convert pipeline run: %w", err)
		}
		pbRun.RunnerId = runnerMap[run.TriggeredBy]

		if CanViewPrivateData(run.Namespace, requesterUID) {
			if len(run.Inputs) == 1 {
				key := run.Inputs[0].Name
				pbRun.Inputs, err = parseMetadataToStructArray(metadataMap, key)
				if err != nil {
					log.Error("Failed to load input metadata", zap.Error(err), zap.String("pipelineUID", run.PipelineUID.String()),
						zap.String("inputReferenceID", key))
				}
			}

			if len(run.Outputs) == 1 {
				key := run.Outputs[0].Name
				pbRun.Outputs, err = parseMetadataToStructArray(metadataMap, key)
				if err != nil {
					log.Error("Failed to load output metadata", zap.Error(err), zap.String("pipelineUID", run.PipelineUID.String()),
						zap.String("outputReferenceID", key))
				}
			}

			if len(run.RecipeSnapshot) == 1 {
				key := run.RecipeSnapshot[0].Name
				pbRun.RecipeSnapshot, pbRun.DataSpecification, err = parseRecipeMetadata(ctx, metadataMap, s.converter, key)
				if err != nil {
					log.Error("Failed to load recipe snapshot", zap.Error(err), zap.String("pipelineUID", run.PipelineUID.String()),
						zap.String("recipeReferenceID", key))
				}
			}
		}

		pbPipelineRuns[i] = pbRun
	}

	return &pipelinepb.ListPipelineRunsResponse{
		PipelineRuns: pbPipelineRuns,
		TotalSize:    int32(totalCount),
		Page:         int32(page),
		PageSize:     int32(pageSize),
	}, nil
}

func (s *service) ListComponentRuns(ctx context.Context, req *pipelinepb.ListComponentRunsRequest, filter filtering.Filter) (*pipelinepb.ListComponentRunsResponse, error) {
	page := s.pageInRange(req.GetPage())
	pageSize := s.pageSizeInRange(req.GetPageSize())
	requesterUID, _ := utils.GetRequesterUIDAndUserUID(ctx)

	log, _ := logger.GetZapLogger(ctx)

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}

	dbPipelineRun, err := s.repository.GetPipelineRunByUID(ctx, uuid.FromStringOrNil(req.GetPipelineRunId()))
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline run by run ID: %s. error: %s", req.GetPipelineRunId(), err.Error())
	}
	dbPipeline, err := s.repository.GetPipelineByUID(ctx, dbPipelineRun.PipelineUID, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline by UID: %s. error: %s", dbPipelineRun.PipelineUID.String(), err.Error())
	}

	isOwner := dbPipeline.OwnerUID().String() == requesterUID

	if !isOwner && requesterUID != dbPipelineRun.Namespace {
		return nil, fmt.Errorf("requester is not pipeline owner/credit owner. they are not allowed to view these component runs")
	}

	componentRuns, totalCount, err := s.repository.GetPaginatedComponentRunsByPipelineRunIDWithPermissions(ctx, req.GetPipelineRunId(), page, pageSize, filter, orderBy)
	if err != nil {
		return nil, fmt.Errorf("failed to get component runs: %w", err)
	}

	var referenceIDs []string
	for _, pipelineRun := range componentRuns {
		if CanViewPrivateData(dbPipelineRun.Namespace, requesterUID) {
			for _, input := range pipelineRun.Inputs {
				referenceIDs = append(referenceIDs, input.Name)
			}
			for _, output := range pipelineRun.Outputs {
				referenceIDs = append(referenceIDs, output.Name)
			}
		}
	}

	log.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		log.Error("failed to get files from minio", zap.Error(err))
	}

	metadataMap := make(map[string][]byte)
	for _, content := range fileContents {
		metadataMap[content.Name] = content.Content
	}

	// Convert datamodel.ComponentRun to pb.ComponentRun
	pbComponentRuns := make([]*pipelinepb.ComponentRun, len(componentRuns))
	for i, run := range componentRuns {
		pbRun, err := s.convertComponentRunToPB(run)
		if err != nil {
			return nil, fmt.Errorf("failed to convert component run: %w", err)
		}

		if CanViewPrivateData(dbPipelineRun.Namespace, requesterUID) {
			if len(run.Inputs) == 1 {
				key := run.Inputs[0].Name
				pbRun.Inputs, err = parseMetadataToStructArray(metadataMap, key)
				if err != nil {
					log.Error("Failed to load input metadata", zap.Error(err), zap.String("ComponentID", run.ComponentID),
						zap.String("inputReferenceID", key))
				}
			}
			if len(run.Outputs) == 1 {
				key := run.Outputs[0].Name
				pbRun.Outputs, err = parseMetadataToStructArray(metadataMap, key)
				if err != nil {
					log.Error("Failed to load output metadata", zap.Error(err), zap.String("ComponentID", run.ComponentID),
						zap.String("outputReferenceID", key))
				}

			}
		}
		pbComponentRuns[i] = pbRun
	}

	return &pipelinepb.ListComponentRunsResponse{
		ComponentRuns: pbComponentRuns,
		TotalSize:     int32(totalCount),
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

func (s *service) ListPipelineRunsByRequester(ctx context.Context, req *pipelinepb.ListPipelineRunsByCreditOwnerRequest) (*pipelinepb.ListPipelineRunsByCreditOwnerResponse, error) {
	page := s.pageInRange(req.GetPage())
	pageSize := s.pageSizeInRange(req.GetPageSize())
	requesterUID, _ := utils.GetRequesterUIDAndUserUID(ctx)

	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("status", filtering.TypeString),
		filtering.DeclareIdent("source", filtering.TypeString),
	}...)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	startedTimeBegin := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if req.GetStart().IsValid() {
		startedTimeBegin = req.GetStart().AsTime()
	}
	startedTimeEnd := now
	if req.GetStop().IsValid() {
		startedTimeEnd = req.GetStop().AsTime()
	}

	if startedTimeBegin.After(startedTimeEnd) {
		return nil, fmt.Errorf("time range end time is earlier than start time")
	}

	pipelineRuns, totalCount, err := s.repository.GetPaginatedPipelineRunsByRequester(ctx, repository.GetPipelineRunsByRequesterParams{
		RequesterUID:   requesterUID,
		StartTimeBegin: startedTimeBegin,
		StartTimeEnd:   startedTimeEnd,
		Page:           page,
		PageSize:       pageSize,
		Filter:         filter,
		Order:          orderBy,
	})
	if err != nil {
		return nil, fmt.Errorf("getting pipeline runs by requester: %w", err)
	}

	requesterIDMap := make(map[string]struct{})
	for _, pipelineRun := range pipelineRuns {
		requesterIDMap[pipelineRun.TriggeredBy] = struct{}{}
	}

	runnerMap := make(map[string]*string)
	for requesterID := range requesterIDMap {
		runner, err := s.mgmtPrivateServiceClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{Uid: requesterID})
		if err != nil {
			return nil, err
		}
		runnerMap[requesterID] = &runner.Id
	}

	pbPipelineRuns := make([]*pipelinepb.PipelineRun, len(pipelineRuns))

	var pbRun *pipelinepb.PipelineRun
	for i, run := range pipelineRuns {
		pbRun, err = s.convertPipelineRunToPB(run)
		if err != nil {
			return nil, fmt.Errorf("converting pipeline run: %w", err)
		}
		pbRun.RunnerId = runnerMap[run.TriggeredBy]
		pbPipelineRuns[i] = pbRun
	}

	return &pipelinepb.ListPipelineRunsByCreditOwnerResponse{
		PipelineRuns: pbPipelineRuns,
		TotalSize:    int32(totalCount),
		Page:         int32(page),
		PageSize:     int32(pageSize),
	}, nil
}
