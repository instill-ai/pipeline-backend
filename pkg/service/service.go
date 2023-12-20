package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gabriel-vasile/mimetype"
	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/pipeline-backend/pkg/worker"
	"github.com/instill-ai/x/errmsg"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	component "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/connector/pkg"
	operator "github.com/instill-ai/operator/pkg"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the service, we'd better use uid as our function params

// Service interface
type Service interface {
	GetOperatorDefinitionById(ctx context.Context, defId string) (*pipelinePB.OperatorDefinition, error)
	ListOperatorDefinitions(ctx context.Context) []*pipelinePB.OperatorDefinition

	ListPipelines(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetPipelineByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)
	CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	ListNamespacePipelines(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View) (*pipelinePB.Pipeline, error)
	UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, updatedPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Pipeline, error)
	DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error
	ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (*pipelinePB.Pipeline, error)
	GetNamespacePipelineDefaultReleaseUid(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error)
	GetNamespacePipelineLatestReleaseUid(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)

	CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error)
	GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error)
	GetNamespacePipelineReleaseByUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.PipelineRelease, error)
	UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, updatedPipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string) error
	RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string) error
	SetDefaultNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string) error
	UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error)

	ListPipelineReleasesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error)

	// Controller APIs
	GetPipelineState(uid uuid.UUID) (*pipelinePB.State, error)
	UpdatePipelineState(uid uuid.UUID, state pipelinePB.State, progress *int32) error
	DeletePipelineState(uid uuid.UUID) error

	// Influx API

	TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error)

	TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowId string) (*longrunningpb.Operation, error)

	WriteNewPipelineDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error

	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	GetRscNamespaceAndNameIDAndReleaseID(path string) (resource.Namespace, string, string, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	ConvertReleaseIdAlias(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineId string, releaseId string) (string, error)

	PBToDBPipeline(ctx context.Context, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, authUser *AuthUser, view View) (*pipelinePB.Pipeline, error)
	DBToPBPipelines(ctx context.Context, dbPipeline []*datamodel.Pipeline, authUser *AuthUser, view View) ([]*pipelinePB.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) ([]*pipelinePB.PipelineRelease, error)

	AuthenticateUser(ctx context.Context, allowVisitor bool) (authUser *AuthUser, err error)

	ListConnectorDefinitions(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter) ([]*pipelinePB.ConnectorDefinition, int32, string, error)
	GetConnectorByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View, credentialMask bool) (*pipelinePB.Connector, error)
	GetConnectorDefinitionByID(ctx context.Context, id string, view View) (*pipelinePB.ConnectorDefinition, error)
	GetConnectorDefinitionByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.ConnectorDefinition, error)

	// Connector common
	ListConnectors(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error)
	CreateNamespaceConnector(ctx context.Context, ns resource.Namespace, authUser *AuthUser, connector *pipelinePB.Connector) (*pipelinePB.Connector, error)
	ListNamespaceConnectors(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error)
	GetNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View, credentialMask bool) (*pipelinePB.Connector, error)
	UpdateNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, connector *pipelinePB.Connector) (*pipelinePB.Connector, error)
	UpdateNamespaceConnectorIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Connector, error)
	UpdateNamespaceConnectorStateByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, state pipelinePB.Connector_State) (*pipelinePB.Connector, error)
	DeleteNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error

	ListConnectorsAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Connector, error)

	// Execute connector
	Execute(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, task string, inputs []*structpb.Struct) ([]*structpb.Struct, error)

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
	mgmtPublicServiceClient  mgmtPB.MgmtPublicServiceClient
	controllerClient         controllerPB.ControllerPrivateServiceClient
	redisClient              *redis.Client
	temporalClient           client.Client
	influxDBWriteClient      api.WriteAPI
	operator                 component.IOperator
	connector                component.IConnector
	aclClient                *acl.ACLClient
}

