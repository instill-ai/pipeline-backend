package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

const defaultPipelineReleaseID = "latest"

func (s *service) logPipelineRunStart(ctx context.Context, pipelineTriggerID string, pipelineUID uuid.UUID, pipelineReleaseID string) *datamodel.PipelineRun {
	runSource := datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API)
	userAgentValue, ok := runpb.RunSource_value[resource.GetRequestSingleHeader(ctx, constant.HeaderUserAgentKey)]
	if ok {
		runSource = datamodel.RunSource(userAgentValue)
	}

	requesterUID, userUID := utils.GetRequesterUIDAndUserUID(ctx)

	pipelineRun := &datamodel.PipelineRun{
		PipelineTriggerUID: uuid.FromStringOrNil(pipelineTriggerID),
		PipelineUID:        pipelineUID,
		PipelineVersion:    pipelineReleaseID,
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:             runSource,
		Namespace:          requesterUID,
		TriggeredBy:        userUID,
		StartedTime:        time.Now(),
	}

	if err := s.repository.UpsertPipelineRun(ctx, pipelineRun); err != nil {
		s.log.Error("failed to log pipeline run", zap.String("pipelineTriggerID", pipelineTriggerID), zap.Error(err))
	}
	return pipelineRun
}

func (s *service) logPipelineRunError(ctx context.Context, pipelineTriggerID string, err error, startedTime time.Time) {
	now := time.Now()
	pipelineRunUpdates := &datamodel.PipelineRun{
		Error:         null.StringFrom(err.Error()),
		Status:        datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED),
		TotalDuration: null.IntFrom(now.Sub(startedTime).Milliseconds()),
		CompletedTime: null.TimeFrom(now),
	}

	if err = s.repository.UpdatePipelineRun(ctx, pipelineTriggerID, pipelineRunUpdates); err != nil {
		s.log.Error("failed to log pipeline run error", zap.String("pipelineTriggerID", pipelineTriggerID), zap.Error(err))
	}
}

func (s *service) ListPipelineRuns(ctx context.Context, req *pb.ListPipelineRunsRequest, filter filtering.Filter) (*pb.ListPipelineRunsResponse, error) {
	ns, err := s.GetRscNamespace(ctx, req.GetNamespaceId())
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

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

	s.log.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		s.log.Error("failed to get files from minio", zap.Error(err))
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
		runnerMap[requesterID] = &runner.Id
	}

	// Convert datamodel.PipelineRun to pb.PipelineRun
	pbPipelineRuns := make([]*pb.PipelineRun, len(pipelineRuns))
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
					s.log.Error("Failed to load input metadata", zap.Error(err), zap.String("pipelineUID", run.PipelineUID.String()),
						zap.String("inputReferenceID", key))
				}
			}

			if len(run.Outputs) == 1 {
				key := run.Outputs[0].Name
				pbRun.Outputs, err = parseMetadataToStructArray(metadataMap, key)
				if err != nil {
					s.log.Error("Failed to load output metadata", zap.Error(err), zap.String("pipelineUID", run.PipelineUID.String()),
						zap.String("outputReferenceID", key))
				}
			}

			if len(run.RecipeSnapshot) == 1 {
				key := run.RecipeSnapshot[0].Name
				pbRun.RecipeSnapshot, pbRun.DataSpecification, err = parseRecipeMetadata(ctx, metadataMap, s.converter, key)
				if err != nil {
					s.log.Error("Failed to load recipe snapshot", zap.Error(err), zap.String("pipelineUID", run.PipelineUID.String()),
						zap.String("recipeReferenceID", key))
				}
			}
		}

		pbPipelineRuns[i] = pbRun
	}

	return &pb.ListPipelineRunsResponse{
		PipelineRuns: pbPipelineRuns,
		TotalSize:    int32(totalCount),
		Page:         int32(page),
		PageSize:     int32(pageSize),
	}, nil
}

func (s *service) ListComponentRuns(ctx context.Context, req *pb.ListComponentRunsRequest, filter filtering.Filter) (*pb.ListComponentRunsResponse, error) {
	page := s.pageInRange(req.GetPage())
	pageSize := s.pageSizeInRange(req.GetPageSize())
	requesterUID, _ := utils.GetRequesterUIDAndUserUID(ctx)

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

	s.log.Info("start to get files from minio", zap.String("referenceIDs", strings.Join(referenceIDs, ",")))
	fileContents, err := s.minioClient.GetFilesByPaths(ctx, referenceIDs)
	if err != nil {
		s.log.Error("failed to get files from minio", zap.Error(err))
	}

	metadataMap := make(map[string][]byte)
	for _, content := range fileContents {
		metadataMap[content.Name] = content.Content
	}

	// Convert datamodel.ComponentRun to pb.ComponentRun
	pbComponentRuns := make([]*pb.ComponentRun, len(componentRuns))
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
					s.log.Error("Failed to load input metadata", zap.Error(err), zap.String("ComponentID", run.ComponentID),
						zap.String("inputReferenceID", key))
				}
			}
			if len(run.Outputs) == 1 {
				key := run.Outputs[0].Name
				pbRun.Outputs, err = parseMetadataToStructArray(metadataMap, key)
				if err != nil {
					s.log.Error("Failed to load output metadata", zap.Error(err), zap.String("ComponentID", run.ComponentID),
						zap.String("outputReferenceID", key))
				}

			}
		}
		pbComponentRuns[i] = pbRun
	}

	return &pb.ListComponentRunsResponse{
		ComponentRuns: pbComponentRuns,
		TotalSize:     int32(totalCount),
		Page:          int32(page),
		PageSize:      int32(pageSize),
	}, nil
}

func (s *service) ListPipelineRunsByRequester(ctx context.Context, req *pb.ListPipelineRunsByCreditOwnerRequest) (*pb.ListPipelineRunsByCreditOwnerResponse, error) {
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

	pbPipelineRuns := make([]*pb.PipelineRun, len(pipelineRuns))

	var pbRun *pb.PipelineRun
	for i, run := range pipelineRuns {
		pbRun, err = s.convertPipelineRunToPB(run)
		if err != nil {
			return nil, fmt.Errorf("converting pipeline run: %w", err)
		}
		pbRun.RunnerId = runnerMap[run.TriggeredBy]
		pbPipelineRuns[i] = pbRun
	}

	return &pb.ListPipelineRunsByCreditOwnerResponse{
		PipelineRuns: pbPipelineRuns,
		TotalSize:    int32(totalCount),
		Page:         int32(page),
		PageSize:     int32(pageSize),
	}, nil
}
