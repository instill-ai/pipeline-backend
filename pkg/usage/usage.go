package usage

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v9"
	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() interface{}
}

type usage struct {
	repository        repository.Repository
	userServiceClient mgmtPB.UserServiceClient
	redisClient       *redis.Client
}

// NewUsage initiates a usage instance
func NewUsage(r repository.Repository, mu mgmtPB.UserServiceClient, rc *redis.Client) Usage {
	return &usage{
		repository:        r,
		userServiceClient: mu,
		redisClient:       rc,
	}
}

func (u *usage) RetrieveUsageData() interface{} {

	logger, _ := logger.GetZapLogger()
	ctx := context.Background()

	logger.Debug("Retrieve usage data...")

	pbPipelineUsageData := []*usagePB.PipelineUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int64(repository.MaxPageSize)
	for {
		userResp, err := u.userServiceClient.ListUser(ctx, &mgmtPB.ListUserRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
		}

		// Roll all pipeline resources on a user
		for _, user := range userResp.Users {
			pipePageToken := ""
			pipeActiveStateNum := int64(0)
			pipeInactiveStateNum := int64(0)
			pipeSyncModeNum := int64(0)
			pipeAsyncModeNum := int64(0)
			for {
				dbPipelines, _, pipeNextPageToken, err := u.repository.ListPipeline(fmt.Sprintf("users/%s", user.GetUid()), int64(repository.MaxPageSize), pipePageToken, true, filtering.Filter{})
				if err != nil {
					logger.Error(fmt.Sprintf("%s", err))
				}

				for _, pipeline := range dbPipelines {
					if pipeline.State == datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE) {
						pipeActiveStateNum++
					}
					if pipeline.State == datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE) {
						pipeInactiveStateNum++
					}
					if pipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC) {
						pipeSyncModeNum++
					}
					if pipeline.Mode == datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC) {
						pipeAsyncModeNum++
					}
				}

				if pipeNextPageToken == "" {
					break
				} else {
					pipePageToken = pipeNextPageToken
				}
			}

			triggerImageNum, err := u.redisClient.Get(ctx, fmt.Sprintf("user:%s:trigger.image.num", user.GetUid())).Int64()
			if err == redis.Nil {
				triggerImageNum = 0
			} else if err != nil {
				logger.Error(fmt.Sprintf("%s", err))
			}

			pbPipelineUsageData = append(pbPipelineUsageData, &usagePB.PipelineUsageData_UserUsageData{
				UserUid:                  user.GetUid(),
				PipelineActiveStateNum:   pipeActiveStateNum,
				PipelineInactiveStateNum: pipeInactiveStateNum,
				PipelineSyncModeNum:      pipeSyncModeNum,
				PipelineAsyncModeNum:     pipeAsyncModeNum,
				TriggerImageNum:          triggerImageNum,
			})

		}

		if userResp.NextPageToken == "" {
			break
		} else {
			userPageToken = userResp.NextPageToken
		}
	}

	logger.Debug("Send retrieved usage data...")

	return &usagePB.SessionReport_PipelineUsageData{
		PipelineUsageData: &usagePB.PipelineUsageData{
			Usages: pbPipelineUsageData,
		},
	}
}
