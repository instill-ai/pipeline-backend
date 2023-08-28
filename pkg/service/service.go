package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/operator"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the service, we'd better use uid as our function params

// Service interface
type Service interface {
	ListPipelines(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.Pipeline, int64, string, error)
	GetPipelineByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*pipelinePB.Pipeline, error)
	CreateUserPipeline(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	ListUserPipelines(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.Pipeline, int64, string, error)
	GetUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, isBasicView bool) (*pipelinePB.Pipeline, error)
	UpdateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, updatedPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	UpdateUserPipelineIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Pipeline, error)
	DeleteUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error
	ValidateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (*pipelinePB.Pipeline, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.Pipeline, int64, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*pipelinePB.Pipeline, error)

	CreateUserPipelineRelease(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	ListUserPipelineReleases(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.PipelineRelease, int64, string, error)
	GetUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, isBasicView bool) (*pipelinePB.PipelineRelease, error)
	GetUserPipelineReleaseByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*pipelinePB.PipelineRelease, error)
	UpdateUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, updatedPipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	DeleteUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	RestoreUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	SetDefaultUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	UpdateUserPipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error)

	ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.PipelineRelease, int64, string, error)

	// Controller APIs
	GetResourceState(uid uuid.UUID) (*pipelinePB.State, error)
	UpdateResourceState(uid uuid.UUID, state pipelinePB.State, progress *int32) error
	DeleteResourceState(uid uuid.UUID) error

	// Influx API

	TriggerUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error)

	TriggerUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error)

	WriteNewDataPoint(ctx context.Context, data utils.UsageMetricData) error

	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	GetRscNamespaceAndNameIDAndReleaseID(path string) (resource.Namespace, string, string, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)

	PBToDBPipeline(ctx context.Context, userUid uuid.UUID, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, userUid uuid.UUID, dbPipeline *datamodel.Pipeline, isBasicView bool) (*pipelinePB.Pipeline, error)
	DBToPBPipelines(ctx context.Context, userUid uuid.UUID, dbPipeline []*datamodel.Pipeline, isBasicView bool) ([]*pipelinePB.Pipeline, error)
	DBToPBPipelineAdmin(ctx context.Context, dbPipeline *datamodel.Pipeline, isBasicView bool) (*pipelinePB.Pipeline, error)
	DBToPBPipelinesAdmin(ctx context.Context, dbPipeline []*datamodel.Pipeline, isBasicView bool) ([]*pipelinePB.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, userUid uuid.UUID, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, userUid uuid.UUID, dbPipelineRelease *datamodel.PipelineRelease, isBasicView bool) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, userUid uuid.UUID, dbPipelineRelease []*datamodel.PipelineRelease, isBasicView bool) ([]*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleaseAdmin(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, isBasicView bool) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleasesAdmin(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, isBasicView bool) ([]*pipelinePB.PipelineRelease, error)

	GetUser(ctx context.Context) (string, uuid.UUID, error)
}

type service struct {
	repository                    repository.Repository
	mgmtPrivateServiceClient      mgmtPB.MgmtPrivateServiceClient
	connectorPublicServiceClient  connectorPB.ConnectorPublicServiceClient
	connectorPrivateServiceClient connectorPB.ConnectorPrivateServiceClient
	controllerClient              controllerPB.ControllerPrivateServiceClient
	redisClient                   *redis.Client
	temporalClient                client.Client
	influxDBWriteClient           api.WriteAPI
	operator                      operator.Operator
	defaultUserUid                uuid.UUID
}

// NewService initiates a service instance
func NewService(r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	c connectorPB.ConnectorPublicServiceClient,
	cPrivate connectorPB.ConnectorPrivateServiceClient,
	ct controllerPB.ControllerPrivateServiceClient,
	rc *redis.Client,
	t client.Client,
	i api.WriteAPI,
	defaultUserUid uuid.UUID,
) Service {
	return &service{
		repository:                    r,
		mgmtPrivateServiceClient:      u,
		connectorPublicServiceClient:  c,
		connectorPrivateServiceClient: cPrivate,
		controllerClient:              ct,
		redisClient:                   rc,
		temporalClient:                t,
		influxDBWriteClient:           i,
		operator:                      operator.InitOperator(),
		defaultUserUid:                defaultUserUid,
	}
}

