package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	rpcStatus "google.golang.org/genproto/googleapis/rpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	component "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/connector/pkg"
	operator "github.com/instill-ai/operator/pkg"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the service, we'd better use uid as our function params

// Service interface
type Service interface {
	GetOperatorDefinitionById(ctx context.Context, defId string) (*pipelinePB.OperatorDefinition, error)
	ListOperatorDefinitions(ctx context.Context) []*pipelinePB.OperatorDefinition

	ListPipelines(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error)
	GetPipelineByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)
	CreateUserPipeline(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	ListUserPipelines(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error)
	GetUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view View) (*pipelinePB.Pipeline, error)
	UpdateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, updatedPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	UpdateUserPipelineIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Pipeline, error)
	DeleteUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error
	ValidateUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (*pipelinePB.Pipeline, error)
	GetUserPipelineDefaultReleaseUid(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (uuid.UUID, error)
	GetUserPipelineLatestReleaseUid(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) (uuid.UUID, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)

	CreateUserPipelineRelease(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	ListUserPipelineReleases(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error)
	GetUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error)
	GetUserPipelineReleaseByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.PipelineRelease, error)
	UpdateUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, updatedPipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	DeleteUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	RestoreUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	SetDefaultUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string) error
	UpdateUserPipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error)

	ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error)

	// Controller APIs
	GetPipelineState(uid uuid.UUID) (*pipelinePB.State, error)
	UpdatePipelineState(uid uuid.UUID, state pipelinePB.State, progress *int32) error
	DeletePipelineState(uid uuid.UUID) error

	// Influx API

	TriggerUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error)

	TriggerUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error)

	WriteNewPipelineDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error

	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	GetRscNamespaceAndNameIDAndReleaseID(path string) (resource.Namespace, string, string, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	ConvertReleaseIdAlias(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineId string, releaseId string) (string, error)

	PBToDBPipeline(ctx context.Context, userUid uuid.UUID, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, view View) (*pipelinePB.Pipeline, error)
	DBToPBPipelines(ctx context.Context, dbPipeline []*datamodel.Pipeline, view View) ([]*pipelinePB.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, userUid uuid.UUID, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) ([]*pipelinePB.PipelineRelease, error)

	GetUser(ctx context.Context) (string, uuid.UUID, error)

	ListConnectorDefinitions(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter) ([]*pipelinePB.ConnectorDefinition, int64, string, error)
	GetConnectorByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view View, credentialMask bool) (*pipelinePB.Connector, error)
	GetConnectorDefinitionByID(ctx context.Context, id string, view View) (*pipelinePB.ConnectorDefinition, error)
	GetConnectorDefinitionByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.ConnectorDefinition, error)

	// Connector common
	ListConnectors(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int64, string, error)
	CreateUserConnector(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, connectorResource *pipelinePB.Connector) (*pipelinePB.Connector, error)
	ListUserConnectors(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int64, string, error)
	GetUserConnectorByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view View, credentialMask bool) (*pipelinePB.Connector, error)
	UpdateUserConnectorByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, connectorResource *pipelinePB.Connector) (*pipelinePB.Connector, error)
	UpdateUserConnectorIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Connector, error)
	UpdateUserConnectorStateByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, state pipelinePB.Connector_State) (*pipelinePB.Connector, error)
	DeleteUserConnectorByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error

	ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int64, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Connector, error)

	// Execute connector
	Execute(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, task string, inputs []*structpb.Struct) ([]*structpb.Struct, error)

	// Shared public/private method for checking connector's connection
	CheckConnectorByUID(ctx context.Context, connUID uuid.UUID) (*pipelinePB.Connector_State, error)

	// Controller custom service
	GetConnectorState(uid uuid.UUID) (*pipelinePB.Connector_State, error)
	UpdateConnectorState(uid uuid.UUID, state pipelinePB.Connector_State, progress *int32) error
	DeleteConnectorState(uid uuid.UUID) error

	// Influx API
	WriteNewConnectorDataPoint(ctx context.Context, data utils.ConnectorUsageMetricData, pipelineMetadata *structpb.Value) error

	// Helper functions
	RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct)
	KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct)
}