// NewService initiates a service instance
func NewService(
	r repository.Repository,
	u mgmtPB.MgmtPrivateServiceClient,
	m mgmtPB.MgmtPublicServiceClient,
	ct controllerPB.ControllerPrivateServiceClient,
	rc *redis.Client,
	t client.Client,
	i api.WriteAPI,
	acl *acl.ACLClient,
) Service {
	logger, _ := logger.GetZapLogger(context.Background())
	return &service{
		repository:               r,
		mgmtPrivateServiceClient: u,
		mgmtPublicServiceClient:  m,
		controllerClient:         ct,
		redisClient:              rc,
		temporalClient:           t,
		influxDBWriteClient:      i,
		operator:                 operator.Init(logger),
		connector:                connector.Init(logger, utils.GetConnectorOptions()),
		aclClient:                acl,
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

type AuthUser struct {
	IsVisitor bool
	UID       uuid.UUID
	ID        string
}

func (a AuthUser) GetACLType() string {
	if a.IsVisitor {
		return "visitor"
	} else {
		return "user"
	}
}

func (a AuthUser) Permalink() string {
	if a.IsVisitor {
		return fmt.Sprintf("visitors/%s", a.UID)
	} else {
		return fmt.Sprintf("users/%s", a.UID)
	}
}

func (s *service) AuthenticateUser(ctx context.Context, allowVisitor bool) (authUser *AuthUser, err error) {
	// Verify if "jwt-sub" is in the header
	headerCtxUserUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	if headerCtxUserUID != "" {
		if allowVisitor && strings.HasPrefix(headerCtxUserUID, "visitor:") {
			_, err := uuid.FromString(strings.Split(headerCtxUserUID, ":")[1])
			if err != nil {
				return nil, ErrUnauthenticated
			}
			return &AuthUser{
				UID:       uuid.FromStringOrNil(strings.Split(headerCtxUserUID, ":")[1]),
				IsVisitor: true,
			}, nil
		} else {
			_, err := uuid.FromString(headerCtxUserUID)
			if err != nil {
				return nil, ErrUnauthenticated
			}
			resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: "users/" + headerCtxUserUID})
			if err != nil {
				return nil, ErrUnauthenticated
			}
			return &AuthUser{
				ID:        resp.User.Id,
				UID:       uuid.FromStringOrNil(headerCtxUserUID),
				IsVisitor: false,
			}, nil
		}

	}

	return nil, ErrUnauthenticated

}

func (s *service) getCode(ctx context.Context) string {
	headerInstillCode := resource.GetRequestSingleHeader(ctx, constant.HeaderInstillCodeKey)
	return headerInstillCode

}

func (s *service) ConvertOwnerPermalinkToName(permalink string) (string, error) {
	if strings.HasPrefix(permalink, "users") {
		userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		return fmt.Sprintf("users/%s", userResp.User.Id), nil
	} else {
		userResp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(context.Background(), &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		return fmt.Sprintf("organizations/%s", userResp.Organization.Id), nil
	}
}

func (s *service) FetchOwnerWithPermalink(permalink string) (*structpb.Struct, error) {
	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("FetchOwnerWithPermalink error")
		}
		owner := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		owner.Fields["profile_avatar"] = structpb.NewStringValue(resp.GetUser().GetProfileAvatar())
		owner.Fields["profile_data"] = structpb.NewStructValue(resp.GetUser().GetProfileData())

		return owner, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(context.Background(), &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("FetchOwnerWithPermalink error")
		}
		owner := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		owner.Fields["profile_avatar"] = structpb.NewStringValue(resp.GetOrganization().GetProfileAvatar())
		owner.Fields["profile_data"] = structpb.NewStructValue(resp.GetOrganization().GetProfileData())

		return owner, nil
	}
}

func (s *service) ConvertOwnerNameToPermalink(name string) (string, error) {
	if strings.HasPrefix(name, "users") {
		userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtPB.GetUserAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("ConvertOwnerNameToPermalink error %w", err)
		}
		return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.GetOrganizationAdmin(context.Background(), &mgmtPB.GetOrganizationAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("ConvertOwnerNameToPermalink error %w", err)
		}
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Uid), nil
	}
}

func (s *service) GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error) {

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.ConvertOwnerNameToPermalink(fmt.Sprintf("%s/%s", splits[0], splits[1]))

	if err != nil {
		return resource.Namespace{}, "", fmt.Errorf("namespace error %w", err)
	}
	if len(splits) < 4 {
		return resource.Namespace{
			NsType: resource.NamespaceType(splits[0]),
			NsID:   splits[1],
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
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
			NsID:   splits[1],
			NsUid:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
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

func (s *service) ConvertReleaseIdAlias(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineId string, releaseId string) (string, error) {
	ownerPermalink := ns.String()

	// TODO: simplify these
	if releaseId == "default" {
		releaseUid, err := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipelineId)
		if err != nil {
			return "", err
		}
		dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineId, true)
		if err != nil {
			return "", err
		}
		dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByUID(ctx, ownerPermalink, dbPipeline.UID, releaseUid, true)
		if err != nil {
			return "", err
		}
		return dbPipelineRelease.ID, nil
	} else if releaseId == "latest" {
		releaseUid, err := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipelineId)
		if err != nil {
			return "", err
		}
		dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineId, true)
		if err != nil {
			return "", err
		}
		dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByUID(ctx, ownerPermalink, dbPipeline.UID, releaseUid, true)
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

func (s *service) ListPipelines(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	uidAllowList, err := s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, int64(pageSize), pageToken, view == VIEW_BASIC, filter, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, authUser, view)
	return pbPipelines, int32(totalSize), nextPageToken, err

}

