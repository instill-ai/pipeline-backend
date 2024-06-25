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
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/worker"
	"github.com/instill-ai/x/errmsg"

	componentbase "github.com/instill-ai/component/base"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

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
			return nil, ErrNoPermission
		}
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			return nil, ErrNoPermission
		}
	}

	dbPipeline, err := s.converter.ConvertPipelineToDB(ctx, ns, pbPipeline)
	if err != nil {
		return nil, err
	}
	if err := s.checkSecret(ctx, dbPipeline.Recipe.Component); err != nil {
		return nil, err
	}

	if dbPipeline.ShareCode == "" {
		dbPipeline.ShareCode = generateShareCode()
	}

	if err := s.repository.CreateNamespacePipeline(ctx, ownerPermalink, dbPipeline); err != nil {
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
		return nil, ErrNoPermission
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, false); existingPipeline == nil {
		return nil, err
	}

	if existingPipeline.ShareCode == "" {
		dbPipeline.ShareCode = generateShareCode()
	}

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, dbPipeline.UID, dbPipeline); err != nil {
		return nil, err
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
	pipeline, err := s.converter.ConvertPipelineToPB(ctx, dbPipelineUpdated, pipelinepb.Pipeline_VIEW_RECIPE, false, true)
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
		return ErrNoPermission
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

func (s *service) CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, id string, targetNS resource.Namespace, targetID string) (*pipelinepb.Pipeline, error) {
	sourcePipeline, err := s.GetNamespacePipelineByID(ctx, ns, id, pipelinepb.Pipeline_VIEW_RECIPE)
	if err != nil {
		return nil, err
	}
	sourcePipeline.Id = targetID
	targetPipeline, err := s.CreateNamespacePipeline(ctx, targetNS, sourcePipeline)
	if err != nil {
		return nil, err
	}
	return targetPipeline, nil
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
		return nil, ErrNoPermission
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
		return nil, ErrNoPermission
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
		if memory[idx].Variable == nil {
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
		return nil, ErrNoPermission
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
		return nil, ErrNoPermission
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
		return nil, ErrNoPermission
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
		return ErrNoPermission
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
		return ErrNoPermission
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

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			BatchSize:        len(pipelineData),
			MemoryStorageKey: memoryKey,
			SystemVariables: recipe.SystemVariables{
				PipelineID:          pipelineID,
				PipelineUID:         pipelineUID,
				PipelineReleaseID:   pipelineReleaseID,
				PipelineReleaseUID:  pipelineReleaseUID,
				PipelineRecipe:      r,
				PipelineOwnerType:   ns.NsType,
				PipelineOwnerUID:    ns.NsUID,
				PipelineUserUID:     userUID,
				HeaderAuthorization: resource.GetRequestSingleHeader(ctx, "authorization"),
			},
			Mode: mgmtpb.Mode_MODE_SYNC,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, nil, err
	}

	err = we.Get(ctx, nil)
	if err != nil {
		var applicationErr *temporal.ApplicationError
		if errors.As(err, &applicationErr) {
			var details worker.EndUserErrorDetails
			if dErr := applicationErr.Details(&details); dErr == nil && details.Message != "" {
				// Note: We categorize all pipeline trigger errors as ErrTriggerFail and mark the code as 400 InvalidArgument for now.
				// We should further categorize them into InvalidArgument or PreconditionFailed or InternalError in the future.
				err = errmsg.AddMessage(fmt.Errorf("%w %s", ErrTriggerFail, err), details.Message)
			}
		}

		return nil, nil, err
	}

	return s.getOutputsAndMetadata(ctx, pipelineTriggerID, r, returnTraces)
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
	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowParam{
			BatchSize:        len(pipelineData),
			MemoryStorageKey: memoryKey,
			SystemVariables: recipe.SystemVariables{
				PipelineID:          pipelineID,
				PipelineUID:         pipelineUID,
				PipelineReleaseID:   pipelineReleaseID,
				PipelineReleaseUID:  pipelineReleaseUID,
				PipelineRecipe:      r,
				PipelineOwnerType:   ns.NsType,
				PipelineOwnerUID:    ns.NsUID,
				PipelineUserUID:     userUID,
				HeaderAuthorization: resource.GetRequestSingleHeader(ctx, "authorization"),
			},
			Mode: mgmtpb.Mode_MODE_ASYNC,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with workflowID %s and RunID %s", we.GetID(), we.GetRunID()))

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

func (s *service) checkTriggerPermission(ctx context.Context, pipelineUID uuid.UUID) (isAdmin bool, err error) {

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", pipelineUID, "reader"); err != nil {
		return false, err
	} else if !granted {
		return false, errdomain.ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", pipelineUID, "executor"); err != nil {
		return false, err
	} else if !granted {
		return false, ErrNoPermission
	}

	if isAdmin, err = s.aclClient.CheckPermission(ctx, "pipeline", pipelineUID, "admin"); err != nil {
		return false, err
	}
	return isAdmin, nil
}

func (s *service) TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, nil, errdomain.ErrNotFound
	}

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline.UID)
	if err != nil {
		return nil, nil, err
	}

	return s.triggerPipeline(ctx, ns, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, data, pipelineTriggerID, returnTraces)
}

func (s *service) TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, errdomain.ErrNotFound
	}
	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline.UID)
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

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline.UID)
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

	isAdmin, err := s.checkTriggerPermission(ctx, dbPipeline.UID)
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