type service struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
	controllerClient         controllerPB.ControllerPrivateServiceClient
	redisClient              *redis.Client
	temporalClient           client.Client
	influxDBWriteClient      api.WriteAPI
	operator                 component.IOperator
	connector                component.IConnector
}

// NewService initiates a service instance
func NewService(
	r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	ct controllerPB.ControllerPrivateServiceClient,
	rc *redis.Client,
	t client.Client,
	i api.WriteAPI,
) Service {
	logger, _ := logger.GetZapLogger(context.Background())
	return &service{
		repository:               r,
		mgmtPrivateServiceClient: u,
		controllerClient:         ct,
		redisClient:              rc,
		temporalClient:           t,
		influxDBWriteClient:      i,
		operator:                 operator.Init(logger),
		connector:                connector.Init(logger, utils.GetConnectorOptions()),
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

func (s *service) ListPipelines(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, userPermalink, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view)
	return pbPipelines, totalSize, nextPageToken, err

}

func (s *service) GetPipelineByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, userPermalink, uid, view == VIEW_BASIC, s.getCode(ctx))
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

	return s.DBToPBPipeline(ctx, dbCreatedPipeline, VIEW_FULL)
}

func (s *service) ListUserPipelines(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelines, ps, pt, err := s.repository.ListUserPipelines(ctx, ownerPermalink, userPermalink, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view)
	return pbPipelines, ps, pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int64, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view)
	return pbPipelines, ps, pt, err

}

func (s *service) GetUserPipelineByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view View) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipeline, err := s.repository.GetUserPipelineByID(ctx, ownerPermalink, userPermalink, id, view == VIEW_BASIC, s.getCode(ctx))
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

func (s *service) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error) {

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, uid, view == VIEW_BASIC)
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

	return s.DBToPBPipeline(ctx, dbPipeline, VIEW_FULL)
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

	return s.DBToPBPipeline(ctx, dbPipeline, VIEW_FULL)

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

	return s.DBToPBPipeline(ctx, dbPipeline, VIEW_FULL)
}

func (s *service) preTriggerPipeline(recipe *datamodel.Recipe, pipelineInputs []*structpb.Struct) error {

	var metadata []byte
	var err error
	for _, comp := range recipe.Components {
		// op start
		if comp.DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3" {
			schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
			schStruct.Fields["type"] = structpb.NewStringValue("object")
			schStruct.Fields["properties"] = structpb.NewStructValue(comp.Configuration.Fields["metadata"].GetStructValue())
			err = component.CompileInstillAcceptFormats(schStruct)
			if err != nil {
				return err
			}
			err = component.CompileInstillFormat(schStruct)
			if err != nil {
				return err
			}
			metadata, err = protojson.Marshal(schStruct)
			if err != nil {
				return err
			}
		}
	}

	c := jsonschema.NewCompiler()
	c.RegisterExtension("instillAcceptFormats", component.InstillAcceptFormatsMeta, component.InstillAcceptFormatsCompiler{})
	c.RegisterExtension("instillFormat", component.InstillFormatMeta, component.InstillFormatCompiler{})

	if err := c.AddResource("schema.json", strings.NewReader(string(metadata))); err != nil {
		return err
	}

	sch, err := c.Compile("schema.json")

	if err != nil {
		return err
	}

	errors := []string{}

	for idx, pipelineInput := range pipelineInputs {
		b, err := protojson.Marshal(pipelineInput)
		if err != nil {
			errors = append(errors, fmt.Sprintf("inputs[%d]: data error", idx))
			continue
		}
		var v interface{}
		if err := json.Unmarshal(b, &v); err != nil {
			errors = append(errors, fmt.Sprintf("inputs[%d]: data error", idx))
			continue
		}

		if err = sch.Validate(v); err != nil {
			e := err.(*jsonschema.ValidationError)

			for _, valErr := range e.DetailedOutput().Errors {
				inputPath := fmt.Sprintf("%s/%d", "inputs", idx)
				component.FormatErrors(inputPath, valErr, &errors)
				for _, subValErr := range valErr.Errors {
					component.FormatErrors(inputPath, subValErr, &errors)
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("[Pipeline Trigger Data Error] %s", strings.Join(errors, "; "))
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

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, VIEW_FULL)
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
	if err := s.UpdatePipelineState(dbCreatedPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbCreatedPipelineRelease, VIEW_FULL, latestUUID, defaultUUID)

}
func (s *service) ListUserPipelineReleases(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbPipelineReleases, ps, pt, err := s.repository.ListUserPipelineReleases(ctx, ownerPermalink, userPermalink, pipelineUid, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, 0, "", err
	}
	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, latestUUID, defaultUUID)
	return pbPipelineReleases, ps, pt, err
}

func (s *service) ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int64, string, error) {

	dbPipelineReleases, ps, pt, err := s.repository.ListPipelineReleasesAdmin(ctx, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, uuid.Nil, uuid.Nil)
	return pbPipelineReleases, ps, pt, err

}

func (s *service) GetUserPipelineReleaseByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByID(ctx, ownerPermalink, userPermalink, pipelineUid, id, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}
func (s *service) GetUserPipelineReleaseByUID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pipelineUid uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbPipelineRelease, err := s.repository.GetUserPipelineReleaseByUID(ctx, ownerPermalink, userPermalink, pipelineUid, uid, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, VIEW_BASIC)
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
	if existingPipeline, _ := s.GetUserPipelineReleaseByID(ctx, ns, userUid, pipelineUid, id, VIEW_BASIC); existingPipeline == nil {
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

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	// Add resource entry to controller
	if err := s.UpdatePipelineState(dbPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, VIEW_FULL, latestUUID, defaultUUID)
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

	pipeline, err := s.GetPipelineByUID(ctx, userUid, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, err
	}
	// Add resource entry to controller
	if err := s.UpdatePipelineState(dbPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}
	latestUUID, _ := s.GetUserPipelineLatestReleaseUid(ctx, ns, userUid, pipeline.Id)
	defaultUUID, _ := s.GetUserPipelineDefaultReleaseUid(ctx, ns, userUid, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, VIEW_FULL, latestUUID, defaultUUID)
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

	if err := s.DeletePipelineState(dbPipelineRelease.UID); err != nil {
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

	err := s.preTriggerPipeline(recipe, pipelineInputs)
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
			// blockchain-number
			if comp.DefinitionName == "connector-definitions/70d8664a-d512-4517-a5e8-5d4da81756a7" {
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

			con, err := s.connector.GetConnectorDefinitionByUID(uuid.FromStringOrNil(strings.Split(comp.DefinitionName, "/")[1]))
			if err != nil {
				return nil, nil, err
			}

			dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, uuid.FromStringOrNil(strings.Split(comp.ResourceName, "/")[1]), false)
			if err != nil {
				return nil, nil, err
			}

			configuration := func() *structpb.Struct {
				if dbConnector.Configuration != nil {
					str := structpb.Struct{}
					err := str.UnmarshalJSON(dbConnector.Configuration)
					if err != nil {
						logger.Fatal(err.Error())
					}
					return &str
				}
				return nil
			}()

			execution, err := s.connector.CreateExecution(uuid.FromStringOrNil(con.Uid), task, configuration, logger)
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

		} else if comp.DefinitionName == "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516" {
			// op end
			responseCompId = comp.Id
			computeTime[comp.Id] = 0
		} else if utils.IsOperatorDefinition(comp.DefinitionName) {

			op, err := s.operator.GetOperatorDefinitionByID(strings.Split(comp.DefinitionName, "/")[1])
			if err != nil {
				return nil, nil, err
			}

			execution, err := s.operator.CreateExecution(uuid.FromStringOrNil(op.Uid), task, nil, logger)
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

func (s *service) RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.RemoveCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.KeepCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) ListConnectorDefinitions(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter) ([]*pipelinePB.ConnectorDefinition, int64, string, error) {

	logger, _ := logger.GetZapLogger(ctx)

	var err error
	prevLastUid := ""

	if pageToken != "" {
		_, prevLastUid, err = paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}
	}

	if pageSize == 0 {
		pageSize = repository.DefaultPageSize
	} else if pageSize > repository.MaxPageSize {
		pageSize = repository.MaxPageSize
	}

	unfilteredDefs := s.connector.ListConnectorDefinitions()

	// don't return definition with tombstone = true
	unfilteredDefsRemoveTombstone := []*pipelinePB.ConnectorDefinition{}
	for idx := range unfilteredDefs {
		if !unfilteredDefs[idx].Tombstone {
			unfilteredDefsRemoveTombstone = append(unfilteredDefsRemoveTombstone, unfilteredDefs[idx])
		}
	}
	unfilteredDefs = unfilteredDefsRemoveTombstone

	var defs []*pipelinePB.ConnectorDefinition
	if filter.CheckedExpr != nil {
		trans := repository.NewTranspiler(filter)
		expr, _ := trans.Transpile()
		typeMap := map[string]bool{}
		for idx := range expr.Vars {
			if idx == 0 {
				typeMap[string(expr.Vars[idx].(protoreflect.Name))] = true
			} else {
				typeMap[string(expr.Vars[idx].([]interface{})[0].(protoreflect.Name))] = true
			}

		}
		for idx := range unfilteredDefs {
			if _, ok := typeMap[unfilteredDefs[idx].Type.String()]; ok {
				defs = append(defs, unfilteredDefs[idx])
			}
		}

	} else {
		defs = unfilteredDefs
	}

	startIdx := 0
	lastUid := ""
	for idx, def := range defs {
		if def.Uid == prevLastUid {
			startIdx = idx + 1
			break
		}
	}

	page := []*pipelinePB.ConnectorDefinition{}
	for i := 0; i < int(pageSize) && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.ConnectorDefinition)
		page = append(page, def)
		lastUid = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUid)
	}

	pageDefs := []*pipelinePB.ConnectorDefinition{}

	for _, def := range page {
		def = proto.Clone(def).(*pipelinePB.ConnectorDefinition)
		if view == VIEW_BASIC {
			def.Spec = nil
		}
		def.VendorAttributes = nil
		pageDefs = append(pageDefs, def)
	}
	return pageDefs, int64(len(defs)), nextPageToken, err

}