func (s *service) GetPipelineByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error) {

	if granted, err := s.aclClient.CheckPermission("pipeline", uid, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, uid, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, view)
}

func (s *service) checkPrivatePipelineQuota(ctx context.Context, ns resource.Namespace, dbPipeline *datamodel.Pipeline, quota int) error {

	if val, ok := dbPipeline.Sharing.Users["*/*"]; ok && val.Enabled {
		return nil
	}
	privateCount := 0
	// TODO: optimize this
	pageToken := ""
	var err error
	var pipelines []*datamodel.Pipeline
	for {
		pipelines, _, pageToken, err = s.repository.ListNamespacePipelines(ctx, ns.String(), int64(100), pageToken, true, filtering.Filter{}, nil, false)
		if err != nil {
			return err
		}
		for _, pipeline := range pipelines {

			if _, ok := pipeline.Sharing.Users["*/*"]; ok {
				if !pipeline.Sharing.Users["*/*"].Enabled {
					privateCount += 1
				}
			} else {
				privateCount += 1
			}

		}
		if pageToken == "" {
			break
		}
	}

	if privateCount >= quota {
		return ErrNamespacePrivatePipelineQuotaExceed
	}

	return nil
}

func (s *service) CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pbPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("organizations/%s", ns.NsID)})
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			if resp.Subscription.Plan == "inactive" {
				return nil, status.Errorf(codes.FailedPrecondition, "the organization subscription is not active")
			}
		}
	}

	ownerPermalink := ns.String()

	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission("organization", ns.NsUid, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "member")
		if err != nil {
			return nil, err
		}
		if !granted {
			return nil, ErrNoPermission
		}
	} else {
		if ns.NsUid != authUser.UID {
			return nil, ErrNoPermission
		}
	}

	dbPipeline, err := s.PBToDBPipeline(ctx, pbPipeline)
	if err != nil {
		return nil, err
	}

	quota := -1

	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			quota = int(resp.Subscription.Quota.PrivatePipeline.Quota)
		}
	} else {
		resp, err := s.mgmtPublicServiceClient.GetUserSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetUserSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			quota = int(resp.Subscription.Quota.PrivatePipeline.Quota)
		}
	}

	if quota > -1 {
		err = s.checkPrivatePipelineQuota(ctx, ns, dbPipeline, quota)
		if err != nil {
			return nil, err
		}
	}

	if dbPipeline.ShareCode == "" {
		dbPipeline.ShareCode = GenerateShareCode()
	}

	if err := s.repository.CreateNamespacePipeline(ctx, ownerPermalink, dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, dbPipeline.ID, false)
	if err != nil {
		return nil, err
	}
	ownerType := string(ns.NsType)[0 : len(string(ns.NsType))-1]
	ownerUID := ns.NsUid
	err = s.aclClient.SetOwner("pipeline", dbCreatedPipeline.UID, ownerType, ownerUID)
	if err != nil {
		return nil, err
	}
	// TODO: use OpenFGA as single source of truth
	err = s.aclClient.SetPipelinePermissionMap(dbCreatedPipeline)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbCreatedPipeline, authUser, VIEW_FULL)
}

func (s *service) ListNamespacePipelines(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	ownerPermalink := ns.String()

	uidAllowList, err := s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	dbPipelines, ps, pt, err := s.repository.ListNamespacePipelines(ctx, ownerPermalink, int64(pageSize), pageToken, view == VIEW_BASIC, filter, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, authUser, view)
	return pbPipelines, int32(ps), pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, int64(pageSize), pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, nil, view)
	return pbPipelines, int32(ps), pt, err

}

func (s *service) GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, view == VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, view)
}

func (s *service) GetNamespacePipelineDefaultReleaseUid(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true)
	if err != nil {
		return uuid.Nil, err
	}

	return dbPipeline.DefaultReleaseUID, nil
}

func (s *service) GetNamespacePipelineLatestReleaseUid(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true)
	if err != nil {
		return uuid.Nil, err
	}

	dbPipelineRelease, err := s.repository.GetLatestNamespacePipelineRelease(ctx, ownerPermalink, dbPipeline.UID, true)
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

	return s.DBToPBPipeline(ctx, dbPipeline, nil, view)

}

