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
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/guregu/null.v4"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"

	"github.com/instill-ai/x/errmsg"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"

	componentbase "github.com/instill-ai/component/base"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
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
		NumberOfPublicPipelines:   int32(hubStats.NumberOfPublicPipelines),
		NumberOfFeaturedPipelines: int32(hubStats.NumberOfFeaturedPipelines),
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

	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, int64(pageSize), pageToken, view <= pipelinepb.Pipeline_VIEW_BASIC, filter, uidAllowList, showDeleted, true, order)
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
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			return nil, errdomain.ErrUnauthorized
		}
	}

	dbPipeline, err := s.converter.ConvertPipelineToDB(ctx, ns, pbPipeline)
	if err != nil {
		return nil, err
	}
	if err := s.checkSecret(ctx, dbPipeline.Recipe.Component); err != nil {
		return nil, err
	}

	dbPipeline.ShareCode = generateShareCode()
	if err := s.setSchedulePipeline(ctx, ns, dbPipeline); err != nil {
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

func (s *service) setSchedulePipeline(ctx context.Context, ns resource.Namespace, dbPipeline *datamodel.Pipeline) error {

	if s.temporalClient == nil {
		return nil
	}
	crons := []string{}
	if dbPipeline.Recipe != nil && dbPipeline.Recipe.On != nil && dbPipeline.Recipe.On.Schedule != nil {
		for _, v := range dbPipeline.Recipe.On.Schedule {
			crons = append(crons, v.Cron)
		}
	}

	scheduleID := fmt.Sprintf("%s_schedule", dbPipeline.UID)

	handle := s.temporalClient.ScheduleClient().GetHandle(ctx, scheduleID)
	_ = handle.Delete(ctx)

	if len(crons) > 0 {

		param := &worker.SchedulePipelineWorkflowParam{
			Namespace:  ns,
			PipelineID: dbPipeline.ID,
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
	if err := s.checkSecret(ctx, dbPipeline.Recipe.Component); err != nil {
		return nil, err
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

	dbPipeline.ShareCode = generateShareCode()
	if err := s.setSchedulePipeline(ctx, ns, dbPipeline); err != nil {
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

func (s *service) generateCloneTargetNamespace(ctx context.Context, target string) (resource.Namespace, error) {
	targetNamespace, targetID, ok := strings.Cut(target, "/")
	if !ok {
		return resource.Namespace{}, errdomain.ErrInvalidCloneTarget
	}

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
			NsID:   targetID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}
	} else if resp.Type == mgmtpb.CheckNamespaceAdminResponse_NAMESPACE_ORGANIZATION {
		targetNS = resource.Namespace{
			NsType: resource.Organization,
			NsID:   targetID,
			NsUID:  uuid.FromStringOrNil(resp.Uid),
		}
	} else {
		return resource.Namespace{}, errdomain.ErrInvalidCloneTarget
	}

	return targetNS, nil
}

func (s *service) CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, id string, target string, description string, sharing *pipelinepb.Sharing) (*pipelinepb.Pipeline, error) {
	sourcePipeline, err := s.GetNamespacePipelineByID(ctx, ns, id, pipelinepb.Pipeline_VIEW_RECIPE)
	if err != nil {
		return nil, err
	}
	targetNS, err := s.generateCloneTargetNamespace(ctx, target)
	if err != nil {
		return nil, err
	}

	newPipeline := &pipelinepb.Pipeline{
		Id:          targetNS.NsID,
		Description: &description,
		Sharing:     sharing,
		Recipe:      sourcePipeline.Recipe,
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

func (s *service) CloneNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, target string, description string, sharing *pipelinepb.Sharing) (*pipelinepb.Pipeline, error) {
	sourcePipelineRelease, err := s.GetNamespacePipelineReleaseByID(ctx, ns, pipelineUID, id, pipelinepb.Pipeline_VIEW_RECIPE)
	if err != nil {
		return nil, err
	}
	targetNS, err := s.generateCloneTargetNamespace(ctx, target)
	if err != nil {
		return nil, err
	}

	newPipeline := &pipelinepb.Pipeline{
		Id:          targetNS.NsID,
		Description: &description,
		Sharing:     sharing,
		Recipe:      sourcePipelineRelease.Recipe,
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

func (s *service) ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) ([]*pipelinepb.PipelineValidationError, error) {

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

func (s *service) preTriggerPipeline(ctx context.Context, isAdmin bool, ns resource.Namespace, r *datamodel.Recipe, pipelineTriggerID string, pipelineData []*pipelinepb.TriggerData) (*recipe.BatchMemoryKey, error) {

	batchSize := len(pipelineData)
	if batchSize > constant.MaxBatchSize {
		return nil, ErrExceedMaxBatchSize
	}

	var metadata []byte

	instillFormatMap := map[string]string{}

	schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	schStruct.Fields["type"] = structpb.NewStringValue("object")
	b, _ := json.Marshal(r.Variable)
	properties := &structpb.Struct{}
	_ = protojson.Unmarshal(b, properties)
	schStruct.Fields["properties"] = structpb.NewStructValue(properties)
	for k, v := range r.Variable {
		instillFormatMap[k] = v.InstillFormat
	}
	err := componentbase.CompileInstillAcceptFormats(schStruct)
	if err != nil {
		return nil, err
	}
	err = componentbase.CompileInstillFormat(schStruct)
	if err != nil {
		return nil, err
	}
	metadata, err = protojson.Marshal(schStruct)
	if err != nil {
		return nil, err
	}

	c := jsonschema.NewCompiler()
	c.RegisterExtension("instillAcceptFormats", componentbase.InstillAcceptFormatsMeta, componentbase.InstillAcceptFormatsCompiler{})
	c.RegisterExtension("instillFormat", componentbase.InstillFormatMeta, componentbase.InstillFormatCompiler{})

	if err := c.AddResource("schema.json", strings.NewReader(string(metadata))); err != nil {
		return nil, err
	}

	sch, err := c.Compile("schema.json")

	if err != nil {
		return nil, err
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
							return nil, fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
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
								return nil, fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
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
		return nil, fmt.Errorf("[Pipeline Trigger Data Error] %s", strings.Join(errors, "; "))
	}

	memory := make([]*recipe.Memory, len(pipelineData))
	for idx := range pipelineData {
		memory[idx] = &recipe.Memory{
			Variable:  make(recipe.VariableMemory),
			Secret:    make(recipe.SecretMemory),
			Component: make(map[string]*recipe.ComponentMemory),
		}
	}

	for idx, data := range pipelineData {
		varJSONBytes, err := protojson.Marshal(data.Variable)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(varJSONBytes, &memory[idx].Variable)
		if err != nil {
			return nil, err
		}
		if memory[idx].Variable == nil { // todo: this is not needed?
			memory[idx].Variable = make(recipe.VariableMemory)
		}

		secretJSONBytes, err := json.Marshal(data.Secret)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(secretJSONBytes, &memory[idx].Secret)
		if err != nil {
			return nil, err
		}
		if memory[idx].Secret == nil {
			memory[idx].Secret = make(recipe.SecretMemory)
		}
	}
	pt := ""
	// TODO: We should only query the needed key.
	for {
		var nsSecrets []*datamodel.Secret
		// TODO: should use ctx user uid
		nsSecrets, _, pt, err = s.repository.ListNamespaceSecrets(ctx, ns.Permalink(), 100, pt, filtering.Filter{})
		if err != nil {
			return nil, err
		}

		for _, nsSecret := range nsSecrets {
			if nsSecret.Value != nil {
				for idx := range pipelineData {
					if _, ok := memory[idx].Secret[nsSecret.ID]; !ok {
						memory[idx].Secret[nsSecret.ID] = *nsSecret.Value
					}
				}
			}
		}

		if pt == "" {
			break
		}
	}

	k, err := recipe.Write(ctx, s.redisClient, pipelineTriggerID, r, memory, ns.Permalink())
	if err != nil {
		return nil, err
	}

	return k, nil
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
	isAdmin bool,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineData []*pipelinepb.TriggerData,
	pipelineTriggerID string,
	returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	logger, _ := logger.GetZapLogger(ctx)

	memoryKey, err := s.preTriggerPipeline(ctx, isAdmin, ns, r, pipelineTriggerID, pipelineData)
	if err != nil {
		return nil, nil, err
	}
	defer recipe.Purge(ctx, s.redisClient, pipelineTriggerID)

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

	runSource := datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API)
	userAgentValue, ok := runpb.RunSource_value[resource.GetRequestSingleHeader(ctx, constant.HeaderUserAgentKey)]
	if ok {
		runSource = datamodel.RunSource(userAgentValue)
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			BatchSize:        len(pipelineData),
			MemoryStorageKey: memoryKey,
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    pipelineTriggerID,
				PipelineID:           pipelineID,
				PipelineUID:          pipelineUID,
				PipelineReleaseID:    pipelineReleaseID,
				PipelineReleaseUID:   pipelineReleaseUID,
				PipelineRecipe:       r,
				PipelineOwnerType:    ns.NsType,
				PipelineOwnerUID:     ns.NsUID,
				PipelineUserUID:      userUID,
				PipelineRequesterUID: requesterUID,
				PipelineRunSource:    runSource,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
			},
			Mode: mgmtpb.Mode_MODE_SYNC,
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

		run, repoErr := s.repository.GetPipelineRunByUID(ctx, uuid.FromStringOrNil(pipelineTriggerID))
		if repoErr != nil {
			logger.Error("failed to log pipeline run error", zap.Error(err), zap.Error(repoErr))
			return nil, nil, err
		}

		now := time.Now()
		pipelineRun := &datamodel.PipelineRun{
			Error:         null.StringFrom(err.Error()),
			CompletedTime: null.TimeFrom(now),
			TotalDuration: null.IntFrom(now.Sub(run.StartedTime).Milliseconds()),
			Status:        datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED),
		}
		repoErr = s.repository.UpdatePipelineRun(ctx, pipelineTriggerID, pipelineRun)
		if repoErr != nil {
			logger.Error("failed to log pipeline run error", zap.Error(err), zap.Error(repoErr))
		}

		return nil, nil, err
	}

	outputs, triggerMetadata, err := s.getOutputsAndMetadata(ctx, pipelineTriggerID, r, returnTraces)
	if err != nil {
		return nil, nil, err
	}

	_, err = s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, worker.UploadOutputsWorkflow, &worker.UploadOutputsWorkflowParam{
		PipelineTriggerID: pipelineTriggerID,
		Outputs:           outputs,
	})
	if err != nil {
		logger.Error(fmt.Sprintf("failed to execute workflow %s", worker.UploadOutputsWorkflow), zap.Error(err))
	}

	return outputs, triggerMetadata, nil
}

func (s *service) triggerPipelineWithStream(
	ctx context.Context,
	ns resource.Namespace,
	r *datamodel.Recipe,
	isAdmin bool,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineData []*pipelinepb.TriggerData,
	pipelineTriggerID string,
	returnTraces bool,
	stream chan<- TriggerResult) error {

	logger, _ := logger.GetZapLogger(ctx)

	memoryKey, err := s.preTriggerPipeline(ctx, isAdmin, ns, r, pipelineTriggerID, pipelineData)
	if err != nil {
		return err
	}
	defer recipe.Purge(ctx, s.redisClient, pipelineTriggerID)

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

	runSource := datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API)
	userAgentValue, ok := runpb.RunSource_value[resource.GetRequestSingleHeader(ctx, constant.HeaderUserAgentKey)]
	if ok {
		runSource = datamodel.RunSource(userAgentValue)
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			BatchSize:        len(pipelineData),
			MemoryStorageKey: memoryKey,
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    pipelineTriggerID,
				PipelineID:           pipelineID,
				PipelineUID:          pipelineUID,
				PipelineReleaseID:    pipelineReleaseID,
				PipelineReleaseUID:   pipelineReleaseUID,
				PipelineRecipe:       r,
				PipelineOwnerType:    ns.NsType,
				PipelineOwnerUID:     ns.NsUID,
				PipelineUserUID:      userUID,
				PipelineRequesterUID: requesterUID,
				PipelineRunSource:    runSource,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
			},
			IsStreaming: true,
			Mode:        mgmtpb.Mode_MODE_SYNC,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return err
	}

	go func() {
		// todo: how about tick every 0.1s? do we need to query 1000 times each sec?
		const interval = 100 * time.Millisecond
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		ctxQ, cancel := context.WithTimeout(context.Background(), time.Second*60)
		defer cancel()

		for { // nolint:gosimple
			select {
			case <-ticker.C:
				queryResult, err := s.temporalClient.QueryWorkflow(ctxQ, we.GetID(), we.GetRunID(), "workflowStatusQuery")
				if err != nil {
					logger.Error("Error querying workflow status: %v", zap.Error(err))
					return
				}

				var status worker.WorkFlowSignal
				if err := queryResult.Get(&status); err != nil {
					logger.Error("Error querying workflow Get status: %v", zap.Error(err))
					return
				}

				const statusCompleted = "completed" // signals that the workflow has completed
				const statusStep = "step"           // signals that a component has completed, but does not specify which
				switch status.Status {
				case statusStep:
					path := status.ID
					data, metadata, err := s.getOutputsAndMetadataStream(ctx, pipelineTriggerID, r, returnTraces, path)
					if err != nil {
						logger.Error("could not get outputs and metadata", zap.Error(err))
						continue
					}

					if len(data) < 1 {
						logger.Error("no data found to send to stream", zap.String("path", path))
						continue
					}

					stream <- TriggerResult{
						Struct:   data,
						Metadata: metadata,
					}
				case statusCompleted:
					close(stream)
					return
				}
			}
		}
	}()

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

		repoErr := s.repository.UpdatePipelineRun(ctx, pipelineTriggerID, &datamodel.PipelineRun{Error: null.StringFrom(err.Error())})
		if repoErr != nil {
			logger.Error("failed to log pipeline run error", zap.Error(repoErr))
		}

		return err
	}

	// todo: should upload outputs here but it seems there is currently some issue with triggerPipelineWithStream.
	//  TriggerPipelineWorkflow will be triggered repeatedly.
	// outputs, _, err := s.getOutputsAndMetadata(ctx, pipelineTriggerID, r, false)
	// if err != nil {
	// 	logger.Error("failed to get stream pipeline run outputs", zap.Error(err))
	// 	return nil
	// }
	//
	// _, err = s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, worker.UploadOutputsWorkflow, &worker.UploadOutputsWorkflowParam{
	// 	PipelineTriggerID: pipelineTriggerID,
	// 	Outputs:           outputs,
	// })
	// if err != nil {
	// 	logger.Error(fmt.Sprintf("failed to execute workflow %s", worker.UploadOutputsWorkflow), zap.Error(err))
	// }

	return nil
}

