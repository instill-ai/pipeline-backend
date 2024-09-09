package service

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v4"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
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