func (s *service) UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, toUpdPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()

	dbPipelineToUpdate, err := s.PBToDBPipeline(ctx, toUpdPipeline)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipelineToUpdate.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipelineToUpdate.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true); existingPipeline == nil {
		return nil, err
	}

	if existingPipeline.ShareCode == "" {
		dbPipelineToUpdate.ShareCode = GenerateShareCode()
	}

	quota := -1
	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			quota = int(resp.Subscription.Quota.PrivatePipeline.Quota)
		}
	} else {
		resp, err := s.mgmtPublicServiceClient.GetUserSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetUserSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			quota = int(resp.Subscription.Quota.PrivatePipeline.Quota)
		}
	}

	if quota > -1 {
		isPublic := false
		oriPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, toUpdPipeline.Id, false)
		if err != nil {
			return nil, err
		}
		if isPublic, err = s.aclClient.CheckPublicExecutable("pipeline", oriPipeline.UID); err != nil {
			return nil, err
		}

		if isPublic {
			err = s.checkPrivatePipelineQuota(ctx, ns, dbPipelineToUpdate, quota)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := s.repository.UpdateNamespacePipelineByID(ctx, ownerPermalink, id, dbPipelineToUpdate); err != nil {
		return nil, err
	}

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, toUpdPipeline.Id, false)
	if err != nil {
		return nil, err
	}

	// TODO: use OpenFGA as single source of truth
	err = s.aclClient.SetPipelinePermissionMap(dbPipeline)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, VIEW_FULL)
}

func (s *service) DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error {
	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	// TODO: pagination
	pipelineReleases, _, _, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, dbPipeline.UID, 1000, "", false, filtering.Filter{}, false)
	if err != nil {
		return err
	}
	for _, pipelineRelease := range pipelineReleases {
		err := s.DeleteNamespacePipelineReleaseByID(ctx, ns, authUser, dbPipeline.UID, pipelineRelease.ID)
		if err != nil {
			return err
		}
	}

	err = s.aclClient.Purge("pipeline", dbPipeline.UID)
	if err != nil {
		return err
	}
	return s.repository.DeleteNamespacePipelineByID(ctx, ownerPermalink, id)
}

func (s *service) ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
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

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, VIEW_FULL)

}

func (s *service) UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.String()

	// Validation: Pipeline existence
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	if err := s.repository.UpdateNamespacePipelineIDByID(ctx, ownerPermalink, id, newID); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, newID, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, VIEW_FULL)
}

func (s *service) preTriggerPipeline(ctx context.Context, isPublic bool, ns resource.Namespace, authUser *AuthUser, recipe *datamodel.Recipe, pipelineInputs []*structpb.Struct) error {

	batchSize := len(pipelineInputs)
	if batchSize > constant.MaxBatchSize {
		return ErrExceedMaxBatchSize
	}
	if isPublic {
		value, err := s.redisClient.Get(context.Background(), fmt.Sprintf("user_rate_limit:user:%s", authUser.UID)).Result()
		// TODO: use a more robust way to check key exist
		if !errors.Is(err, redis.Nil) {
			requestLeft, _ := strconv.ParseInt(value, 10, 64)
			if requestLeft <= 0 {
				return ErrRateLimiting
			} else {
				_ = s.redisClient.Decr(context.Background(), fmt.Sprintf("user_rate_limit:user:%s", authUser.UID))
			}
		}
	} else {
		if ns.NsType == resource.Organization {
			resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
				metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
				&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
			)
			if err != nil {
				s, ok := status.FromError(err)
				if !ok {
					return err
				}
				if s.Code() != codes.Unimplemented {
					return err
				}
			} else {
				if resp.Subscription.Quota.PrivatePipelineTrigger.Quota != -1 && resp.Subscription.Quota.PrivatePipelineTrigger.Remain-int32(batchSize) < 0 {
					return ErrNamespaceTriggerQuotaExceed
				}
			}

		} else {
			resp, err := s.mgmtPublicServiceClient.GetUserSubscription(
				metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
				&mgmtPB.GetUserSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
			)
			if err != nil {
				s, ok := status.FromError(err)
				if !ok {
					return err
				}
				if s.Code() != codes.Unimplemented {
					return err
				}
			} else {
				if resp.Subscription.Quota.PrivatePipelineTrigger.Quota != -1 && resp.Subscription.Quota.PrivatePipelineTrigger.Remain-int32(batchSize) < 0 {
					return ErrNamespaceTriggerQuotaExceed
				}
			}
		}
	}

	var metadata []byte

	instillFormatMap := map[string]string{}
	for _, comp := range recipe.Components {
		// op start
		if comp.DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3" {
			schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
			schStruct.Fields["type"] = structpb.NewStringValue("object")
			schStruct.Fields["properties"] = structpb.NewStructValue(comp.Configuration.Fields["metadata"].GetStructValue())
			for k, v := range comp.Configuration.Fields["metadata"].GetStructValue().Fields {
				instillFormatMap[k] = v.GetStructValue().Fields["instillFormat"].GetStringValue()
			}
			err := component.CompileInstillAcceptFormats(schStruct)
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
		var i interface{}
		if err := json.Unmarshal(b, &i); err != nil {
			errors = append(errors, fmt.Sprintf("inputs[%d]: data error", idx))
			continue
		}

		m := i.(map[string]interface{})

		for k := range m {
			switch s := m[k].(type) {
			case string:
				if instillFormatMap[k] != "string" {
					if !strings.HasPrefix(s, "data:") {
						b, err := base64.StdEncoding.DecodeString(s)
						if err != nil {
							return fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
						}
						mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]
						pipelineInput.Fields[k] = structpb.NewStringValue(fmt.Sprintf("data:%s;base64,%s", mimeType, s))
					}
				}
			case []interface{}:
				if instillFormatMap[k] != "array:string" {
					for idx := range s {
						switch item := s[idx].(type) {
						case string:
							if !strings.HasPrefix(item, "data:") {
								b, err := base64.StdEncoding.DecodeString(item)
								if err != nil {
									return fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
								}
								mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]
								pipelineInput.Fields[k].GetListValue().GetValues()[idx] = structpb.NewStringValue(fmt.Sprintf("data:%s;base64,%s", mimeType, item))
							}
						}

					}
				}
			}
		}

		if err = sch.Validate(m); err != nil {
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
		resp.TypeUrl = "buf.build/instill-ai/protobufs/vdp.pipeline.v1beta.TriggerUserPipelineResponse"
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

func (s *service) CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_FULL)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	pipelineRelease.Recipe = proto.Clone(pipeline.Recipe).(*pipelinePB.Recipe)
	pipelineRelease.Metadata = proto.Clone(pipeline.Metadata).(*structpb.Struct)

	dbPipelineReleaseToCreate, err := s.PBToDBPipelineRelease(ctx, pipelineUid, pipelineRelease)
	if err != nil {
		return nil, err
	}

	if err := s.repository.CreateNamespacePipelineRelease(ctx, ownerPermalink, pipelineUid, dbPipelineReleaseToCreate); err != nil {
		return nil, err
	}

	dbCreatedPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, pipelineRelease.Id, false)
	if err != nil {
		return nil, err
	}
	// Add resource entry to controller
	if err := s.UpdatePipelineState(dbCreatedPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbCreatedPipelineRelease, VIEW_FULL, latestUUID, defaultUUID)

}
func (s *service) ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error) {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, 0, "", err
	} else if !granted {
		return nil, 0, "", ErrNotFound
	}

	dbPipelineReleases, ps, pt, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, pipelineUid, int64(pageSize), pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipeline.Id)

	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, latestUUID, defaultUUID)
	return pbPipelineReleases, int32(ps), pt, err
}