// GetUserPermalink returns the api user
func (s *service) GetUser(ctx context.Context) (string, uuid.UUID, error) {
	// Verify if "jwt-sub" is in the header
	headerUserUId := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if headerUserUId != "" {
		_, err := uuid.FromString(headerUserUId)
		if err != nil {
			return "", uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: "users/" + headerUserUId})
		if err != nil {
			return "", uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
		}

		return resp.User.Id, uuid.FromStringOrNil(headerUserUId), nil
	}

	return constant.DefaultUserID, s.defaultUserUid, nil
}

func (s *service) ConvertOwnerPermalinkToName(permalink string) (string, error) {
	userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
	if err != nil {
		return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
	}
	return fmt.Sprintf("users/%s", userResp.User.Id), nil
}
func (s *service) ConvertOwnerNameToPermalink(name string) (string, error) {
	userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtPB.GetUserAdminRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("ConvertOwnerNameToPermalink error")
	}
	return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
}

func (s *service) GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error) {

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", splits[0], splits[1]))

	if err != nil {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, splits[3], nil
}

func (s *service) GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink((fmt.Sprintf("%s/%s", splits[0], splits[1])))
	if err != nil {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

func (s *service) GetRscNamespaceAndNameIDAndReleaseID(path string) (resource.Namespace, string, string, error) {
	ns, pipelineId, err := s.GetRscNamespaceAndNameID(path)
	if err != nil {
		return ns, pipelineId, "", err
	}
	splits := strings.Split(path, "/")

	if len(splits) < 6 {
		return ns, pipelineId, "", fmt.Errorf("path error")
	}
	return ns, pipelineId, splits[5], err
}

func (s *service) ListPipelines(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.Pipeline, int64, string, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, userPermalink, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelines, err := s.DBToPBPipelines(ctx, userUid, dbPipelines, isBasicView)
	return pbPipelines, totalSize, nextPageToken, err

}

func (s *service) GetPipelineByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*pipelinePB.Pipeline, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, userPermalink, uid, isBasicView)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, userUid, dbPipeline, isBasicView)
}

func (s *service) CreateUserPipeline(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pbPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.PBToDBPipeline(ctx, userUid, pbPipeline)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := s.repository.CreateUserPipeline(ctx, ownerPermalink, userPermalink, dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, dbPipeline.ID, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, userUid, dbCreatedPipeline, false)
}

func (s *service) ListUserPipelines(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.Pipeline, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelines, ps, pt, err := s.repository.ListUserPipelines(ctx, ownerPermalink, userPermalink, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, userUid, dbPipelines, isBasicView)
	return pbPipelines, ps, pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.Pipeline, int64, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelinesAdmin(ctx, dbPipelines, isBasicView)
	return pbPipelines, ps, pt, err

}

func (s *service) GetUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, isBasicView bool) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, isBasicView)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, userUid, dbPipeline, isBasicView)
}

func (s *service) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*pipelinePB.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, uid, isBasicView)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineAdmin(ctx, dbPipeline, isBasicView)

}

func (s *service) UpdateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, toUpdPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineToCreate, err := s.PBToDBPipeline(ctx, userUid, toUpdPipeline)
	if err != nil {
		return nil, err
	}

	// Validation: Pipeline existence
	if existingPipeline, _ := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	if err := s.repository.UpdateUserPipelineByID(ctx, ownerPermalink, userPermalink, id, dbPipelineToCreate); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, toUpdPipeline.Id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, userUid, dbPipeline, false)
}

func (s *service) DeleteUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return err
	}

	if err := s.DeleteResourceState(dbPipeline.UID); err != nil {
		return err
	}

	return s.repository.DeleteUserPipelineByID(ctx, ownerPermalink, userPermalink, id)
}

