package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"

	component "github.com/instill-ai/component/pkg/base"
	operator "github.com/instill-ai/operator/pkg"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the service, we'd better use uid as our function params

// Service interface
type Service interface {
	GetOperatorDefinitionById(ctx context.Context, defId string) (*pipelinePB.OperatorDefinition, error)
	ListOperatorDefinitions(ctx context.Context) []*pipelinePB.OperatorDefinition

	ListPipelines(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error)
	GetPipelineByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view pipelinePB.View) (*pipelinePB.Pipeline, error)
	CreateUserPipeline(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	ListUserPipelines(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error)
	GetUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view pipelinePB.View) (*pipelinePB.Pipeline, error)
	UpdateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, updatedPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	UpdateUserPipelineIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Pipeline, error)
	DeleteUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error
	ValidateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (*pipelinePB.Pipeline, error)
	GetUserPipelineDefaultReleaseUid(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (uuid.UUID, error)
	GetUserPipelineLatestReleaseUid(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (uuid.UUID, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view pipelinePB.View) (*pipelinePB.Pipeline, error)

	CreateUserPipelineRelease(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	ListUserPipelineReleases(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error)
	GetUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, view pipelinePB.View) (*pipelinePB.PipelineRelease, error)
	GetUserPipelineReleaseByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, uid uuid.UUID, view pipelinePB.View) (*pipelinePB.PipelineRelease, error)
	UpdateUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, updatedPipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	DeleteUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	RestoreUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	SetDefaultUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	UpdateUserPipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error)

	ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error)

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
	ConvertReleaseIdAlias(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineId string, releaseId string) (string, error)

	PBToDBPipeline(ctx context.Context, userUid uuid.UUID, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, view pipelinePB.View) (*pipelinePB.Pipeline, error)
	DBToPBPipelines(ctx context.Context, dbPipeline []*datamodel.Pipeline, view pipelinePB.View) ([]*pipelinePB.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, userUid uuid.UUID, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view pipelinePB.View, latestUUID uuid.UUID, defaultUUID uuid.UUID) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view pipelinePB.View, latestUUID uuid.UUID, defaultUUID uuid.UUID) ([]*pipelinePB.PipelineRelease, error)

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
	operator                      component.IOperator
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
) Service {
	logger, _ := logger.GetZapLogger(context.Background())
	return &service{
		repository:                    r,
		mgmtPrivateServiceClient:      u,
		connectorPublicServiceClient:  c,
		connectorPrivateServiceClient: cPrivate,
		controllerClient:              ct,
		redisClient:                   rc,
		temporalClient:                t,
		influxDBWriteClient:           i,
		operator:                      operator.Init(logger),
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randomStrWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GenerateShareCode() string {
	return randomStrWithCharset(32, charset)
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

	return "", uuid.Nil, status.Errorf(codes.Unauthenticated, "Unauthorized")
}

func (s *service) getCode(ctx context.Context) string {
	headerInstillCode := resource.GetRequestSingleHeader(ctx, constant.HeaderInstillCodeKey)
	return headerInstillCode

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

func (s *service) ConvertReleaseIdAlias(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineId string, releaseId string) (string, error) {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	// TODO: simplify these
	if releaseId == "default" {
		releaseUid, err := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipelineId)
		if err != nil {
			return "", err
		}
		dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, pipelineId, true, s.getCode(ctx))
		if err != nil {
			return "", err
		}
		dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, dbPipeline.UID, releaseUid, true)
		if err != nil {
			return "", err
		}
		return dbPipelineRelease.ID, nil
	} else if releaseId == "latest" {
		releaseUid, err := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipelineId)
		if err != nil {
			return "", err
		}
		dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, pipelineId, true, s.getCode(ctx))
		if err != nil {
			return "", err
		}
		dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, dbPipeline.UID, releaseUid, true)
		if err != nil {
			return "", err
		}
		return dbPipelineRelease.ID, nil
	}
	return releaseId, nil

}

func (s *service) GetOperatorDefinitionById(ctx context.Context, defId string) (*pipelinePB.OperatorDefinition, error) {
	return s.operator.GetOperatorDefinitionByID(defId)
}