func (s *service) GetConnectorByUID(ctx context.Context, userUid uuid.UUID, uid uuid.UUID, view View, credentialMask bool) (*pipelinePB.Connector, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbConnector, err := s.repository.GetConnectorByUID(ctx, userPermalink, uid, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, view, credentialMask)
}

func (s *service) GetConnectorDefinitionByID(ctx context.Context, id string, view View) (*pipelinePB.ConnectorDefinition, error) {

	def, err := s.connector.GetConnectorDefinitionByID(id)
	if err != nil {
		return nil, err
	}
	def = proto.Clone(def).(*pipelinePB.ConnectorDefinition)
	if view == VIEW_BASIC {
		def.Spec = nil
	}
	def.VendorAttributes = nil

	return def, nil
}
func (s *service) GetConnectorDefinitionByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.ConnectorDefinition, error) {

	def, err := s.connector.GetConnectorDefinitionByUID(uid)
	if err != nil {
		return nil, err
	}
	def = proto.Clone(def).(*pipelinePB.ConnectorDefinition)
	if view == VIEW_BASIC {
		def.Spec = nil
	}
	def.VendorAttributes = nil

	return def, nil
}

func (s *service) ListConnectors(ctx context.Context, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int64, string, error) {

	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListConnectors(ctx, userPermalink, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, totalSize, nextPageToken, err

}

func (s *service) CreateUserConnector(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, connectorResource *pipelinePB.Connector) (*pipelinePB.Connector, error) {

	logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	connDefResp, err := s.connector.GetConnectorDefinitionByID(strings.Split(connectorResource.ConnectorDefinitionName, "/")[1])
	if err != nil {
		return nil, err
	}

	connDefUID, err := uuid.FromString(connDefResp.GetUid())
	if err != nil {
		return nil, err
	}

	connConfig, err := connectorResource.GetConfiguration().MarshalJSON()
	if err != nil {

		return nil, err
	}

	connDesc := sql.NullString{
		String: connectorResource.GetDescription(),
		Valid:  len(connectorResource.GetDescription()) > 0,
	}

	dbConnectorToCreate := &datamodel.Connector{
		ID:                     connectorResource.Id,
		Owner:                  resource.UserUidToUserPermalink(userUid),
		ConnectorDefinitionUID: connDefUID,
		Tombstone:              false,
		Configuration:          connConfig,
		ConnectorType:          datamodel.ConnectorType(connDefResp.GetType()),
		Description:            connDesc,
		Visibility:             datamodel.ConnectorVisibility(connectorResource.Visibility),
	}

	if existingConnector, _ := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, dbConnectorToCreate.ID, true); existingConnector != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.AlreadyExists,
			"[service] create connector",
			"connectors",
			fmt.Sprintf("Connector id %s", dbConnectorToCreate.ID),
			dbConnectorToCreate.Owner,
			"Already exists",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}

	if err := s.repository.CreateUserConnector(ctx, ownerPermalink, userPermalink, dbConnectorToCreate); err != nil {
		return nil, err
	}

	// User desire state = DISCONNECTED
	if err := s.repository.UpdateUserConnectorStateByID(ctx, ownerPermalink, userPermalink, dbConnectorToCreate.ID, datamodel.ConnectorState(pipelinePB.Connector_STATE_DISCONNECTED)); err != nil {
		return nil, err
	}
	if err := s.UpdateConnectorState(dbConnectorToCreate.UID, pipelinePB.Connector_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, dbConnectorToCreate.ID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)

}