func (s *service) ListPipelineReleasesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error) {

	dbPipelineReleases, ps, pt, err := s.repository.ListPipelineReleasesAdmin(ctx, int64(pageSize), pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, uuid.Nil, uuid.Nil)
	return pbPipelineReleases, int32(ps), pt, err

}

func (s *service) GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}
func (s *service) GetNamespacePipelineReleaseByUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByUID(ctx, ownerPermalink, pipelineUid, uid, view == VIEW_BASIC)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}

func (s *service) UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, toUpdPipeline *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	if _, err := s.GetNamespacePipelineReleaseByID(ctx, ns, authUser, pipelineUid, id, VIEW_BASIC); err != nil {
		return nil, err
	}

	pbPipelineReleaseToUpdate, err := s.PBToDBPipelineRelease(ctx, pipelineUid, toUpdPipeline)
	if err != nil {
		return nil, err
	}
	if err := s.repository.UpdateNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, pbPipelineReleaseToUpdate); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, toUpdPipeline.Id, false)
	if err != nil {
		return nil, err
	}

	// Add resource entry to controller
	if err := s.UpdatePipelineState(dbPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, VIEW_FULL, latestUUID, defaultUUID)
}

func (s *service) UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	// Validation: Pipeline existence
	_, err = s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, true)
	if err != nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespacePipelineReleaseIDByID(ctx, ownerPermalink, pipelineUid, id, newID); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, newID, false)
	if err != nil {
		return nil, err
	}

	// Add resource entry to controller
	if err := s.UpdatePipelineState(dbPipelineRelease.UID, pipelinePB.State_STATE_ACTIVE, nil); err != nil {
		return nil, err
	}
	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUid(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, VIEW_FULL, latestUUID, defaultUUID)
}

func (s *service) DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}

	if err := s.DeletePipelineState(dbPipelineRelease.UID); err != nil {
		return err
	}

	return s.repository.DeleteNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id)
}

func (s *service) RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string) error {
	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if _, err = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUid, false); err != nil {
		return err
	}
	existingPipeline.Recipe = dbPipelineRelease.Recipe

	if err := s.repository.UpdateNamespacePipelineByID(ctx, ownerPermalink, id, existingPipeline); err != nil {
		return err
	}

	return nil
}