func (s *service) ListOperatorDefinitions(ctx context.Context) []*pipelinePB.OperatorDefinition {
	return s.operator.ListOperatorDefinitions()
}

func (s *service) ListPipelines(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, userPermalink, pageSize, pageToken, view == pipelinePB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view)
	return pbPipelines, totalSize, nextPageToken, err

}

func (s *service) GetPipelineByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view pipelinePB.View) (*pipelinePB.Pipeline, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, userPermalink, uid, view == pipelinePB.View_VIEW_BASIC, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, view)
}

func (s *service) CreateUserPipeline(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pbPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.PBToDBPipeline(ctx, userUid, pbPipeline)

	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if dbPipeline.ShareCode == "" {
		dbPipeline.ShareCode = GenerateShareCode()
	}

	if err := s.repository.CreateUserPipeline(ctx, ownerPermalink, userPermalink, dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, dbPipeline.ID, false, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbCreatedPipeline, pipelinePB.View_VIEW_FULL)
}

func (s *service) ListUserPipelines(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelines, ps, pt, err := s.repository.ListUserPipelines(ctx, ownerPermalink, userPermalink, pageSize, pageToken, view == pipelinePB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view)
	return pbPipelines, ps, pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, pageSize, pageToken, view == pipelinePB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view)
	return pbPipelines, ps, pt, err

}

func (s *service) GetUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view pipelinePB.View) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, view == pipelinePB.View_VIEW_BASIC, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, view)
}

func (s *service) GetUserPipelineDefaultReleaseUid(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (uuid.UUID, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, true, s.getCode(ctx))
	if err != nil {
		return uuid.Nil, err
	}

	return dbPipeline.DefaultReleaseUID, nil
}

func (s *service) GetUserPipelineLatestReleaseUid(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (uuid.UUID, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, true, s.getCode(ctx))
	if err != nil {
		return uuid.Nil, err
	}

	dbPipelineRelease, err := s.repository.GetLatestUserPipelineRelease(ctx, ownerPermalink, userPermalink, dbPipeline.UID, true)
	if err != nil {
		return uuid.Nil, err
	}

	return dbPipelineRelease.UID, nil
}

func (s *service) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view pipelinePB.View) (*pipelinePB.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, uid, view == pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, view)

}

func (s *service) UpdateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, toUpdPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineToCreate, err := s.PBToDBPipeline(ctx, userUid, toUpdPipeline)
	if err != nil {
		return nil, err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, true, s.getCode(ctx)); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	if existingPipeline.ShareCode == "" {
		dbPipelineToCreate.ShareCode = GenerateShareCode()
	}

	if err := s.repository.UpdateUserPipelineByID(ctx, ownerPermalink, userPermalink, id, dbPipelineToCreate); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, toUpdPipeline.Id, false, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, pipelinePB.View_VIEW_FULL)
}

func (s *service) DeleteUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false, s.getCode(ctx))
	if err != nil {
		return err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	// TODO: pagination
	pipelineReleases, _, _, err := s.repository.ListUserPipelineReleases(ctx, ownerPermalink, userPermalink, dbPipeline.UID, 1000, "", false, filtering.Filter{}, false)
	if err != nil {
		return err
	}
	for _, pipelineRelease := range pipelineReleases {
		err := s.DeleteUserPipelineReleaseByID(ctx, ns, userUid, dbPipeline.UID, pipelineRelease.ID)
		if err != nil {
			return err
		}
	}

	return s.repository.DeleteUserPipelineByID(ctx, ownerPermalink, userPermalink, id)
}

func (s *service) ValidateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false, s.getCode(ctx))
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

	dbPipeline, err = s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, pipelinePB.View_VIEW_FULL)

}

func (s *service) UpdateUserPipelineIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	// Validation: Pipeline existence
	existingPipeline, _ := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, true, s.getCode(ctx))
	if existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}

	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	if err := s.repository.UpdateUserPipelineIDByID(ctx, ownerPermalink, userPermalink, id, newID); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, newID, false, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, pipelinePB.View_VIEW_FULL)
}