func (s *service) triggerAsyncPipeline(
	ctx context.Context,
	ns resource.Namespace,
	r *datamodel.Recipe,
	isAdmin bool,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineData []*pipelinepb.TriggerData,
	pipelineTriggerID string,
	returnTraces bool) (*longrunningpb.Operation, error) {

	memoryKey, err := s.preTriggerPipeline(ctx, isAdmin, ns, r, pipelineTriggerID, pipelineData)
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

	runSource := datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API)
	userAgentValue, ok := runpb.RunSource_value[resource.GetRequestSingleHeader(ctx, constant.HeaderUserAgentKey)]
	if ok {
		runSource = datamodel.RunSource(userAgentValue)
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			BatchSize:        len(pipelineData),
			MemoryStorageKey: memoryKey,
			SystemVariables: recipe.SystemVariables{
				PipelineTriggerID:    pipelineTriggerID,
				PipelineID:           pipelineID,
				PipelineUID:          pipelineUID,
				PipelineReleaseID:    pipelineReleaseID,
				PipelineReleaseUID:   pipelineReleaseUID,
				PipelineRecipe:       r,
				PipelineOwnerType:    ns.NsType,
				PipelineOwnerUID:     ns.NsUID,
				PipelineUserUID:      userUID,
				PipelineRequesterUID: requesterUID,
				PipelineRunSource:    runSource,
				HeaderAuthorization:  resource.GetRequestSingleHeader(ctx, "authorization"),
			},
			Mode: mgmtpb.Mode_MODE_ASYNC,
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

			run, repoErr := s.repository.GetPipelineRunByUID(ctx, uuid.FromStringOrNil(pipelineTriggerID))
			if repoErr != nil {
				logger.Error("failed to log pipeline run error", zap.Error(err), zap.Error(repoErr))
				return
			}

			now := time.Now()
			pipelineRun := &datamodel.PipelineRun{
				Error:         null.StringFrom(err.Error()),
				CompletedTime: null.TimeFrom(now),
				TotalDuration: null.IntFrom(now.Sub(run.StartedTime).Milliseconds()),
				Status:        datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED),
			}
			repoErr = s.repository.UpdatePipelineRun(ctx, pipelineTriggerID, pipelineRun)
			if repoErr != nil {
				logger.Error("failed to log pipeline run error", zap.Error(err), zap.Error(repoErr))
			}

			return
		}

		outputs, _, err := s.getOutputsAndMetadata(subCtx, pipelineTriggerID, r, false)
		if err != nil {
			logger.Error("failed to get async pipeline run outputs", zap.Error(err))
			return
		}

		_, err = s.temporalClient.ExecuteWorkflow(subCtx, workflowOptions, worker.UploadOutputsWorkflow, &worker.UploadOutputsWorkflowParam{
			PipelineTriggerID: pipelineTriggerID,
			Outputs:           outputs,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("failed to execute workflow %s", worker.UploadOutputsWorkflow), zap.Error(err))
		}
	})

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", pipelineTriggerID),
		Done: false,
	}, nil

}