func (s *service) SetDefaultNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string) error {

	ownerPermalink := ns.String()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUid, VIEW_BASIC)
	if err != nil {
		return ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, err = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUid, false); err != nil {
		return err
	}

	existingPipeline.DefaultReleaseUID = dbPipelineRelease.UID

	if err := s.repository.UpdateNamespacePipelineByID(ctx, ownerPermalink, existingPipeline.ID, existingPipeline); err != nil {
		return err
	}
	return nil
}

// TODO: share the code with worker/workflow.go
func (s *service) triggerPipeline(
	ctx context.Context,
	ns resource.Namespace,
	authUser *AuthUser,
	recipe *datamodel.Recipe,
	isPublic bool,
	pipelineId string,
	pipelineUid uuid.UUID,
	pipelineReleaseId string,
	pipelineReleaseUid uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerId string,
	returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	logger, _ := logger.GetZapLogger(ctx)

	err := s.preTriggerPipeline(ctx, isPublic, ns, authUser, recipe, pipelineInputs)
	if err != nil {
		return nil, nil, err
	}

	inputBlobRedisKeys := []string{}
	for idx, input := range pipelineInputs {
		inputJson, err := protojson.Marshal(input)
		if err != nil {
			return nil, nil, err
		}

		inputBlobRedisKey := fmt.Sprintf("async_pipeline_request:%s:%d", pipelineTriggerId, idx)
		s.redisClient.Set(
			context.Background(),
			inputBlobRedisKey,
			inputJson,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
		defer s.redisClient.Del(context.Background(), inputBlobRedisKey)
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
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowRequest{
			PipelineInputBlobRedisKeys: inputBlobRedisKeys,
			PipelineId:                 pipelineId,
			PipelineUid:                pipelineUid,
			PipelineReleaseId:          pipelineReleaseId,
			PipelineReleaseUid:         pipelineReleaseUid,
			PipelineRecipe:             recipe,
			OwnerPermalink:             ns.String(),
			UserPermalink:              authUser.Permalink(),
			ReturnTraces:               returnTraces,
			Mode:                       mgmtPB.Mode_MODE_SYNC,
			IsPublic:                   isPublic,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, nil, err
	}

	var result *worker.TriggerPipelineWorkflowResponse
	err = we.Get(context.Background(), &result)
	if err != nil {
		var applicationErr *temporal.ApplicationError
		if errors.As(err, &applicationErr) {
			var details worker.EndUserErrorDetails
			if dErr := applicationErr.Details(&details); dErr == nil && details.Message != "" {
				err = errmsg.AddMessage(err, details.Message)
			}
		}

		return nil, nil, err
	}
	pipelineResp := &pipelinePB.TriggerUserPipelineResponse{}

	blob, err := s.redisClient.Get(context.Background(), result.OutputBlobRedisKey).Bytes()
	if err != nil {
		return nil, nil, err
	}
	s.redisClient.Del(context.Background(), result.OutputBlobRedisKey)

	err = protojson.Unmarshal(blob, pipelineResp)
	if err != nil {
		return nil, nil, err
	}

	return pipelineResp.Outputs, pipelineResp.Metadata, nil
}

func (s *service) triggerAsyncPipeline(
	ctx context.Context,
	ns resource.Namespace,
	authUser *AuthUser,
	recipe *datamodel.Recipe,
	isPublic bool,
	pipelineId string,
	pipelineUid uuid.UUID,
	pipelineReleaseId string,
	pipelineReleaseUid uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerId string,
	returnTraces bool) (*longrunningpb.Operation, error) {

	err := s.preTriggerPipeline(ctx, isPublic, ns, authUser, recipe, pipelineInputs)
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
		"TriggerPipelineWorkflow",
		&worker.TriggerPipelineWorkflowRequest{
			PipelineInputBlobRedisKeys: inputBlobRedisKeys,
			PipelineId:                 pipelineId,
			PipelineUid:                pipelineUid,
			PipelineReleaseId:          pipelineReleaseId,
			PipelineReleaseUid:         pipelineReleaseUid,
			PipelineRecipe:             recipe,
			OwnerPermalink:             ns.String(),
			UserPermalink:              authUser.Permalink(),
			ReturnTraces:               returnTraces,
			Mode:                       mgmtPB.Mode_MODE_ASYNC,
			IsPublic:                   isPublic,
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

func (s *service) TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "executor"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNoPermission
	}

	isPublic := false
	if isPublic, err = s.aclClient.CheckPublicExecutable("pipeline", dbPipeline.UID); err != nil {
		return nil, nil, err
	}

	return s.triggerPipeline(ctx, ns, authUser, dbPipeline.Recipe, isPublic, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerId, returnTraces)

}

func (s *service) TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	isPublic := false
	if isPublic, err = s.aclClient.CheckPublicExecutable("pipeline", dbPipeline.UID); err != nil {
		return nil, err
	}

	return s.triggerAsyncPipeline(ctx, ns, authUser, dbPipeline.Recipe, isPublic, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerId, returnTraces)

}

func (s *service) TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUid, false)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "executor"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, false)
	if err != nil {
		return nil, nil, err
	}

	isPublic := false
	if isPublic, err = s.aclClient.CheckPublicExecutable("pipeline", dbPipeline.UID); err != nil {
		return nil, nil, err
	}

	plan := ""
	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, nil, err
			}
		} else {
			plan = resp.Subscription.Plan
		}
	} else {
		resp, err := s.mgmtPublicServiceClient.GetUserSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetUserSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, nil, err
			}
		} else {
			plan = resp.Subscription.Plan
		}
	}

	latestReleaseUID, err := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, dbPipeline.ID)
	if err != nil {
		return nil, nil, err
	}
	if plan == "freemium" && dbPipelineRelease.UID != latestReleaseUID {
		return nil, nil, ErrCanNotTriggerNonLatestPipelineRelease
	}

	return s.triggerPipeline(ctx, ns, authUser, dbPipelineRelease.Recipe, isPublic, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerId, returnTraces)
}