func (s *service) preTriggerPipeline(recipe *datamodel.Recipe, pipelineInputs []*structpb.Struct) error {

	var metadata []byte
	var err error
	for _, comp := range recipe.Components {
		if comp.DefinitionName == "operator-definitions/op-start" {
			schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
			schStruct.Fields["type"] = structpb.NewStringValue("object")
			schStruct.Fields["properties"] = structpb.NewStructValue(comp.Configuration.Fields["metadata"].GetStructValue())
			metadata, err = protojson.Marshal(schStruct)
			if err != nil {
				return err
			}
		}
	}

	sch, err := jsonschema.CompileString("", string(metadata))
	sch.Location = ""
	if err != nil {
		return err
	}

	for _, pipelineInput := range pipelineInputs {
		b, err := protojson.Marshal(pipelineInput)
		if err != nil {
			return err
		}
		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}

		if err := sch.Validate(v); err != nil {
			return err
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
		resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1alpha.TriggerUserPipelineResponse"
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

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, pipelinePB.View_VIEW_FULL)
	if err != nil {
		return nil, err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	pipelineRelease.Recipe = proto.Clone(pipeline.Recipe).(*pipelinePB.Recipe)
	pipelineRelease.Metadata = proto.Clone(pipeline.Metadata).(*structpb.Struct)

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
	// Add resource entry to controller
	if err := s.UpdateResourceState(dbCreatedPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbCreatedPipelineRelease, pipelinePB.View_VIEW_FULL, latestUUID, defaultUUID)

}
func (s *service) ListUserPipelineReleases(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipelineReleases, ps, pt, err := s.repository.ListUserPipelineReleases(ctx, ownerPermalink, userPermalink, pipelineUid, pageSize, pageToken, view == pipelinePB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, 0, "", err
	}
	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, latestUUID, defaultUUID)
	return pbPipelineReleases, ps, pt, err
}

func (s *service) ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, view pipelinePB.View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error) {

	dbPipelineReleases, ps, pt, err := s.repository.ListPipelineReleasesAdmin(ctx, pageSize, pageToken, view == pipelinePB.View_VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, uuid.Nil, uuid.Nil)
	return pbPipelineReleases, ps, pt, err

}

func (s *service) GetUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, view pipelinePB.View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, view == pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}
func (s *service) GetUserPipelineReleaseByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, uid uuid.UUID, view pipelinePB.View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, pipelineUid, uid, view == pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}

func (s *service) UpdateUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, toUpdPipeline *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	// Validation: Pipeline existence
	if existingPipeline, _ := s.GetUserPipelineReleaseByID(ctx, ns, userUid, pipelineUid, id, pipelinePB.View_VIEW_BASIC); existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
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

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	// Add resource entry to controller
	if err := s.UpdateResourceState(dbPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, pipelinePB.View_VIEW_FULL, latestUUID, defaultUUID)
}

func (s *service) UpdateUserPipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	// Validation: Pipeline existence
	existingPipeline, _ := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, true)
	if existingPipeline == nil {
		return nil, status.Errorf(codes.NotFound, "Pipeline id %s is not found", id)
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	if err := s.repository.UpdateUserPipelineReleaseIDByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, newID); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, newID, false)
	if err != nil {
		return nil, err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, pipelinePB.View_VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	// Add resource entry to controller
	if err := s.UpdateResourceState(dbPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}
	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, pipelinePB.View_VIEW_FULL, latestUUID, defaultUUID)
}

func (s *service) DeleteUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return status.Errorf(codes.PermissionDenied, "Permission Denied")
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
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return status.Errorf(codes.PermissionDenied, "Permission Denied")
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
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return status.Errorf(codes.PermissionDenied, "Permission Denied")
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

