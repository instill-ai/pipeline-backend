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
	"github.com/instill-ai/x/repo"

	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagePB "github.com/instill-ai/protogen-go/core/usage/v1beta"
	usageClient "github.com/instill-ai/usage-client/client"
	usageReporter "github.com/instill-ai/usage-client/reporter"
)

// Usage interface
type Usage interface {
	RetrievePipelineUsageData() interface{}
	RetrieveConnectorUsageData() interface{}
	StartReporter(ctx context.Context)
	TriggerSingleReporter(ctx context.Context)
}

type usage struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	redisClient              *redis.Client
	pipelineReporter         usageReporter.Reporter
	connectorReporter        usageReporter.Reporter
	version                  string
}

// NewUsage initiates a usage instance
func NewUsage(ctx context.Context, r repository.Repository, mu mgmtPB.MgmtPrivateServiceClient, rc *redis.Client, usc usagePB.UsageServiceClient) Usage {
	logger, _ := logger.GetZapLogger(ctx)

	version, err := repo.ReadReleaseManifest("release-please/manifest.json")
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	var defaultOwnerUID string
	if resp, err := mu.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{UserId: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
	}

	pipelineReporter, err := usageClient.InitReporter(ctx, usc, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, version, defaultOwnerUID)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}
	connectorReporter, err := usageClient.InitReporter(ctx, usc, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, version, defaultOwnerUID)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	return &usage{
		repository:               r,
		mgmtPrivateServiceClient: mu,
		redisClient:              rc,
		pipelineReporter:         pipelineReporter,
		connectorReporter:        connectorReporter,
		version:                  version,
	}
}