func (s *service) TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUid uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerId string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.String()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUid, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUid, id, false)
	if err != nil {
		return nil, err
	}

	isPublic := false
	if isPublic, err = s.aclClient.CheckPublicExecutable("pipeline", dbPipeline.UID); err != nil {
		return nil, err
	}

	plan := ""
	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			plan = resp.Subscription.Plan
		}
	} else {
		resp, err := s.mgmtPublicServiceClient.GetUserSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetUserSubscriptionRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
		)
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			plan = resp.Subscription.Plan
		}
	}

	latestReleaseUID, err := s.GetNamespacePipelineLatestReleaseUid(ctx, ns, authUser, dbPipeline.ID)
	if err != nil {
		return nil, err
	}
	if plan == "freemium" && dbPipelineRelease.UID != latestReleaseUID {
		return nil, ErrCanNotTriggerNonLatestPipelineRelease
	}

	return s.triggerAsyncPipeline(ctx, ns, authUser, dbPipelineRelease.Recipe, isPublic, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerId, returnTraces)
}

func (s *service) RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.RemoveCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.KeepCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) ListConnectorDefinitions(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter) ([]*pipelinePB.ConnectorDefinition, int32, string, error) {

	var err error
	prevLastUid := ""

	if pageToken != "" {
		_, prevLastUid, err = paginate.DecodeToken(pageToken)
		if err != nil {

			return nil, 0, "", repository.ErrPageTokenDecode
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
	return pageDefs, int32(len(defs)), nextPageToken, err

}

func (s *service) GetConnectorByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View, credentialMask bool) (*pipelinePB.Connector, error) {

	if granted, err := s.aclClient.CheckPermission("connector", uid, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	userPermalink := resource.UserUidToUserPermalink(authUser.UID)
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

func (s *service) ListConnectors(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error) {

	userPermalink := resource.UserUidToUserPermalink(authUser.UID)

	uidAllowList, err := s.aclClient.ListPermissions("connector", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListConnectors(ctx, userPermalink, int64(pageSize), pageToken, view == VIEW_BASIC, filter, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, int32(totalSize), nextPageToken, err

}

func (s *service) CreateNamespaceConnector(ctx context.Context, ns resource.Namespace, authUser *AuthUser, connector *pipelinePB.Connector) (*pipelinePB.Connector, error) {

	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPublicServiceClient.GetOrganizationSubscription(
			metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)),
			&mgmtPB.GetOrganizationSubscriptionRequest{Parent: fmt.Sprintf("organizations/%s", ns.NsID)})
		if err != nil {
			s, ok := status.FromError(err)
			if !ok {
				return nil, err
			}
			if s.Code() != codes.Unimplemented {
				return nil, err
			}
		} else {
			if resp.Subscription.Plan == "inactive" {
				return nil, status.Errorf(codes.FailedPrecondition, "the organization subscription is not active")
			}
		}

	}

	ownerPermalink := ns.String()

	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		if granted, err := s.aclClient.CheckPermission("organization", ns.NsUid, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "member"); err != nil {
			return nil, err
		} else if !granted {
			return nil, ErrNoPermission
		}
	} else {
		if ns.NsUid != authUser.UID {
			return nil, ErrNoPermission
		}
	}

	connDefResp, err := s.connector.GetConnectorDefinitionByID(strings.Split(connector.ConnectorDefinitionName, "/")[1])
	if err != nil {
		return nil, err
	}

	connDefUID, err := uuid.FromString(connDefResp.GetUid())
	if err != nil {
		return nil, err
	}

	connConfig, err := connector.GetConfiguration().MarshalJSON()
	if err != nil {

		return nil, err
	}

	connDesc := sql.NullString{
		String: connector.GetDescription(),
		Valid:  len(connector.GetDescription()) > 0,
	}

	dbConnectorToCreate := &datamodel.Connector{
		ID:                     connector.Id,
		Owner:                  ns.String(),
		ConnectorDefinitionUID: connDefUID,
		Tombstone:              false,
		Configuration:          connConfig,
		ConnectorType:          datamodel.ConnectorType(connDefResp.GetType()),
		Description:            connDesc,
		Visibility:             datamodel.ConnectorVisibility(connector.Visibility),
	}

	if existingConnector, _ := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, dbConnectorToCreate.ID, true); existingConnector != nil {
		return nil, err
	}

	if err := s.repository.CreateNamespaceConnector(ctx, ownerPermalink, dbConnectorToCreate); err != nil {
		return nil, err
	}

	// User desire state = DISCONNECTED
	if err := s.repository.UpdateNamespaceConnectorStateByID(ctx, ownerPermalink, dbConnectorToCreate.ID, datamodel.ConnectorState(pipelinePB.Connector_STATE_DISCONNECTED)); err != nil {
		return nil, err
	}
	if err := s.UpdateConnectorState(dbConnectorToCreate.UID, pipelinePB.Connector_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, dbConnectorToCreate.ID, false)
	if err != nil {
		return nil, err
	}
	ownerType := string(ns.NsType)[0 : len(string(ns.NsType))-1]
	ownerUID := ns.NsUid
	err = s.aclClient.SetOwner("connector", dbConnector.UID, ownerType, ownerUID)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)

}