func (s *service) triggerPipeline(
	ctx context.Context,
	ownerPermalink string,
	recipe *datamodel.Recipe,
	pipelineId string,
	pipelineUid uuid.UUID,
	pipelineReleaseId string,
	pipelineReleaseUid uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerId string,
	returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	logger, _ := logger.GetZapLogger(ctx)

	recipe, err := s.dbRecipePermalinkToName(recipe)
	if err != nil {
		return nil, nil, err
	}
	err = s.preTriggerPipeline(recipe, pipelineInputs)
	if err != nil {
		return nil, nil, err
	}

	var inputs [][]byte

	batchSize := len(pipelineInputs)

	for idx := range pipelineInputs {
		inputStruct := structpb.NewStructValue(pipelineInputs[idx])
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

	memory := make([]map[string]interface{}, batchSize)
	computeTime := map[string]float32{}

	for idx := range inputs {
		memory[idx] = map[string]interface{}{}
		var inputStruct map[string]interface{}
		err := json.Unmarshal(inputs[idx], &inputStruct)
		if err != nil {
			return nil, nil, err
		}

		memory[idx][orderedComp[0].Id] = inputStruct
		computeTime[orderedComp[0].Id] = 0

		memory[idx]["global"], err = utils.GenerateGlobalValue(pipelineUid, recipe, ownerPermalink)
		if err != nil {
			return nil, nil, err
		}
	}

	responseCompId := ""
	for _, comp := range orderedComp[1:] {

		var compInputs []*structpb.Struct

		for idx := 0; idx < batchSize; idx++ {

			memory[idx][comp.Id] = map[string]interface{}{
				"input":  map[string]interface{}{},
				"output": map[string]interface{}{},
			}

			compInputTemplate := comp.Configuration
			// TODO: remove this hardcode injection
			if comp.DefinitionName == "connector-definitions/blockchain-numbers" {
				recipeByte, err := json.Marshal(recipe)
				if err != nil {
					return nil, nil, err
				}
				recipePb := &structpb.Struct{}
				err = protojson.Unmarshal(recipeByte, recipePb)
				if err != nil {
					return nil, nil, err
				}

				metadata, err := structpb.NewValue(map[string]interface{}{
					"pipeline": map[string]interface{}{
						"uid":    "{global.pipeline.uid}",
						"recipe": "{global.pipeline.recipe}",
					},
					"owner": map[string]interface{}{
						"uid": "{global.owner.uid}",
					},
				})
				if err != nil {
					return nil, nil, err
				}
				if compInputTemplate.Fields["input"].GetStructValue().Fields["custom"].GetStructValue() == nil {
					compInputTemplate.Fields["input"].GetStructValue().Fields["custom"] = structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{}})
				}
				compInputTemplate.Fields["input"].GetStructValue().Fields["custom"].GetStructValue().Fields["metadata"] = metadata
			}

			compInputTemplateJson, err := protojson.Marshal(compInputTemplate.Fields["input"].GetStructValue())
			if err != nil {
				return nil, nil, err
			}

			var compInputTemplateStruct interface{}
			err = json.Unmarshal(compInputTemplateJson, &compInputTemplateStruct)
			if err != nil {
				return nil, nil, err
			}

			compInputStruct, err := utils.RenderInput(compInputTemplateStruct, memory[idx])
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

			memory[idx][comp.Id].(map[string]interface{})["input"] = compInputStruct
			compInputs = append(compInputs, compInput)
		}

		task := ""
		if comp.Configuration.Fields["task"] != nil {
			task = comp.Configuration.Fields["task"].GetStringValue()
		}

		if utils.IsConnectorDefinition(comp.DefinitionName) && comp.ResourceName != "" {
			start := time.Now()
			resp, err := s.connectorPublicServiceClient.ExecuteUserConnectorResource(
				utils.InjectOwnerToContextWithOwnerPermalink(
					metadata.AppendToOutgoingContext(ctx,
						"id", pipelineId,
						"uid", pipelineUid.String(),
						"release_id", pipelineReleaseId,
						"release_uid", pipelineReleaseUid.String(),
						"owner", ownerPermalink,
						"trigger_id", pipelineTriggerId,
					),
					ownerPermalink),
				&connectorPB.ExecuteUserConnectorResourceRequest{
					Name:   comp.ResourceName,
					Task:   task,
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
				memory[idx][comp.Id].(map[string]interface{})["output"] = outputStruct
			}

		} else if comp.DefinitionName == "operator-definitions/op-end" {
			responseCompId = comp.Id
			computeTime[comp.Id] = 0
		} else if utils.IsOperatorDefinition(comp.DefinitionName) {

			op, err := s.operator.GetOperatorDefinitionByID(strings.Split(comp.DefinitionName, "/")[1])
			if err != nil {
				return nil, nil, err
			}

			execution, err := s.operator.CreateExecution(uuid.FromStringOrNil(op.Uid), task, comp.Configuration, logger)
			if err != nil {
				return nil, nil, err
			}
			start := time.Now()
			compOutputs, err := execution.ExecuteWithValidation(compInputs)

			computeTime[comp.Id] = float32(time.Since(start).Seconds())
			if err != nil {
				return nil, nil, err
			}
			for idx := range compOutputs {

				outputJson, err := protojson.Marshal(compOutputs[idx])
				if err != nil {
					return nil, nil, err
				}
				var outputStruct map[string]interface{}
				err = json.Unmarshal(outputJson, &outputStruct)
				if err != nil {
					return nil, nil, err
				}
				memory[idx][comp.Id].(map[string]interface{})["output"] = outputStruct
			}

		}

	}

	pipelineOutputs := []*structpb.Struct{}
	for idx := 0; idx < batchSize; idx++ {
		pipelineOutput := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		for key, value := range memory[idx][responseCompId].(map[string]interface{})["input"].(map[string]interface{}) {
			structVal, err := structpb.NewValue(value)
			if err != nil {
				return nil, nil, err
			}
			pipelineOutput.Fields[key] = structVal

		}
		pipelineOutputs = append(pipelineOutputs, pipelineOutput)

	}
	var traces map[string]*pipelinePB.Trace
	if returnTraces {
		traces, err = utils.GenerateTraces(orderedComp, memory, computeTime, batchSize)
		if err != nil {
			return nil, nil, err
		}
	}
	metadata := &pipelinePB.TriggerMetadata{
		Traces: traces,
	}

	return pipelineOutputs, metadata, nil
}