func (s *service) getOutputsAndMetadata(ctx context.Context, pipelineTriggerID string, r *datamodel.Recipe, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	memory, err := recipe.LoadMemoryByTriggerID(ctx, s.redisClient, pipelineTriggerID)
	if err != nil {
		return nil, nil, err
	}

	pipelineOutputs := make([]*structpb.Struct, len(memory))

	for idx := range memory {
		pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		for k, v := range r.Output {
			o, err := recipe.RenderInput(v.Value, idx, memory[idx])
			if err != nil {
				return nil, nil, err
			}
			structVal, err := structpb.NewValue(o)
			if err != nil {
				return nil, nil, err
			}
			pipelineOutput.Fields[k] = structVal

		}
		pipelineOutputs[idx] = pipelineOutput
	}

	var metadata *pipelinepb.TriggerMetadata
	if returnTraces {
		traces, err := recipe.GenerateTraces(r.Component, memory)
		if err != nil {
			return nil, nil, err
		}
		metadata = &pipelinepb.TriggerMetadata{
			Traces: traces,
		}
	}
	return pipelineOutputs, metadata, nil
}

func (s *service) getOutputsAndMetadataStream(ctx context.Context, pipelineTriggerID string, r *datamodel.Recipe, returnTraces bool, path string) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {
	memory, err := recipe.LoadMemoryByTriggerID(ctx, s.redisClient, pipelineTriggerID)
	if err != nil {
		return nil, nil, fmt.Errorf("LoadMemoryByTriggerID: %w", err)
	}

	if memory == nil {
		return nil, nil, fmt.Errorf("memory is nil")
	}

	pipelineOutputs := make([]*structpb.Struct, len(memory))

	for idx, mem := range memory {
		pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}

		for k, v := range r.Output {
			input := v.Value[2:]
			input = input[:len(input)-1]
			input = strings.TrimSpace(input)

			var val any
			if strings.Contains(input, path) {
				if input == recipe.SegSecret+"."+constant.GlobalSecretKey {
					val = componentbase.SecretKeyword
				} else {
					val, err = recipe.TraverseBinding(mem, input)
					if err != nil {
						// If the path is not found, we should continue to the next output
						continue
					}
				}

				structVal, err := structpb.NewValue(val)
				if err != nil {
					return nil, nil, err
				}
				pipelineOutput.Fields[k] = structVal
			}
		}
		pipelineOutputs[idx] = pipelineOutput
	}

	var metadata *pipelinepb.TriggerMetadata
	if returnTraces {
		traces, err := recipe.GenerateTraces(r.Component, memory)
		if err != nil {
			return nil, nil, err
		}

		// Find the trace that matches the component
		for compID, trace := range traces {
			if strings.Contains(compID, path) {
				metadata = &pipelinepb.TriggerMetadata{
					Traces: map[string]*pipelinepb.Trace{compID: trace},
				}
				break // We only want the first matching trace
			}
		}
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
		return fmt.Errorf("authenticated user doesn't belong to requester organization: %s", errdomain.ErrUnauthorized)
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

func (s *service) checkTriggerPermission(ctx context.Context, pipeline *datamodel.Pipeline) (isAdmin bool, err error) {
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", pipeline.UID, "reader"); err != nil {
		return false, err
	} else if !granted {
		return false, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", pipeline.UID, "executor"); err != nil {
		return false, err
	} else if !granted {
		return false, errdomain.ErrUnauthorized
	}

	// For now, impersonation is only implemented for pipeline triggers. When
	// this is used in other entrypoints, the requester permission should be
	// checked at a higher level (e.g. handler or middleware).
	if err := s.checkRequesterPermission(ctx, pipeline); err != nil {
		return false, fmt.Errorf("checking requester permission: %w", err)
	}

	if isAdmin, err = s.aclClient.CheckPermission(ctx, "pipeline", pipeline.UID, "admin"); err != nil {
		return false, err
	}

	return isAdmin, nil
}