func (s *service) ListNamespaceConnectors(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error) {

	uidAllowList, err := s.aclClient.ListPermissions("connector", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	ownerPermalink := ns.String()

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListNamespaceConnectors(ctx, ownerPermalink, int64(pageSize), pageToken, view == VIEW_BASIC, filter, uidAllowList, showDeleted)

	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, int32(totalSize), nextPageToken, err

}

func (s *service) ListConnectorsAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error) {

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListConnectorsAdmin(ctx, int64(pageSize), pageToken, view == VIEW_BASIC, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, int32(totalSize), nextPageToken, err
}

func (s *service) GetNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View, credentialMask bool) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, view == VIEW_BASIC)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("connector", dbConnector.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
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

func (s *service) UpdateNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, connector *pipelinePB.Connector) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()

	dbConnectorToUpdate, err := s.convertProtoToDatamodel(ctx, connector)
	if err != nil {
		return nil, err
	}
	if granted, err := s.aclClient.CheckPermission("connector", dbConnectorToUpdate.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}
	dbConnectorToUpdate.Owner = ownerPermalink

	if err := s.repository.UpdateNamespaceConnectorByID(ctx, ownerPermalink, id, dbConnectorToUpdate); err != nil {
		return nil, err
	}

	// Check connector state
	if err := s.UpdateConnectorState(dbConnectorToUpdate.UID, pipelinePB.Connector_STATE_DISCONNECTED, nil); err != nil {
		return nil, err
	}

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, dbConnectorToUpdate.ID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)

}

func (s *service) DeleteNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error {
	// logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.String()

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("connector", dbConnector.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
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

	err = s.aclClient.Purge("connector", dbConnector.UID)
	if err != nil {
		return err
	}

	return s.repository.DeleteNamespaceConnectorByID(ctx, ownerPermalink, id)
}

func (s *service) UpdateNamespaceConnectorStateByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, state pipelinePB.Connector_State) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()

	// Validation: trigger and response connector cannot be disconnected
	conn, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("connector", conn.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
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
		if err := s.repository.UpdateNamespaceConnectorStateByID(ctx, ownerPermalink, id, datamodel.ConnectorState(pipelinePB.Connector_STATE_CONNECTED)); err != nil {
			return nil, err
		}

		if err := s.UpdateConnectorState(conn.UID, pipelinePB.Connector_STATE_CONNECTED, nil); err != nil {
			return nil, err
		}

	case pipelinePB.Connector_STATE_DISCONNECTED:

		if err := s.repository.UpdateNamespaceConnectorStateByID(ctx, ownerPermalink, id, datamodel.ConnectorState(pipelinePB.Connector_STATE_DISCONNECTED)); err != nil {
			return nil, err
		}
		if err := s.UpdateConnectorState(conn.UID, pipelinePB.Connector_State(state), nil); err != nil {
			return nil, err
		}
	}

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)
}

func (s *service) UpdateNamespaceConnectorIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.String()

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("connector", dbConnector.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if err := s.repository.UpdateNamespaceConnectorIDByID(ctx, ownerPermalink, id, newID); err != nil {
		return nil, err
	}

	dbConnector, err = s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, newID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, VIEW_FULL, true)

}

func (s *service) Execute(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, task string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	logger, _ := logger.GetZapLogger(ctx)
	ownerPermalink := ns.String()

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("connector", dbConnector.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
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