func (s *service) ValidateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	// user desires to be active or inactive, state stay the same
	// but update etcd storage with checkState
	err = s.checkState(dbPipeline.Recipe)
	if err != nil {
		return nil, err
	}

	recipeErr := s.checkRecipe(ownerPermalink, dbPipeline.Recipe)

	if recipeErr != nil {
		return nil, recipeErr
	}

	dbPipeline, err = s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, userUid, dbPipeline, false)

}

func (s *service) UpdateUserPipelineIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	// Validation: Pipeline existence
	existingPipeline, _ := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, true)
	if existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	if err := s.repository.UpdateUserPipelineIDByID(ctx, ownerPermalink, userPermalink, id, newID); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, newID, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, userUid, dbPipeline, false)
}

func (s *service) preTriggerPipeline(recipe *datamodel.Recipe, pipelineInputs []*structpb.Struct) error {

	typeMap := map[string]string{}
	for _, comp := range recipe.Components {
		if comp.DefinitionName == "operator-definitions/start-operator" {
			for key, value := range comp.Configuration.Fields["body"].GetStructValue().Fields {
				typeMap[key] = value.GetStructValue().Fields["type"].GetStringValue()
			}
		}
	}
	for idx := range pipelineInputs {
		for key, val := range pipelineInputs[idx].Fields {
			switch typeMap[key] {
			case "integer":
				v, err := strconv.ParseInt(val.GetStringValue(), 10, 64)
				if err != nil {
					return err
				}
				pipelineInputs[idx].Fields[key] = structpb.NewNumberValue(float64(v))
			case "number":
				v, err := strconv.ParseFloat(val.GetStringValue(), 64)
				if err != nil {
					return err
				}
				pipelineInputs[idx].Fields[key] = structpb.NewNumberValue(v)
			case "boolean":
				v, err := strconv.ParseBool(val.GetStringValue())
				if err != nil {
					return err
				}
				pipelineInputs[idx].Fields[key] = structpb.NewBoolValue(v)
			case "text", "image", "audio", "video":
			case "integer_array", "number_array", "boolean_array", "text_array", "image_array", "audio_array", "video_array":
				if val.GetListValue() == nil {
					return fmt.Errorf("%s should be a array", key)
				}

				switch typeMap[key] {
				case "integer_array":
					vals := []interface{}{}
					for _, val := range val.GetListValue().AsSlice() {
						n, err := strconv.ParseInt(val.(string), 10, 64)
						if err != nil {
							return err
						}
						vals = append(vals, n)
					}
					structVal, err := structpb.NewList(vals)
					if err != nil {
						return err
					}
					pipelineInputs[idx].Fields[key] = structpb.NewListValue(structVal)

				case "number_array":
					vals := []interface{}{}
					for _, val := range val.GetListValue().AsSlice() {
						n, err := strconv.ParseFloat(val.(string), 64)
						if err != nil {
							return err
						}
						vals = append(vals, n)
					}
					structVal, err := structpb.NewList(vals)
					if err != nil {
						return err
					}
					pipelineInputs[idx].Fields[key] = structpb.NewListValue(structVal)
				case "boolean_array":
					vals := []interface{}{}
					for _, val := range val.GetListValue().AsSlice() {
						n, err := strconv.ParseBool(val.(string))
						if err != nil {
							return err
						}
						vals = append(vals, n)
					}
					structVal, err := structpb.NewList(vals)
					if err != nil {
						return err
					}
					pipelineInputs[idx].Fields[key] = structpb.NewListValue(structVal)

				}
			}

		}
	}

	return nil
}