func (s *service) CheckPipelineEventCode(ctx context.Context, ns resource.Namespace, id string, code string) (bool, error) {
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ns.Permalink(), id, false, true)
	if err != nil {
		return false, errdomain.ErrNotFound
	}

	return dbPipeline.ShareCode == code, nil
}

func (s *service) HandleNamespacePipelineEventByID(ctx context.Context, ns resource.Namespace, id string, eventID string, data *structpb.Struct, pipelineTriggerID string) (*structpb.Struct, error) {

	targetType := ""
	ownerPermalink := ns.Permalink()
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	if e, ok := dbPipeline.Recipe.On.Event[eventID]; ok {

		targetType = e.Type
	} else {
		return nil, fmt.Errorf("eventID not correct")
	}

	md, _ := metadata.FromIncomingContext(ctx)

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
			if eventID == s[1] {
				path := strings.Join(s[3:], ".")
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

	_, err = s.triggerAsyncPipeline(ctx, ns, dbPipeline.Recipe, true, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, triggerData, pipelineTriggerID, false)
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

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline)
	if err != nil {
		return nil, nil, fmt.Errorf("check trigger permission error: %w", err)
	}

	return s.triggerPipeline(ctx, ns, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, data, pipelineTriggerID, returnTraces)
}

func (s *service) TriggerNamespacePipelineByIDWithStream(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool, stream chan<- TriggerResult) error {
	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return errdomain.ErrNotFound
	}

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline)
	if err != nil {
		return err
	}

	if err := s.triggerPipelineWithStream(ctx, ns, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, data, pipelineTriggerID, returnTraces, stream); err != nil {
		return err
	}

	return nil
}