func (s *service) triggerAsyncPipeline(
	ctx context.Context,
	ownerPermalink string,
	recipe *datamodel.Recipe,
	pipelineId string,
	pipelineUid uuid.UUID,
	pipelineReleaseId string,
	pipelineReleaseUid uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerId string,
	returnTraces bool) (*longrunningpb.Operation, error) {

	recipe, err := s.dbRecipePermalinkToName(recipe)
	if err != nil {
		return nil, err
	}
	err = s.preTriggerPipeline(recipe, pipelineInputs)
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
			PipelineReleaseId:          pipelineReleaseId,
			PipelineReleaseUid:         pipelineReleaseUid,
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
	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false, s.getCode(ctx))
	if err != nil {
		return nil, nil, err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	return s.triggerPipeline(ctx, ownerPermalink, dbPipeline.Recipe, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerId, returnTraces)

}

func (s *service) TriggerAsyncUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, false, s.getCode(ctx))
	if err != nil {
		return nil, err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	return s.triggerAsyncPipeline(ctx, ownerPermalink, dbPipeline.Recipe, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerId, returnTraces)

}

func (s *service) TriggerUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return nil, nil, err
	}

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, userPermalink, pipelineUid, false, s.getCode(ctx))
	if err != nil {
		return nil, nil, err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}

	return s.triggerPipeline(ctx, ownerPermalink, dbPipelineRelease.Recipe, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerId, returnTraces)
}

func (s *service) TriggerAsyncUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, false)
	if err != nil {
		return nil, err
	}
	// TODO: use ACL
	if ownerPermalink != userPermalink {
		return nil, status.Errorf(codes.PermissionDenied, "Permission Denied")
	}
	dbPipeline, err := s.repository.GetPipelineByUID(ctx, userPermalink, pipelineUid, false, s.getCode(ctx))
	if err != nil {
		return nil, err
	}

	return s.triggerAsyncPipeline(ctx, ownerPermalink, dbPipelineRelease.Recipe, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerId, returnTraces)
}