func (s *service) GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowId, "")

	if err != nil {
		return nil, err
	}
	return s.getOperationFromWorkflowInfo(workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) getOperationFromWorkflowInfo(workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:

		pipelineResp := &pipelinePB.TriggerUserPipelineResponse{}

		blobRedisKey := fmt.Sprintf("async_pipeline_response:%s", workflowExecutionInfo.Execution.WorkflowId)
		blob, err := s.redisClient.Get(context.Background(), blobRedisKey).Bytes()
		if err != nil {
			return nil, err
		}

		err = protojson.Unmarshal(blob, pipelineResp)
		if err != nil {
			return nil, err
		}

		resp, err := anypb.New(pipelineResp)
		if err != nil {
			return nil, err
		}
		resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1alpha.TriggerPipelineResponse"
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

func (s *service) CreateUserPipelineRelease(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, false)
	if err != nil {
		return nil, err
	}
	pipelineRelease.Recipe = pipeline.Recipe
	dbPipelineReleaseToCreate, err := s.PBToDBPipelineRelease(ctx, userUid, pipelineUid, pipelineRelease)
	if err != nil {
		return nil, err
	}
	if err := s.repository.CreateUserPipelineRelease(ctx, ownerPermalink, userPermalink, pipelineUid, dbPipelineReleaseToCreate); err != nil {
		return nil, err
	}

	dbCreatedPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, pipelineRelease.Id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, userUid, dbCreatedPipelineRelease, false)

}
func (s *service) ListUserPipelineReleases(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.PipelineRelease, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipelineReleases, ps, pt, err := s.repository.ListUserPipelineReleases(ctx, ownerPermalink, userPermalink, pipelineUid, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, userUid, dbPipelineReleases, isBasicView)
	return pbPipelineReleases, ps, pt, err
}

func (s *service) ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*pipelinePB.PipelineRelease, int64, string, error) {

	dbPipelineReleases, ps, pt, err := s.repository.ListPipelineReleasesAdmin(ctx, pageSize, pageToken, isBasicView, filter)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelineReleases, err := s.DBToPBPipelineReleasesAdmin(ctx, dbPipelineReleases, isBasicView)
	return pbPipelineReleases, ps, pt, err

}

func (s *service) GetUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, isBasicView bool) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, isBasicView)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, userUid, dbPipelineRelease, isBasicView)

}
func (s *service) GetUserPipelineReleaseByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, pipelineUid, uid, isBasicView)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, userUid, dbPipelineRelease, isBasicView)

}

func (s *service) UpdateUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, toUpdPipeline *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	// Validation: Pipeline existence
	if existingPipeline, _ := s.GetUserPipelineReleaseByID(ctx, ns, userUid, pipelineUid, id, true); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	pbPipelineReleaseToUpdate, err := s.PBToDBPipelineRelease(ctx, userUid, pipelineUid, toUpdPipeline)
	if err != nil {
		return nil, err
	}
	if err := s.repository.UpdateUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, pbPipelineReleaseToUpdate); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, toUpdPipeline.Id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, userUid, dbPipelineRelease, false)
}

func (s *service) UpdateUserPipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	// Validation: Pipeline existence
	existingPipeline, _ := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, true)
	if existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	if err := s.repository.UpdateUserPipelineReleaseIDByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, newID); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, newID, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, userUid, dbPipelineRelease, false)
}

func (s *service) DeleteUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}

	if err := s.DeleteResourceState(dbPipelineRelease.UID); err != nil {
		return err
	}

	// TODO
	return s.repository.DeleteUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id)
}

func (s *service) RestoreUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUid, false); existingPipeline == nil {
		return status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}
	existingPipeline.Recipe = dbPipelineRelease.Recipe

	if err := s.repository.UpdateUserPipelineByID(ctx, ownerPermalink, userPermalink, id, existingPipeline); err != nil {
		return err
	}

	return nil
}

func (s *service) SetDefaultUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUid, false); existingPipeline == nil {
		return status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	existingPipeline.DefaultReleaseUID = dbPipelineRelease.UID

	if err := s.repository.UpdateUserPipelineByID(ctx, ownerPermalink, userPermalink, existingPipeline.ID, existingPipeline); err != nil {
		return err
	}
	return nil
}