func (s *service) ListUserConnectors(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int64, string, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbConnectors, totalSize, nextPageToken, err := s.repository.ListUserConnectors(ctx, ownerPermalink, userPermalink, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)

	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, totalSize, nextPageToken, err

}

func (s *service) ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int64, string, error) {

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListConnectorsAdmin(ctx, pageSize, pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, totalSize, nextPageToken, err
}

func (s *service) GetUserConnectorByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, view View, credentialMask bool) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)
	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, id, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, view, credentialMask)
}

func (s *service) GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Connector, error) {

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, uid, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, view, true)
}

func (s *service) UpdateUserConnectorByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, connectorResource *pipelinePB.Connector) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnectorToUpdate, err := s.convertProtoToDatamodel(ctx, connectorResource)
	if err != nil {
		return nil, err
	}
	dbConnectorToUpdate.Owner = ownerPermalink

	if err := s.repository.UpdateUserConnectorByID(ctx, ownerPermalink, userPermalink, id, dbConnectorToUpdate); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.UpdateConnectorState(dbConnectorToUpdate.UID, pipelinePB.Connector_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, dbConnectorToUpdate.ID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)

}

func (s *service) DeleteUserConnectorByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string) error {
	// logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return err
	}

	// TODO
	// filter := fmt.Sprintf("recipe.components.resource_name:\"connector-resources/%s\"", dbConnector.UID)

	// pipeResp, err := s.pipelinePublicServiceClient.ListPipelines(s.injectUserToContext(context.Background(), ownerPermalink), &pipelinePB.ListPipelinesRequest{
	// 	Filter: &filter,
	// })
	// if err != nil {
	// 	return err
	// }

	// if len(pipeResp.Pipelines) > 0 {
	// 	var pipeIDs []string
	// 	for _, pipe := range pipeResp.Pipelines {
	// 		pipeIDs = append(pipeIDs, pipe.GetId())
	// 	}
	// 	st, err := sterr.CreateErrorPreconditionFailure(
	// 		"[service] delete connector",
	// 		[]*errdetails.PreconditionFailure_Violation{
	// 			{
	// 				Type:        "DELETE",
	// 				Subject:     fmt.Sprintf("id %s", id),
	// 				Description: fmt.Sprintf("The connector is still in use by pipeline: %s", strings.Join(pipeIDs, " ")),
	// 			},
	// 		})
	// 	if err != nil {
	// 		logger.Error(err.Error())
	// 	}
	// 	return st.Err()
	// }

	if err := s.DeleteConnectorState(dbConnector.UID); err != nil {
		return err
	}

	return s.repository.DeleteUserConnectorByID(ctx, ownerPermalink, userPermalink, id)
}

func (s *service) UpdateUserConnectorStateByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, state pipelinePB.Connector_State) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	// Validation: trigger and response connector cannot be disconnected
	conn, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	if conn.Tombstone {
		st, _ := sterr.CreateErrorPreconditionFailure(
			"[service] update connector state",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "STATE",
					Subject:     fmt.Sprintf("id %s", id),
					Description: "the connector definition is deprecated, you can not use anymore",
				},
			})
		return nil, st.Err()
	}

	switch state {
	case pipelinePB.Connector_STATE_CONNECTED:

		// Set connector state to user desire state
		if err := s.repository.UpdateUserConnectorStateByID(ctx, ownerPermalink, userPermalink, id, datamodel.ConnectorState(pipelinePB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		if err := s.UpdateConnectorState(conn.UID, pipelinePB.Connector_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}

	case pipelinePB.Connector_STATE_DISCONNECTED:

		if err := s.repository.UpdateUserConnectorStateByID(ctx, ownerPermalink, userPermalink, id, datamodel.ConnectorState(pipelinePB.Connector_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateConnectorState(conn.UID, pipelinePB.Connector_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)
}

func (s *service) UpdateUserConnectorIDByID(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, newID string) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	if err := s.repository.UpdateUserConnectorIDByID(ctx, ownerPermalink, userPermalink, id, newID); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, newID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)

}

func (s *service) Execute(ctx context.Context, ns resource.Namespace, userUid uuid.UUID, id string, task string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	logger, _ := logger.GetZapLogger(ctx)
	ownerPermalink := ns.String()
	userPermalink := resource.UserUidToUserPermalink(userUid)

	dbConnector, err := s.repository.GetUserConnectorByID(ctx, ownerPermalink, userPermalink, id, false)
	if err != nil {
		return nil, err
	}

	configuration := func() *structpb.Struct {
		if dbConnector.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbConnector.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			return &str
		}
		return nil
	}()

	con, err := s.connector.CreateExecution(dbConnector.ConnectorDefinitionUID, task, configuration, logger)

	if err != nil {
		return nil, err
	}

	return con.ExecuteWithValidation(inputs)
}

func (s *service) CheckConnectorByUID(ctx context.Context, connUID uuid.UUID) (*pipelinePB.Connector_State, error) {

	logger, _ := logger.GetZapLogger(ctx)

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, connUID, false)
	if err != nil {
		return pipelinePB.Connector_STATE_ERROR.Enum(), nil
	}

	configuration := func() *structpb.Struct {
		if dbConnector.Configuration != nil {
			str := structpb.Struct{}
			err := str.UnmarshalJSON(dbConnector.Configuration)
			if err != nil {
				logger.Fatal(err.Error())
			}
			return &str
		}
		return nil
	}()

	state, err := s.connector.Test(dbConnector.ConnectorDefinitionUID, configuration, logger)
	if err != nil {
		return pipelinePB.Connector_STATE_ERROR.Enum(), nil
	}

	switch state {
	case pipelinePB.Connector_STATE_CONNECTED:
		if err := s.UpdateConnectorState(dbConnector.UID, pipelinePB.Connector_STATE_CONNECTED, nil); err != nil {
			return pipelinePB.Connector_STATE_ERROR.Enum(), nil
		}
		return pipelinePB.Connector_STATE_CONNECTED.Enum(), nil
	case pipelinePB.Connector_STATE_ERROR:
		if err := s.UpdateConnectorState(dbConnector.UID, pipelinePB.Connector_STATE_ERROR, nil); err != nil {
			return pipelinePB.Connector_STATE_ERROR.Enum(), nil
		}
		return pipelinePB.Connector_STATE_ERROR.Enum(), nil
	default:
		if err := s.UpdateConnectorState(dbConnector.UID, pipelinePB.Connector_STATE_ERROR, nil); err != nil {
			return pipelinePB.Connector_STATE_ERROR.Enum(), nil
		}
		return pipelinePB.Connector_STATE_ERROR.Enum(), nil
	}

}
