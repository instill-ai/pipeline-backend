package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"go.einride.tech/aip/filtering"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/x/repo"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
	usageClient "github.com/instill-ai/usage-client/client"
	usageReporter "github.com/instill-ai/usage-client/reporter"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() interface{}
	StartReporter(ctx context.Context)
	TriggerSingleReporter(ctx context.Context)
}

type usage struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	redisClient              *redis.Client
	reporter                 usageReporter.Reporter
	version                  string
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, mu mgmtPB.MgmtPrivateServiceClient, rc *redis.Client, usc usagePB.UsageServiceClient) Usage {
	logger, _ := logger.GetZapLogger()

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	reporter, err := usageClient.InitReporter(ctx, usc, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, version)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	return &usage{
		repository:               r,
		mgmtPrivateServiceClient: mu,
		redisClient:              rc,
		reporter:                 reporter,
		version:                  version,
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
		userResp, err := u.mgmtPrivateServiceClient.ListUsersAdmin(ctx, &mgmtPB.ListUsersAdminRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
			break
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

			triggerNum, err := u.redisClient.Get(ctx, fmt.Sprintf("user:%s:trigger.num", user.GetUid())).Int64()
			if err == redis.Nil {
				triggerNum = 0
			} else if err != nil {
				logger.Error(fmt.Sprintf("%s", err))
			}

			pbPipelineUsageData = append(pbPipelineUsageData, &usagePB.PipelineUsageData_UserUsageData{
				UserUid:                  user.GetUid(),
				PipelineActiveStateNum:   pipeActiveStateNum,
				PipelineInactiveStateNum: pipeInactiveStateNum,
				PipelineSyncModeNum:      pipeSyncModeNum,
				PipelineAsyncModeNum:     pipeAsyncModeNum,
				TriggerNum:               triggerNum,
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

func (u *usage) StartReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger()
	go func() {
		time.Sleep(5 * time.Second)
		err := usageClient.StartReporter(ctx, u.reporter, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, u.version, u.RetrieveUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func (u *usage) TriggerSingleReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}
	logger, _ := logger.GetZapLogger()
	err := usageClient.SingleReporter(ctx, u.reporter, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, u.version, u.RetrieveUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
}