func (s *service) triggerPipeline(ctx context.Context, ownerPermalink string, recipe *datamodel.Recipe, pipelineId string, pipelineUid uuid.UUID, pipelineInputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {
	err := s.preTriggerPipeline(recipe, pipelineInputs)
	if err != nil {
		return nil, nil, err
	}

	var inputs [][]byte

	batchSize := len(pipelineInputs)

	for idx := range pipelineInputs {
		inputStruct := &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		}
		inputStruct.Fields["body"] = structpb.NewStructValue(pipelineInputs[idx])

		input, err := protojson.Marshal(inputStruct)
		if err != nil {
			return nil, nil, err
		}
		inputs = append(inputs, input)
	}

	dag, err := utils.GenerateDAG(recipe.Components)
	if err != nil {
		return nil, nil, err
	}

	orderedComp, err := dag.TopologicalSort()
	if err != nil {
		return nil, nil, err
	}

	inputCache := make([]map[string]interface{}, batchSize)
	outputCache := make([]map[string]interface{}, batchSize)
	computeTime := map[string]float32{}

	for idx := range inputs {
		inputCache[idx] = map[string]interface{}{}
		outputCache[idx] = map[string]interface{}{}
		var inputStruct map[string]interface{}
		err := json.Unmarshal(inputs[idx], &inputStruct)
		if err != nil {
			return nil, nil, err
		}

		inputCache[idx][orderedComp[0].Id] = inputStruct
		outputCache[idx][orderedComp[0].Id] = inputStruct
		computeTime[orderedComp[0].Id] = 0

	}

	responseCompId := ""
	for _, comp := range orderedComp[1:] {

		var compInputs []*structpb.Struct

		for idx := 0; idx < batchSize; idx++ {
			compInputTemplate := comp.Configuration
			compInputTemplateJson, err := protojson.Marshal(compInputTemplate)
			if err != nil {
				return nil, nil, err
			}

			var compInputTemplateStruct interface{}
			err = json.Unmarshal(compInputTemplateJson, &compInputTemplateStruct)
			if err != nil {
				return nil, nil, err
			}

			compInputStruct, err := utils.RenderInput(compInputTemplateStruct, outputCache[idx])
			if err != nil {
				return nil, nil, err
			}
			compInputJson, err := json.Marshal(compInputStruct)
			if err != nil {
				return nil, nil, err
			}

			compInput := &structpb.Struct{}
			err = protojson.Unmarshal([]byte(compInputJson), compInput)
			if err != nil {
				return nil, nil, err
			}

			inputCache[idx][comp.Id] = compInput
			compInputs = append(compInputs, compInput)
		}

		if comp.ResourceName != "" {

			start := time.Now()
			resp, err := s.connectorPublicServiceClient.ExecuteUserConnectorResource(
				utils.InjectOwnerToContextWithOwnerPermalink(
					metadata.AppendToOutgoingContext(ctx,
						"id", pipelineId,
						"uid", pipelineUid.String(),
						"owner", ownerPermalink,
						"trigger_id", pipelineTriggerId,
					),
					ownerPermalink),
				&connectorPB.ExecuteUserConnectorResourceRequest{
					Name:   comp.ResourceName,
					Inputs: compInputs,
				},
			)
			computeTime[comp.Id] = float32(time.Since(start).Seconds())
			if err != nil {
				return nil, nil, err
			}
			for idx := range resp.Outputs {

				outputJson, err := protojson.Marshal(resp.Outputs[idx])
				if err != nil {
					return nil, nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, nil, err
				}
				outputCache[idx][comp.Id] = outputStruct
			}

		}

		if comp.DefinitionName == "operator-definitions/end-operator" {
			responseCompId = comp.Id
			for idx := range compInputs {
				outputJson, err := protojson.Marshal(compInputs[idx])
				if err != nil {
					return nil, nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, nil, err
				}
				outputCache[idx][comp.Id] = outputStruct
			}
			computeTime[comp.Id] = 0

		}

	}

	pipelineOutputs := []*structpb.Struct{}
	for idx := 0; idx < batchSize; idx++ {
		pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		for key, value := range outputCache[idx][responseCompId].(map[string]interface{})["body"].(map[string]interface{}) {
			structVal, err := structpb.NewValue(value.(map[string]interface{})["value"])
			if err != nil {
				return nil, nil, err
			}
			pipelineOutput.Fields[key] = structVal

		}
		pipelineOutputs = append(pipelineOutputs, pipelineOutput)

	}
	var traces map[string]*pipelinePB.Trace
	if returnTraces {
		traces, err = utils.GenerateTraces(orderedComp, inputCache, outputCache, computeTime, batchSize)
		if err != nil {
			return nil, nil, err
		}
	}
	metadata := &pipelinePB.TriggerMetadata{
		Traces: traces,
	}

	return pipelineOutputs, metadata, nil
}