func (u *usage) RetrievePipelineUsageData() interface{} {

	ctx := context.Background()
	logger, _ := logger.GetZapLogger(ctx)

	logger.Debug("Retrieve usage data...")

	pbPipelineUsageData := []*usagePB.PipelineUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int32(repository.MaxPageSize)
	for {
		userResp, err := u.mgmtPrivateServiceClient.ListUsersAdmin(ctx, &mgmtPB.ListUsersAdminRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUsersAdmin] %s", err))
			break
		}

		// Roll all pipeline resources on a user
		for _, user := range userResp.GetUsers() {

			triggerDataList := []*usagePB.PipelineUsageData_UserUsageData_PipelineTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", user.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for i := int64(0); i < triggerCount; i++ {

					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", user.GetUid())).Val()

					triggerData := &utils.PipelineUsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagePB.PipelineUsageData_UserUsageData_PipelineTriggerData{
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

			pbPipelineUsageData = append(pbPipelineUsageData, &usagePB.PipelineUsageData_UserUsageData{
				OwnerUid:            user.GetUid(),
				OwnerType:           mgmtPB.OwnerType_OWNER_TYPE_USER,
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
		orgResp, err := u.mgmtPrivateServiceClient.ListOrganizationsAdmin(ctx, &mgmtPB.ListOrganizationsAdminRequest{
			PageSize:  &orgPageSizeMax,
			PageToken: &orgPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListOrganizationsAdmin] %s", err))
			break
		}

		// Roll all pipeline resources on a user
		for _, org := range orgResp.GetOrganizations() {

			triggerDataList := []*usagePB.PipelineUsageData_UserUsageData_PipelineTriggerData{}

			triggerCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", org.GetUid())).Val() // O(1)

			if triggerCount != 0 {
				for i := int64(0); i < triggerCount; i++ {

					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:pipeline.trigger_data", org.GetUid())).Val()

					triggerData := &utils.PipelineUsageMetricData{}
					if err := json.Unmarshal([]byte(strData), triggerData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					triggerTime, _ := time.Parse(time.RFC3339Nano, triggerData.TriggerTime)

					triggerDataList = append(
						triggerDataList,
						&usagePB.PipelineUsageData_UserUsageData_PipelineTriggerData{
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

			pbPipelineUsageData = append(pbPipelineUsageData, &usagePB.PipelineUsageData_UserUsageData{
				OwnerUid:            org.GetUid(),
				OwnerType:           mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION,
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

	return &usagePB.SessionReport_PipelineUsageData{
		PipelineUsageData: &usagePB.PipelineUsageData{
			Usages: pbPipelineUsageData,
		},
	}
}

func (u *usage) RetrieveConnectorUsageData() interface{} {

	ctx := context.Background()
	logger, _ := logger.GetZapLogger(ctx)

	logger.Debug("Retrieve usage data...")

	pbConnectorUsageData := []*usagePB.ConnectorUsageData_UserUsageData{}

	// Roll over all users and update the metrics with the cached uuid
	userPageToken := ""
	userPageSizeMax := int32(repository.MaxPageSize)

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

			executeDataList := []*usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{}

			executeCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:connector.execute_data", user.GetUid())).Val() // O(1)

			if executeCount != 0 {
				for i := int64(0); i < executeCount; i++ {
					// LPop O(1)
					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:connector.execute_data", user.GetUid())).Val()

					executeData := &utils.ConnectorUsageMetricData{}
					if err := json.Unmarshal([]byte(strData), executeData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					executeTime, _ := time.Parse(time.RFC3339Nano, executeData.ExecuteTime)

					executeDataList = append(
						executeDataList,
						&usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{
							ExecuteUid:             executeData.ConnectorExecuteUID,
							ExecuteTime:            timestamppb.New(executeTime),
							ConnectorUid:           executeData.ConnectorUID,
							ConnectorDefinitionUid: executeData.ConnectorDefinitionUID,
							Status:                 executeData.Status,
							UserUid:                executeData.UserUID,
							UserType:               executeData.UserType,
						},
					)
				}
			}

			pbConnectorUsageData = append(pbConnectorUsageData, &usagePB.ConnectorUsageData_UserUsageData{
				OwnerUid:             user.GetUid(),
				OwnerType:            mgmtPB.OwnerType_OWNER_TYPE_USER,
				ConnectorExecuteData: executeDataList,
			})

		}

		if userResp.NextPageToken == "" {
			break
		} else {
			userPageToken = userResp.NextPageToken
		}
	}

	// orgs trigger usage data
	orgPageToken := ""
	orgPageSizeMax := int32(repository.MaxPageSize)

	for {
		orgResp, err := u.mgmtPrivateServiceClient.ListOrganizationsAdmin(ctx, &mgmtPB.ListOrganizationsAdminRequest{
			PageSize:  &orgPageSizeMax,
			PageToken: &orgPageToken,
		})
		if err != nil {
			logger.Error(fmt.Sprintf("[mgmt-backend: ListUser] %s", err))
			break
		}

		// Roll all pipeline resources on a user
		for _, org := range orgResp.GetOrganizations() {

			executeDataList := []*usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{}

			executeCount := u.redisClient.LLen(ctx, fmt.Sprintf("user:%s:connector.execute_data", org.GetUid())).Val() // O(1)

			if executeCount != 0 {
				for i := int64(0); i < executeCount; i++ {
					// LPop O(1)
					strData := u.redisClient.LPop(ctx, fmt.Sprintf("user:%s:connector.execute_data", org.GetUid())).Val()

					executeData := &utils.ConnectorUsageMetricData{}
					if err := json.Unmarshal([]byte(strData), executeData); err != nil {
						logger.Warn("Usage data might be corrupted")
					}

					executeTime, _ := time.Parse(time.RFC3339Nano, executeData.ExecuteTime)

					executeDataList = append(
						executeDataList,
						&usagePB.ConnectorUsageData_UserUsageData_ConnectorExecuteData{
							ExecuteUid:             executeData.ConnectorExecuteUID,
							ExecuteTime:            timestamppb.New(executeTime),
							ConnectorUid:           executeData.ConnectorUID,
							ConnectorDefinitionUid: executeData.ConnectorDefinitionUID,
							Status:                 executeData.Status,
							UserUid:                executeData.UserUID,
							UserType:               executeData.UserType,
						},
					)
				}
			}

			pbConnectorUsageData = append(pbConnectorUsageData, &usagePB.ConnectorUsageData_UserUsageData{
				OwnerUid:             org.GetUid(),
				OwnerType:            mgmtPB.OwnerType_OWNER_TYPE_ORGANIZATION,
				ConnectorExecuteData: executeDataList,
			})

		}

		if orgResp.NextPageToken == "" {
			break
		} else {
			orgPageToken = orgResp.NextPageToken
		}
	}

	logger.Debug("Send retrieved usage data...")

	return &usagePB.SessionReport_ConnectorUsageData{
		ConnectorUsageData: &usagePB.ConnectorUsageData{
			Usages: pbConnectorUsageData,
		},
	}
}

func (u *usage) StartReporter(ctx context.Context) {
	if u.pipelineReporter == nil || u.connectorReporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger(ctx)

	var defaultOwnerUID string
	if resp, err := u.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{UserId: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		err := usageClient.StartReporter(ctx, u.pipelineReporter, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, u.version, defaultOwnerUID, u.RetrievePipelineUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
		err = usageClient.StartReporter(ctx, u.connectorReporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, defaultOwnerUID, u.RetrieveConnectorUsageData)
		if err != nil {
			logger.Error(fmt.Sprintf("unable to start reporter: %v\n", err))
		}
	}()
}

func (u *usage) TriggerSingleReporter(ctx context.Context) {
	if u.pipelineReporter == nil || u.connectorReporter == nil {
		return
	}

	logger, _ := logger.GetZapLogger(ctx)

	var defaultOwnerUID string
	if resp, err := u.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{UserId: constant.DefaultUserID}); err == nil {
		defaultOwnerUID = resp.GetUser().GetUid()
	} else {
		logger.Error(err.Error())
		return
	}

	err := usageClient.SingleReporter(ctx, u.pipelineReporter, usagePB.Session_SERVICE_PIPELINE, config.Config.Server.Edition, u.version, defaultOwnerUID, u.RetrievePipelineUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
	err = usageClient.SingleReporter(ctx, u.connectorReporter, usagePB.Session_SERVICE_CONNECTOR, config.Config.Server.Edition, u.version, defaultOwnerUID, u.RetrieveConnectorUsageData())
	if err != nil {
		logger.Error(fmt.Sprintf("unable to trigger single reporter: %v\n", err))
	}
}