func (s *service) TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline)
	if err != nil {
		return nil, err
	}

	return s.triggerAsyncPipeline(ctx, ns, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, data, pipelineTriggerID, returnTraces)

}

func (s *service) TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, true)
	if err != nil {
		return nil, nil, errdomain.ErrNotFound
	}

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline)
	if err != nil {
		return nil, nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, nil, err
	}

	return s.triggerPipeline(ctx, ns, dbPipelineRelease.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, data, pipelineTriggerID, returnTraces)
}

func (s *service) TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline)
	if err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, err
	}

	return s.triggerAsyncPipeline(ctx, ns, dbPipelineRelease.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, data, pipelineTriggerID, returnTraces)
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
		ownerPermalink := recipe.LoadOwnerPermalink(ctx, s.redisClient, pipelineTriggerID)
		r, err := recipe.LoadRecipe(ctx, s.redisClient, fmt.Sprintf("%s:%s", pipelineTriggerID, recipe.SegRecipe))
		if err != nil {
			return nil, err
		}
		outputs, metadata, err := s.getOutputsAndMetadata(ctx, pipelineTriggerID, r, true)
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(ownerPermalink, "user") {
			pipelineResp := &pipelinepb.TriggerUserPipelineResponse{
				Outputs:  outputs,
				Metadata: metadata,
			}

			resp, err := anypb.New(pipelineResp)
			if err != nil {
				return nil, err
			}
			resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1beta.TriggerUserPipelineResponse"
			operation = longrunningpb.Operation{
				Done: true,
				Result: &longrunningpb.Operation_Response{
					Response: resp,
				},
			}
		} else {
			pipelineResp := &pipelinepb.TriggerOrganizationPipelineResponse{
				Outputs:  outputs,
				Metadata: metadata,
			}

			resp, err := anypb.New(pipelineResp)
			if err != nil {
				return nil, err
			}
			resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1beta.TriggerOrganizationPipelineResponse"
			operation = longrunningpb.Operation{
				Done: true,
				Result: &longrunningpb.Operation_Response{
					Response: resp,
				},
			}
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
				Error: &rpcStatus.Status{
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

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ns.Permalink(), req.GetPipelineId(), true, false)
	if err != nil {
		return nil, err
	}

	ctxUserUID := utils.GetUserUID(ctx)
	page := s.pageInRange(req.GetPage())
	pageSize := s.pageSizeInRange(req.GetPageSize())

	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}

	pipelineRuns, totalCount, err := s.repository.GetPaginatedPipelineRunsWithPermissions(ctx, ctxUserUID, dbPipeline.UID.String(), page, pageSize, filter, orderBy, dbPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get pipeline runs: %w", err)
	}

	var referenceIDs []string
	for _, pipelineRun := range pipelineRuns {
		if pipelineRun.TriggeredBy == ctxUserUID { // only the runner could see their input/output data
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

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		logger.Error("failed to get files from minio", zap.Error(err))
		return nil, err
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
		logger.Info("CheckNamespaceByUIDAdmin finished", zap.String("RequesterUID", requesterID), zap.String("runnerId", runner.Id))
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

		if run.TriggeredBy == ctxUserUID { // only the runner could see their input/output data
			if len(run.Inputs) == 1 {
				data, ok := metadataMap[run.Inputs[0].Name]
				if !ok {
					return nil, fmt.Errorf("failed to load input metadata. pipeline UID: %s input reference ID: %s", run.PipelineUID.String(), run.Inputs[0].Name)
				}
				pbRun.Inputs = make([]*structpb.Struct, 0)
				err = json.Unmarshal(data, &pbRun.Inputs)
				if err != nil {
					return nil, err
				}

			}
			if len(run.Outputs) == 1 {
				data, ok := metadataMap[run.Outputs[0].Name]
				if !ok {
					return nil, fmt.Errorf("failed to load output metadata. pipeline UID: %s output reference ID: %s", run.PipelineUID.String(), run.Outputs[0].Name)
				}
				pbRun.Outputs = make([]*structpb.Struct, 0)
				err = json.Unmarshal(data, &pbRun.Outputs)
				if err != nil {
					return nil, err
				}
			}
			if len(run.RecipeSnapshot) == 1 {
				data, ok := metadataMap[run.RecipeSnapshot[0].Name]
				if !ok {
					return nil, fmt.Errorf("failed to load output metadata. pipeline UID: %s output reference ID: %s", run.PipelineUID.String(), run.Outputs[0].Name)
				}
				r := make(map[string]any)
				err = json.Unmarshal(data, &r)
				if err != nil {
					return nil, fmt.Errorf("failed to load recipe metadata. pipeline UID: %s recipe reference ID: %s", run.PipelineUID.String(), run.RecipeSnapshot[0].Name)
				}
				pbRun.RecipeSnapshot, err = structpb.NewStruct(r)
				if err != nil {
					return nil, err
				}
			}
		}

		pbPipelineRuns[i] = pbRun
	}

	return &pipelinepb.ListPipelineRunsResponse{
		PipelineRuns: pbPipelineRuns,
		TotalSize:    totalCount,
		Page:         int32(page),
		PageSize:     int32(pageSize),
	}, nil
}