func (s *service) triggerAsyncPipeline(ctx context.Context, ownerPermalink string, recipe *datamodel.Recipe, pipelineId string, pipelineUid uuid.UUID, pipelineInputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	err := s.preTriggerPipeline(recipe, pipelineInputs)
	if err != nil {
		return nil, err
	}
	logger, _ := logger.GetZapLogger(ctx)

	inputBlobRedisKeys := []string{}
	for idx, input := range pipelineInputs {
		inputJson, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		inputBlobRedisKey := fmt.Sprintf("async_pipeline_request:%s:%d", pipelineTriggerId, idx)
		s.redisClient.Set(
			context.Background(),
			inputBlobRedisKey,
			inputJson,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
	}
	memo := map[string]interface{}{}
	memo["number_of_data"] = len(inputBlobRedisKeys)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerId,
		TaskQueue:                worker.TaskQueue,
		WorkflowExecutionTimeout: time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout) * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: config.Config.Server.Workflow.MaxWorkflowRetry,
		},
		Memo: memo,
	}

	we, err := s.temporalClient.ExecuteWorkflow(
		ctx,
		workflowOptions,
		"TriggerAsyncPipelineWorkflow",
		&worker.TriggerAsyncPipelineWorkflowRequest{
			PipelineInputBlobRedisKeys: inputBlobRedisKeys,
			PipelineId:                 pipelineId,
			PipelineUid:                pipelineUid,
			PipelineRecipe:             recipe,
			OwnerPermalink:             ownerPermalink,
			ReturnTraces:               returnTraces,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with WorkflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", pipelineTriggerId),
		Done: false,
	}, nil

}

func (s *service) TriggerUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, nil, err
	}
	recipe, err := s.recipePermalinkToName(userUid, dbPipeline.Recipe)
	if err != nil {
		return nil, nil, err
	}
	return s.triggerPipeline(ctx, ownerPermalink, recipe, dbPipeline.ID, dbPipeline.UID, inputs, pipelineTriggerId, returnTraces)

}

func (s *service) TriggerAsyncUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}
	recipe, err := s.recipePermalinkToName(userUid, dbPipeline.Recipe)
	if err != nil {
		return nil, err
	}
	return s.triggerAsyncPipeline(ctx, ownerPermalink, recipe, dbPipeline.ID, dbPipeline.UID, inputs, pipelineTriggerId, returnTraces)

}

func (s *service) TriggerUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return nil, nil, err
	}

	dbPipeline, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, pipelineUid, dbPipelineRelease.UID, false)
	if err != nil {
		return nil, nil, err
	}

	recipe, err := s.recipePermalinkToName(userUid, dbPipelineRelease.Recipe)
	if err != nil {
		return nil, nil, err
	}
	return s.triggerPipeline(ctx, ownerPermalink, recipe, dbPipeline.ID, dbPipeline.UID, inputs, pipelineTriggerId, returnTraces)
}

func (s *service) TriggerAsyncUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return nil, err
	}
	dbPipeline, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, pipelineUid, dbPipelineRelease.UID, false)
	if err != nil {
		return nil, err
	}
	recipe, err := s.recipePermalinkToName(userUid, dbPipelineRelease.Recipe)
	if err != nil {
		return nil, err
	}
	return s.triggerAsyncPipeline(ctx, ownerPermalink, recipe, dbPipeline.ID, dbPipeline.UID, inputs, pipelineTriggerId, returnTraces)
}
