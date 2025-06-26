package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
	usageclient "github.com/instill-ai/usage-client/client"
	usagereporter "github.com/instill-ai/usage-client/reporter"
)

// Usage interface
type Usage interface {
	RetrieveUsageData() any
	StartReporter(ctx context.Context)
	TriggerSingleReporter(ctx context.Context)
}

type usage struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtpb.MgmtPrivateServiceClient
	redisClient              *redis.Client
	reporter                 usagereporter.Reporter
	serviceVersion           string
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, m mgmtpb.MgmtPrivateServiceClient, rc *redis.Client, usc usagepb.UsageServiceClient, serviceVersion string) Usage {
	logger, _ := logger.GetZapLogger(ctx)

	var defaultOwnerUID string
	if resp, err := m.GetUserAdmin(ctx, &mgmtpb.GetUserAdminRequest{UserId: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
	}

	reporter, err := usageclient.InitReporter(ctx, usc, usagepb.Session_SERVICE_PIPELINE, config.Config.Server.Edition, serviceVersion, defaultOwnerUID)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	return &usage{
		repository:               r,
		mgmtPrivateServiceClient: m,
		redisClient:              rc,
		reporter:                 reporter,
	}
}

func (u *usage) RetrieveUsageData() any {

	ctx := context.Background()
	logger, _ := logger.GetZapLogger(ctx)

	logger.Debug("Retrieve usage data...")

	pbPipelineUsageData := []*usagepb.PipelineUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int32(repository.MaxPageSize)
	for {
		userResp, err := u.mgmtPrivateServiceClient.ListUsersAdmin(ctx, &mgmtpb.ListUsersAdminRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUsersAdmin] %s", err))
			break
		}

		// Roll all pipeline resources on a user
		for _, user := range userResp.GetUsers() {

			triggerDataList := []*usagepb.PipelineUsageData_UserUsageData_PipelineTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", user.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for range make([]struct{}, triggerCount) {
					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", user.GetUid())).Val()

					triggerData := &utils.PipelineUsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagepb.PipelineUsageData_UserUsageData_PipelineTriggerData{
							PipelineId:         triggerData.PipelineID,
							PipelineUid:        triggerData.PipelineUID,
							PipelineReleaseId:  triggerData.PipelineReleaseID,
							PipelineReleaseUid: triggerData.PipelineReleaseUID,
							TriggerUid:         triggerData.PipelineTriggerUID,
							TriggerTime:        timestamppb.New(triggerTime),
							TriggerMode:        triggerData.TriggerMode,
							Status:             triggerData.Status,
							UserUid:            triggerData.UserUID,
							UserType:           triggerData.UserType,
						},
					)
				}
			}

			pbPipelineUsageData = append(pbPipelineUsageData, &usagepb.PipelineUsageData_UserUsageData{
				OwnerUid:            user.GetUid(),
				OwnerType:           mgmtpb.OwnerType_OWNER_TYPE_USER,
				PipelineTriggerData: triggerDataList,
			})

		}

		if userResp.NextPageToken == "" {
			break
		} else {
			userPageToken = userResp.NextPageToken
		}
	}

	// Roll over all orgs and update the metrics with the cached uuid
	orgPageToken := ""
	orgPageSizeMax := int32(repository.MaxPageSize)
	for {
		orgResp, err := u.mgmtPrivateServiceClient.ListOrganizationsAdmin(ctx, &mgmtpb.ListOrganizationsAdminRequest{
			PageSize:  &orgPageSizeMax,
			PageToken: &orgPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListOrganizationsAdmin] %s", err))
			break
		}

		// Roll all pipeline resources on a user
		for _, org := range orgResp.GetOrganizations() {

			triggerDataList := []*usagepb.PipelineUsageData_UserUsageData_PipelineTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", org.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for range make([]struct{}, triggerCount) {

					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", org.GetUid())).Val()

					triggerData := &utils.PipelineUsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagepb.PipelineUsageData_UserUsageData_PipelineTriggerData{
							PipelineId:         triggerData.PipelineID,
							PipelineUid:        triggerData.PipelineUID,
							PipelineReleaseId:  triggerData.PipelineReleaseID,
							PipelineReleaseUid: triggerData.PipelineReleaseUID,
							TriggerUid:         triggerData.PipelineTriggerUID,
							TriggerTime:        timestamppb.New(triggerTime),
							TriggerMode:        triggerData.TriggerMode,
							Status:             triggerData.Status,
							UserUid:            triggerData.UserUID,
							UserType:           triggerData.UserType,
						},
					)
				}
			}

			pbPipelineUsageData = append(pbPipelineUsageData, &usagepb.PipelineUsageData_UserUsageData{
				OwnerUid:            org.GetUid(),
				OwnerType:           mgmtpb.OwnerType_OWNER_TYPE_ORGANIZATION,
				PipelineTriggerData: triggerDataList,
			})

		}

		if orgResp.NextPageToken == "" {
			break
		} else {
			orgPageToken = orgResp.NextPageToken
		}
	}

	logger.Debug("Send retrieved usage data...")

	return &usagepb.SessionReport_PipelineUsageData{
		PipelineUsageData: &usagepb.PipelineUsageData{
			Usages: pbPipelineUsageData,
		},
	}
}

func (u *usage) StartReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger(ctx)

	var defaultOwnerUID string
	if resp, err := u.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtpb.GetUserAdminRequest{UserId: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		err := usageclient.StartReporter(ctx, u.reporter, usagepb.Session_SERVICE_PIPELINE, config.Config.Server.Edition, u.serviceVersion, defaultOwnerUID, u.RetrieveUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func (u *usage) TriggerSingleReporter(ctx context.Context) {
	if u.reporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger(ctx)

	var defaultOwnerUID string
	if resp, err := u.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtpb.GetUserAdminRequest{UserId: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
		return
	}

	err := usageclient.SingleReporter(ctx, u.reporter, usagepb.Session_SERVICE_PIPELINE, config.Config.Server.Edition, u.serviceVersion, defaultOwnerUID, u.RetrieveUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
}