func (s *service) ListComponentRuns(ctx context.Context, req *pipelinepb.ListComponentRunsRequest, filter filtering.Filter) (*pipelinepb.ListComponentRunsResponse, error) {
	page := s.pageInRange(req.GetPage())
	pageSize := s.pageSizeInRange(req.GetPageSize())
	ctxUserUID := utils.GetUserUID(ctx)

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

	// if the view requester is the pipeline owner, they could see all component runs. else...
	if dbPipeline.OwnerUID().String() != ctxUserUID {
		// if  the view requester is the pipeline runner, they could view the component runs of the pipeline run.
		// else they could see nothing
		if dbPipelineRun.TriggeredBy != ctxUserUID {
			return nil, fmt.Errorf("requester is not pipeline owner or runner. they are not allowed to view these component runs")
		}
	}

	componentRuns, totalCount, err := s.repository.GetPaginatedComponentRunsByPipelineRunIDWithPermissions(ctx, ctxUserUID, req.GetPipelineRunId(), page, pageSize, filter, orderBy, dbPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get component runs: %w", err)
	}

	var referenceIDs []string
	for _, pipelineRun := range componentRuns {
		if dbPipelineRun.TriggeredBy == ctxUserUID { // only the runner could see their input/output data
			for _, input := range pipelineRun.Inputs {
				referenceIDs = append(referenceIDs, input.Name)
			}
			for _, output := range pipelineRun.Outputs {
				referenceIDs = append(referenceIDs, output.Name)
			}
		}
	}

	logger, _ := logger.GetZapLogger(ctx)
	logger.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		logger.Error("failed to get files from minio", zap.Error(err))
		return nil, err
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

		if dbPipelineRun.TriggeredBy == ctxUserUID { // only the runner could see their input/output data
			if len(run.Inputs) == 1 {
				data, ok := metadataMap[run.Inputs[0].Name]
				if !ok {
					return nil, fmt.Errorf("failed to load input metadata. component UID: %s input reference ID: %s", run.ComponentID, run.Inputs[0].Name)
				}
				pbRun.Inputs = make([]*structpb.Struct, 0)
				err = json.Unmarshal(data, &pbRun.Inputs)
				if err != nil {
					return nil, err
				}

			}
			if len(run.Outputs) == 1 {
				data, ok := metadataMap[run.Outputs[0].Name]
				if !ok {
					return nil, fmt.Errorf("failed to load output metadata. component UID: %s output reference ID: %s", run.ComponentID, run.Outputs[0].Name)
				}
				pbRun.Outputs = make([]*structpb.Struct, 0)
				err = json.Unmarshal(data, &pbRun.Outputs)
				if err != nil {
					return nil, err
				}
			}
		}
		pbComponentRuns[i] = pbRun
	}

	return &pipelinepb.ListComponentRunsResponse{
		ComponentRuns: pbComponentRuns,
		TotalSize:     totalCount,
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

// Helper methods
func (s *service) convertPipelineRunToPB(run datamodel.PipelineRun) (*pipelinepb.PipelineRun, error) {
	result := &pipelinepb.PipelineRun{
		PipelineUid:     run.PipelineUID.String(),
		PipelineRunUid:  run.PipelineTriggerUID.String(),
		PipelineVersion: run.PipelineVersion,
		Status:          runpb.RunStatus(run.Status),
		Source:          runpb.RunSource(run.Source),
		StartTime:       timestamppb.New(run.StartedTime),
		Error:           run.Error.Ptr(),
	}

	if run.TotalDuration.Valid {
		totalDuration := int32(run.TotalDuration.Int64)
		result.TotalDuration = &totalDuration
	}
	if run.CompletedTime.Valid {
		result.CompleteTime = timestamppb.New(run.CompletedTime.Time)
	}

	for _, fileReference := range run.Inputs {
		result.InputsReference = append(result.InputsReference, &pipelinepb.FileReference{
			Name: fileReference.Name,
			Type: fileReference.Type,
			Size: fileReference.Size,
			Url:  fileReference.URL,
		})
	}
	for _, fileReference := range run.Outputs {
		result.OutputsReference = append(result.OutputsReference, &pipelinepb.FileReference{
			Name: fileReference.Name,
			Type: fileReference.Type,
			Size: fileReference.Size,
			Url:  fileReference.URL,
		})
	}

	return result, nil
}

func (s *service) convertComponentRunToPB(run datamodel.ComponentRun) (*pipelinepb.ComponentRun, error) {
	result := &pipelinepb.ComponentRun{
		PipelineRunUid: run.PipelineTriggerUID.String(),
		ComponentId:    run.ComponentID,
		Status:         runpb.RunStatus(run.Status),
		StartTime:      timestamppb.New(run.StartedTime),
		Error:          run.Error.Ptr(),
	}

	if run.TotalDuration.Valid {
		totalDuration := int32(run.TotalDuration.Int64)
		result.TotalDuration = &totalDuration
	}
	if run.CompletedTime.Valid {
		result.CompleteTime = timestamppb.New(run.CompletedTime.Time)
	}

	for _, fileReference := range run.Inputs {
		result.InputsReference = append(result.InputsReference, &pipelinepb.FileReference{
			Name: fileReference.Name,
			Type: fileReference.Type,
			Size: fileReference.Size,
			Url:  fileReference.URL,
		})
	}
	for _, fileReference := range run.Outputs {
		result.OutputsReference = append(result.OutputsReference, &pipelinepb.FileReference{
			Name: fileReference.Name,
			Type: fileReference.Type,
			Size: fileReference.Size,
			Url:  fileReference.URL,
		})
	}
	return result, nil
}
