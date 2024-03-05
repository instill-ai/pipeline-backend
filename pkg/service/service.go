package service

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"
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
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	workflowpb "go.temporal.io/api/workflow/v1"
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
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the service, we'd better use uid as our function params

// Service interface
type Service interface {
	GetOperatorDefinitionByID(ctx context.Context, defID string) (*pipelinePB.OperatorDefinition, error)
	ListOperatorDefinitions(context.Context, *pipelinePB.ListOperatorDefinitionsRequest) (*pipelinePB.ListOperatorDefinitionsResponse, error)

	ListPipelines(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetPipelineByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)
	CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	ListNamespacePipelines(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View) (*pipelinePB.Pipeline, error)
	UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, updatedPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Pipeline, error)
	DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error
	ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (*pipelinePB.Pipeline, error)
	GetNamespacePipelineDefaultReleaseUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error)
	GetNamespacePipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error)
	CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, targetNS resource.Namespace, targetID string) (*pipelinePB.Pipeline, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)

	CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error)
	GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error)
	GetNamespacePipelineReleaseByUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.PipelineRelease, error)
	UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, updatedPipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string) error
	RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string) error
	SetDefaultNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string) error
	UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error)

	ListPipelineReleasesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error)

	// Influx API

	TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)

	TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)

	WriteNewPipelineDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error

	GetRscNamespaceAndNameID(path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(path string) (resource.Namespace, uuid.UUID, error)
	GetRscNamespaceAndNameIDAndReleaseID(path string) (resource.Namespace, string, string, error)
	ConvertOwnerPermalinkToName(permalink string) (string, error)
	ConvertOwnerNameToPermalink(name string) (string, error)
	ConvertReleaseIDAlias(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineID string, releaseID string) (string, error)

	PBToDBPipeline(ctx context.Context, ns resource.Namespace, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, authUser *AuthUser, view View) (*pipelinePB.Pipeline, error)
	DBToPBPipelines(ctx context.Context, dbPipeline []*datamodel.Pipeline, authUser *AuthUser, view View) ([]*pipelinePB.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, dbPipelineRelease *datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, dbPipelineRelease []*datamodel.PipelineRelease, view View, latestUUID uuid.UUID, defaultUUID uuid.UUID) ([]*pipelinePB.PipelineRelease, error)

	AuthenticateUser(ctx context.Context, allowVisitor bool) (authUser *AuthUser, err error)

	ListComponentDefinitions(context.Context, *pipelinePB.ListComponentDefinitionsRequest) (*pipelinePB.ListComponentDefinitionsResponse, error)

	ListConnectorDefinitions(context.Context, *pipelinePB.ListConnectorDefinitionsRequest) (*pipelinePB.ListConnectorDefinitionsResponse, error)
	GetConnectorByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View, credentialMask bool) (*pipelinePB.Connector, error)
	GetConnectorDefinitionByID(ctx context.Context, id string, view View) (*pipelinePB.ConnectorDefinition, error)

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
	// Verify if "Instill-User-Uid" is in the header
	authType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	if authType == "user" {
		headerCtxUserUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
		if headerCtxUserUID == "" {
			return nil, ErrUnauthenticated
		}
		return &AuthUser{
			UID:       uuid.FromStringOrNil(headerCtxUserUID),
			IsVisitor: false,
		}, nil
	} else {
		if !allowVisitor {
			return nil, ErrUnauthenticated
		}
		headerCtxVisitorUID := resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
		if headerCtxVisitorUID == "" {
			return nil, ErrUnauthenticated
		}

		return &AuthUser{
			UID:       uuid.FromStringOrNil(headerCtxVisitorUID),
			IsVisitor: true,
		}, nil
	}

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

func (s *service) FetchOwnerWithPermalink(permalink string) (*mgmtPB.Owner, error) {
	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(context.Background(), &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("FetchOwnerWithPermalink error")
		}

		return &mgmtPB.Owner{Owner: &mgmtPB.Owner_User{User: resp.User}}, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(context.Background(), &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("FetchOwnerWithPermalink error")
		}
		return &mgmtPB.Owner{Owner: &mgmtPB.Owner_Organization{Organization: resp.Organization}}, nil

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
			NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, "", nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
		NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
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
			NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
		}, uuid.Nil, nil
	}
	return resource.Namespace{
		NsType: resource.NamespaceType(splits[0]),
		NsID:   splits[1],
		NsUID:  uuid.FromStringOrNil(strings.Split(uidStr, "/")[1]),
	}, uuid.FromStringOrNil(splits[3]), nil
}

func (s *service) GetRscNamespaceAndNameIDAndReleaseID(path string) (resource.Namespace, string, string, error) {
	ns, pipelineID, err := s.GetRscNamespaceAndNameID(path)
	if err != nil {
		return ns, pipelineID, "", err
	}
	splits := strings.Split(path, "/")

	if len(splits) < 6 {
		return ns, pipelineID, "", fmt.Errorf("path error")
	}
	return ns, pipelineID, splits[5], err
}

func (s *service) ConvertReleaseIDAlias(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineID string, releaseID string) (string, error) {
	ownerPermalink := ns.Permalink()

	// TODO: simplify these
	if releaseID == "default" {
		releaseUID, err := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipelineID)
		if err != nil {
			return "", err
		}
		dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true)
		if err != nil {
			return "", err
		}
		dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByUID(ctx, ownerPermalink, dbPipeline.UID, releaseUID, true)
		if err != nil {
			return "", err
		}
		return dbPipelineRelease.ID, nil
	} else if releaseID == "latest" {
		releaseUID, err := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipelineID)
		if err != nil {
			return "", err
		}
		dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true)
		if err != nil {
			return "", err
		}
		dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByUID(ctx, ownerPermalink, dbPipeline.UID, releaseUID, true)
		if err != nil {
			return "", err
		}
		return dbPipelineRelease.ID, nil
	}
	return releaseID, nil

}

func (s *service) GetOperatorDefinitionByID(ctx context.Context, defID string) (*pipelinePB.OperatorDefinition, error) {
	return s.operator.GetOperatorDefinitionByID(defID, nil)
}

func (s *service) operatorDefinitions() []*pipelinePB.OperatorDefinition {
	allDefs := s.operator.ListOperatorDefinitions()

	// don't return definition with tombstone = true
	withoutTombstone := make([]*pipelinePB.OperatorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if !def.Tombstone {
			withoutTombstone = append(withoutTombstone, def)
		}
	}

	return withoutTombstone
}

func (s *service) ListOperatorDefinitions(ctx context.Context, req *pipelinePB.ListOperatorDefinitionsRequest) (*pipelinePB.ListOperatorDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	prevLastUID, err := s.lastUIDFromToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	defs := s.operatorDefinitions()

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}
	page := make([]*pipelinePB.OperatorDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.OperatorDefinition)
		page = append(page, def)
		lastUID = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUID)
	}

	view := parseView(int32(req.GetView()))
	for _, def := range page {
		s.applyViewToOperatorDefinition(def, view)
	}

	resp := &pipelinePB.ListOperatorDefinitionsResponse{
		NextPageToken:       nextPageToken,
		TotalSize:           int32(len(page)),
		OperatorDefinitions: page,
	}

	return resp, nil
}

func (s *service) ListPipelines(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	var uidAllowList []uuid.UUID
	var err error

	// TODO: optimize the logic
	if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), uuid.Nil, "reader")
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), authUser.UID, "reader")
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), uuid.Nil, "reader")
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), authUser.UID, "reader")
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, int64(pageSize), pageToken, view == ViewBasic, filter, uidAllowList, showDeleted)
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

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, uid, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, view)
}

func (s *service) CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pbPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission("organization", ns.NsUID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "member")
		if err != nil {
			return nil, err
		}
		if !granted {
			return nil, ErrNoPermission
		}
	} else {
		if ns.NsUID != authUser.UID {
			return nil, ErrNoPermission
		}
	}

	dbPipeline, err := s.PBToDBPipeline(ctx, ns, pbPipeline)
	if err != nil {
		return nil, err
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
	ownerUID := ns.NsUID
	err = s.aclClient.SetOwner("pipeline", dbCreatedPipeline.UID, ownerType, ownerUID)
	if err != nil {
		return nil, err
	}
	// TODO: use OpenFGA as single source of truth
	err = s.aclClient.SetPipelinePermissionMap(dbCreatedPipeline)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbCreatedPipeline, authUser, ViewFull)
}

func (s *service) ListNamespacePipelines(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	ownerPermalink := ns.Permalink()

	var uidAllowList []uuid.UUID
	var err error

	// TODO: optimize the logic
	if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), uuid.Nil, "reader")
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), authUser.UID, "reader")
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), uuid.Nil, "reader")
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions("pipeline", authUser.GetACLType(), authUser.UID, "reader")
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbPipelines, ps, pt, err := s.repository.ListNamespacePipelines(ctx, ownerPermalink, int64(pageSize), pageToken, view == ViewBasic, filter, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, authUser, view)
	return pbPipelines, int32(ps), pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, int64(pageSize), pageToken, view == ViewBasic, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, nil, view)
	return pbPipelines, int32(ps), pt, err

}

func (s *service) GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, view == ViewBasic)
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

func (s *service) GetNamespacePipelineDefaultReleaseUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true)
	if err != nil {
		return uuid.Nil, err
	}

	return dbPipeline.DefaultReleaseUID, nil
}

func (s *service) GetNamespacePipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (uuid.UUID, error) {

	ownerPermalink := ns.Permalink()

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

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, uid, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, nil, view)

}

func (s *service) UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, toUpdPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipelineToUpdate, err := s.PBToDBPipeline(ctx, ns, toUpdPipeline)
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

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, ViewFull)
}

func (s *service) DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error {
	ownerPermalink := ns.Permalink()

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

func (s *service) CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, targetNS resource.Namespace, targetID string) (*pipelinePB.Pipeline, error) {
	sourcePipeline, err := s.GetNamespacePipelineByID(ctx, ns, authUser, id, ViewRecipe)
	if err != nil {
		return nil, err
	}
	for idx := range sourcePipeline.Recipe.Components {
		switch sourcePipeline.Recipe.Components[idx].Component.(type) {
		case *pipelinePB.Component_ConnectorComponent:
			sourcePipeline.Recipe.Components[idx].GetConnectorComponent().ConnectorName = ""
		}

	}
	sourcePipeline.Id = targetID
	targetPipeline, err := s.CreateNamespacePipeline(ctx, targetNS, authUser, sourcePipeline)
	if err != nil {
		return nil, err
	}
	return targetPipeline, nil
}

func (s *service) ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

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

	recipeErr := s.checkRecipe(ownerPermalink, dbPipeline.Recipe)

	if recipeErr != nil {
		return nil, recipeErr
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, ViewFull)

}

func (s *service) UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

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

	return s.DBToPBPipeline(ctx, dbPipeline, authUser, ViewFull)
}

func (s *service) preTriggerPipeline(ctx context.Context, isAdmin bool, ns resource.Namespace, authUser *AuthUser, recipe *datamodel.Recipe, pipelineInputs []*structpb.Struct) error {

	batchSize := len(pipelineInputs)
	if batchSize > constant.MaxBatchSize {
		return ErrExceedMaxBatchSize
	}

	checkRateLimited := !isAdmin

	if !checkRateLimited {
		if ns.NsType == resource.Organization {
			resp, err := s.mgmtPrivateServiceClient.GetOrganizationSubscriptionAdmin(
				ctx,
				&mgmtPB.GetOrganizationSubscriptionAdminRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
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
				if resp.Subscription.Plan == mgmtPB.OrganizationSubscription_PLAN_FREEMIUM {
					checkRateLimited = true
				}
			}

		} else {
			resp, err := s.mgmtPrivateServiceClient.GetUserSubscriptionAdmin(
				ctx,
				&mgmtPB.GetUserSubscriptionAdminRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
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
				if resp.Subscription.Plan == mgmtPB.UserSubscription_PLAN_FREEMIUM {
					checkRateLimited = true
				}
			}
		}
	}

	if checkRateLimited {
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
	}

	var metadata []byte

	instillFormatMap := map[string]string{}
	for _, comp := range recipe.Components {
		// op start
		if comp.IsStartComponent() {

			schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
			schStruct.Fields["type"] = structpb.NewStringValue("object")
			b, _ := json.Marshal(comp.StartComponent.Fields)
			properties := &structpb.Struct{}
			_ = protojson.Unmarshal(b, properties)
			schStruct.Fields["properties"] = structpb.NewStructValue(properties)
			for k, v := range comp.StartComponent.Fields {
				instillFormatMap[k] = v.InstillFormat
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
		var i any
		if err := json.Unmarshal(b, &i); err != nil {
			errors = append(errors, fmt.Sprintf("inputs[%d]: data error", idx))
			continue
		}

		m := i.(map[string]any)

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
			case []string:
				if instillFormatMap[k] != "array:string" {
					for idx := range s {
						if !strings.HasPrefix(s[idx], "data:") {
							b, err := base64.StdEncoding.DecodeString(s[idx])
							if err != nil {
								return fmt.Errorf("can not decode file %s, %s", instillFormatMap[k], s)
							}
							mimeType := strings.Split(mimetype.Detect(b).String(), ";")[0]
							pipelineInput.Fields[k].GetListValue().GetValues()[idx] = structpb.NewStringValue(fmt.Sprintf("data:%s;base64,%s", mimeType, s[idx]))
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

func (s *service) GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error) {
	workflowExecutionRes, err := s.temporalClient.DescribeWorkflowExecution(ctx, workflowID, "")

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

func (s *service) CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewFull)
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

	dbPipelineReleaseToCreate, err := s.PBToDBPipelineRelease(ctx, pipelineUID, pipelineRelease)
	if err != nil {
		return nil, err
	}

	if err := s.repository.CreateNamespacePipelineRelease(ctx, ownerPermalink, pipelineUID, dbPipelineReleaseToCreate); err != nil {
		return nil, err
	}

	dbCreatedPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, pipelineRelease.Id, false)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbCreatedPipelineRelease, ViewFull, latestUUID, defaultUUID)

}
func (s *service) ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error) {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, 0, "", err
	} else if !granted {
		return nil, 0, "", ErrNotFound
	}

	dbPipelineReleases, ps, pt, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, pipelineUID, int64(pageSize), pageToken, view == ViewBasic, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipeline.Id)

	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, latestUUID, defaultUUID)
	return pbPipelineReleases, int32(ps), pt, err
}

func (s *service) ListPipelineReleasesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error) {

	dbPipelineReleases, ps, pt, err := s.repository.ListPipelineReleasesAdmin(ctx, int64(pageSize), pageToken, view == ViewBasic, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipelineReleases, view, uuid.Nil, uuid.Nil)
	return pbPipelineReleases, int32(ps), pt, err

}

func (s *service) GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}
func (s *service) GetNamespacePipelineReleaseByUID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, uid uuid.UUID, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission("pipeline", uuid.FromStringOrNil(pipeline.GetUid()), authUser.GetACLType(), authUser.UID, s.getCode(ctx), "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByUID(ctx, ownerPermalink, pipelineUID, uid, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, view, latestUUID, defaultUUID)

}

func (s *service) UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, toUpdPipeline *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
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

	if _, err := s.GetNamespacePipelineReleaseByID(ctx, ns, authUser, pipelineUID, id, ViewBasic); err != nil {
		return nil, err
	}

	pbPipelineReleaseToUpdate, err := s.PBToDBPipelineRelease(ctx, pipelineUID, toUpdPipeline)
	if err != nil {
		return nil, err
	}
	if err := s.repository.UpdateNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, pbPipelineReleaseToUpdate); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, toUpdPipeline.Id, false)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, ViewFull, latestUUID, defaultUUID)
}

func (s *service) UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
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
	_, err = s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, true)
	if err != nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespacePipelineReleaseIDByID(ctx, ownerPermalink, pipelineUID, id, newID); err != nil {
		return nil, err
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, newID, false)
	if err != nil {
		return nil, err
	}

	latestUUID, _ := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, pipeline.Id)
	defaultUUID, _ := s.GetNamespacePipelineDefaultReleaseUID(ctx, ns, authUser, pipeline.Id)

	return s.DBToPBPipelineRelease(ctx, dbPipelineRelease, ViewFull, latestUUID, defaultUUID)
}

func (s *service) DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string) error {
	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
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

	return s.repository.DeleteNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id)
}

func (s *service) RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string) error {
	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
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

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, err = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUID, false); err != nil {
		return err
	}
	existingPipeline.Recipe = dbPipelineRelease.Recipe

	if err := s.repository.UpdateNamespacePipelineByID(ctx, ownerPermalink, existingPipeline.ID, existingPipeline); err != nil {
		return err
	}

	return nil
}

func (s *service) SetDefaultNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string) error {

	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, authUser, pipelineUID, ViewBasic)
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

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return err
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, err = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUID, false); err != nil {
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
	isAdmin bool,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerID string,
	returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	logger, _ := logger.GetZapLogger(ctx)

	err := s.preTriggerPipeline(ctx, isAdmin, ns, authUser, recipe, pipelineInputs)
	if err != nil {
		return nil, nil, err
	}

	inputBlobRedisKeys := []string{}
	for idx, input := range pipelineInputs {
		inputJSON, err := protojson.Marshal(input)
		if err != nil {
			return nil, nil, err
		}

		inputBlobRedisKey := fmt.Sprintf("async_pipeline_request:%s:%d", pipelineTriggerID, idx)
		s.redisClient.Set(
			context.Background(),
			inputBlobRedisKey,
			inputJSON,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
		defer s.redisClient.Del(context.Background(), inputBlobRedisKey)
	}
	memo := map[string]any{}
	memo["number_of_data"] = len(inputBlobRedisKeys)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerID,
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
			PipelineID:                 pipelineID,
			PipelineUID:                pipelineUID,
			PipelineReleaseID:          pipelineReleaseID,
			PipelineReleaseUID:         pipelineReleaseUID,
			PipelineRecipe:             recipe,
			OwnerPermalink:             ns.Permalink(),
			UserPermalink:              authUser.Permalink(),
			ReturnTraces:               returnTraces,
			Mode:                       mgmtPB.Mode_MODE_SYNC,
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
	isAdmin bool,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerID string,
	returnTraces bool) (*longrunningpb.Operation, error) {

	err := s.preTriggerPipeline(ctx, isAdmin, ns, authUser, recipe, pipelineInputs)
	if err != nil {
		return nil, err
	}
	logger, _ := logger.GetZapLogger(ctx)

	inputBlobRedisKeys := []string{}
	for idx, input := range pipelineInputs {
		inputJSON, err := protojson.Marshal(input)
		if err != nil {
			return nil, err
		}

		inputBlobRedisKey := fmt.Sprintf("async_pipeline_request:%s:%d", pipelineTriggerID, idx)
		s.redisClient.Set(
			context.Background(),
			inputBlobRedisKey,
			inputJSON,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
	}
	memo := map[string]any{}
	memo["number_of_data"] = len(inputBlobRedisKeys)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       pipelineTriggerID,
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
			PipelineID:                 pipelineID,
			PipelineUID:                pipelineUID,
			PipelineReleaseID:          pipelineReleaseID,
			PipelineReleaseUID:         pipelineReleaseUID,
			PipelineRecipe:             recipe,
			OwnerPermalink:             ns.Permalink(),
			UserPermalink:              authUser.Permalink(),
			ReturnTraces:               returnTraces,
			Mode:                       mgmtPB.Mode_MODE_ASYNC,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, err
	}

	logger.Info(fmt.Sprintf("started workflow with workflowID %s and RunID %s", we.GetID(), we.GetRunID()))

	return &longrunningpb.Operation{
		Name: fmt.Sprintf("operations/%s", pipelineTriggerID),
		Done: false,
	}, nil

}

func (s *service) TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

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

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, nil, err
	}

	return s.triggerPipeline(ctx, ns, authUser, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerID, returnTraces)

}

func (s *service) TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

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

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	}

	return s.triggerAsyncPipeline(ctx, ns, authUser, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerID, returnTraces)

}

func (s *service) TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false)
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

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, nil, err
	}

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, nil, err
	}

	latestReleaseUID, err := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, dbPipeline.ID)
	if err != nil {
		return nil, nil, err
	}

	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPrivateServiceClient.GetOrganizationSubscriptionAdmin(
			ctx,
			&mgmtPB.GetOrganizationSubscriptionAdminRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
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
			if resp.Subscription.Plan == mgmtPB.OrganizationSubscription_PLAN_FREEMIUM && dbPipelineRelease.UID != latestReleaseUID {
				return nil, nil, ErrCanNotTriggerNonLatestPipelineRelease
			}
		}
	} else {
		resp, err := s.mgmtPrivateServiceClient.GetUserSubscriptionAdmin(
			ctx,
			&mgmtPB.GetUserSubscriptionAdminRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
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
			if resp.Subscription.Plan == mgmtPB.UserSubscription_PLAN_FREEMIUM && dbPipelineRelease.UID != latestReleaseUID {
				return nil, nil, ErrCanNotTriggerNonLatestPipelineRelease
			}
		}
	}

	return s.triggerPipeline(ctx, ns, authUser, dbPipelineRelease.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerID, returnTraces)
}

func (s *service) TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pipelineUID uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false)
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

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, err
	}

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission("pipeline", dbPipeline.UID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	}

	latestReleaseUID, err := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, authUser, dbPipeline.ID)
	if err != nil {
		return nil, err
	}

	if ns.NsType == resource.Organization {
		resp, err := s.mgmtPrivateServiceClient.GetOrganizationSubscriptionAdmin(
			ctx,
			&mgmtPB.GetOrganizationSubscriptionAdminRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
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
			if resp.Subscription.Plan == mgmtPB.OrganizationSubscription_PLAN_FREEMIUM && dbPipelineRelease.UID != latestReleaseUID {
				return nil, ErrCanNotTriggerNonLatestPipelineRelease
			}
		}
	} else {
		resp, err := s.mgmtPrivateServiceClient.GetUserSubscriptionAdmin(
			ctx,
			&mgmtPB.GetUserSubscriptionAdminRequest{Parent: fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)},
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
			if resp.Subscription.Plan == mgmtPB.UserSubscription_PLAN_FREEMIUM && dbPipelineRelease.UID != latestReleaseUID {
				return nil, ErrCanNotTriggerNonLatestPipelineRelease
			}
		}
	}

	return s.triggerAsyncPipeline(ctx, ns, authUser, dbPipelineRelease.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerID, returnTraces)
}

func (s *service) RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.RemoveCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.KeepCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) connectorDefinitions() []*pipelinePB.ConnectorDefinition {
	allDefs := s.connector.ListConnectorDefinitions()

	// don't return definition with tombstone = true
	withoutTombstone := make([]*pipelinePB.ConnectorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if !def.Tombstone {
			withoutTombstone = append(withoutTombstone, def)
		}
	}

	return withoutTombstone
}

func (s *service) filterConnectorDefinitions(defs []*pipelinePB.ConnectorDefinition, filter filtering.Filter) []*pipelinePB.ConnectorDefinition {
	if filter.CheckedExpr == nil {
		return defs
	}

	filtered := make([]*pipelinePB.ConnectorDefinition, 0, len(defs))
	trans := repository.NewTranspiler(filter)
	expr, _ := trans.Transpile()
	typeMap := map[string]bool{}
	for i, v := range expr.Vars {
		if i == 0 {
			typeMap[string(v.(protoreflect.Name))] = true
			continue
		}

		typeMap[string(v.([]any)[0].(protoreflect.Name))] = true
	}

	for _, def := range defs {
		if _, ok := typeMap[def.Type.String()]; ok {
			filtered = append(filtered, def)
		}
	}

	return filtered
}

func (s *service) lastUIDFromToken(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	_, id, err := paginate.DecodeToken(token)
	if err != nil {
		return "", repository.ErrPageTokenDecode
	}

	return id, nil
}

func (s *service) pageSizeInRange(pageSize int32) int {
	if pageSize <= 0 {
		return repository.DefaultPageSize
	}

	if pageSize > repository.MaxPageSize {
		return repository.MaxPageSize
	}

	return int(pageSize)
}

func (s *service) offsetInRange(offset int32) int {
	if offset <= 0 {
		return 0
	}

	return int(offset)
}

func (s *service) applyViewToConnectorDefinition(cd *pipelinePB.ConnectorDefinition, v View) {
	cd.VendorAttributes = nil
	if v == ViewBasic {
		cd.Spec = nil
	}
}

func (s *service) applyViewToOperatorDefinition(od *pipelinePB.OperatorDefinition, v View) {
	od.Name = fmt.Sprintf("operator-definitions/%s", od.Id)
	if v == ViewBasic {
		od.Spec = nil
	}
}

// ListComponentDefinitions returns a paginated list of components.
func (s *service) ListComponentDefinitions(ctx context.Context, req *pipelinePB.ListComponentDefinitionsRequest) (*pipelinePB.ListComponentDefinitionsResponse, error) {
	view := parseView(int32(req.GetView()))
	pageSize := s.pageSizeInRange(req.GetPageSize())
	offset := s.offsetInRange(req.GetPage())
	startIdx := pageSize * offset

	connDefs := s.connectorDefinitions()
	opDefs := s.operatorDefinitions()

	// Build a list with all the component definitions.
	compDefs := make([]*pipelinePB.ComponentDefinition, 0, len(connDefs)+len(opDefs))
	for _, cd := range connDefs {
		compDefs = append(compDefs, &pipelinePB.ComponentDefinition{
			Type:       connectorTypeToComponentType[cd.Type],
			Definition: &pipelinePB.ComponentDefinition_ConnectorDefinition{ConnectorDefinition: cd},
		})
	}
	for _, od := range opDefs {
		compDefs = append(compDefs, &pipelinePB.ComponentDefinition{
			Type:       pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR,
			Definition: &pipelinePB.ComponentDefinition_OperatorDefinition{OperatorDefinition: od},
		})
	}

	// Extract a page from the list and compute view.
	totalSize := len(compDefs)
	compPage := make([]*pipelinePB.ComponentDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < totalSize; i++ {
		d := proto.Clone(compDefs[startIdx+i]).(*pipelinePB.ComponentDefinition)
		if cd := d.GetConnectorDefinition(); cd != nil {
			s.applyViewToConnectorDefinition(cd, view)
		} else if od := d.GetOperatorDefinition(); od != nil {
			s.applyViewToOperatorDefinition(od, view)
		}

		compPage = append(compPage, d)
	}

	resp := &pipelinePB.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(offset),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: compPage,
	}

	return resp, nil
}

func (s *service) ListConnectorDefinitions(ctx context.Context, req *pipelinePB.ListConnectorDefinitionsRequest) (*pipelinePB.ListConnectorDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	prevLastUID, err := s.lastUIDFromToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	var connType pipelinePB.ConnectorType
	declarations, err := filtering.NewDeclarations([]filtering.DeclarationOption{
		filtering.DeclareStandardFunctions(),
		filtering.DeclareEnumIdent("connector_type", connType.Type()),
	}...)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	defs := s.filterConnectorDefinitions(s.connectorDefinitions(), filter)

	startIdx := 0
	lastUID := ""
	for idx, def := range defs {
		if def.Uid == prevLastUID {
			startIdx = idx + 1
			break
		}
	}

	page := make([]*pipelinePB.ConnectorDefinition, 0, pageSize)
	for i := 0; i < pageSize && startIdx+i < len(defs); i++ {
		def := proto.Clone(defs[startIdx+i]).(*pipelinePB.ConnectorDefinition)
		page = append(page, def)
		lastUID = def.Uid
	}

	nextPageToken := ""

	if startIdx+len(page) < len(defs) {
		nextPageToken = paginate.EncodeToken(time.Time{}, lastUID)
	}

	view := parseView(int32(req.GetView()))
	pageDefs := make([]*pipelinePB.ConnectorDefinition, 0, len(page))
	for _, def := range page {
		s.applyViewToConnectorDefinition(def, view)
		pageDefs = append(pageDefs, def)
	}

	return &pipelinePB.ListConnectorDefinitionsResponse{
		ConnectorDefinitions: pageDefs,
		NextPageToken:        nextPageToken,
		TotalSize:            int32(len(defs)),
	}, nil
}

func (s *service) GetConnectorByUID(ctx context.Context, authUser *AuthUser, uid uuid.UUID, view View, credentialMask bool) (*pipelinePB.Connector, error) {

	if granted, err := s.aclClient.CheckPermission("connector", uid, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	userPermalink := resource.UserUIDToUserPermalink(authUser.UID)
	dbConnector, err := s.repository.GetConnectorByUID(ctx, userPermalink, uid, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, view, credentialMask)
}

func (s *service) GetConnectorDefinitionByID(ctx context.Context, id string, view View) (*pipelinePB.ConnectorDefinition, error) {

	def, err := s.connector.GetConnectorDefinitionByID(id, nil, nil)
	if err != nil {
		return nil, err
	}
	def = proto.Clone(def).(*pipelinePB.ConnectorDefinition)
	if view == ViewBasic {
		def.Spec = nil
	}
	def.VendorAttributes = nil

	return def, nil
}

func (s *service) ListConnectors(ctx context.Context, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error) {

	userPermalink := resource.UserUIDToUserPermalink(authUser.UID)

	uidAllowList, err := s.aclClient.ListPermissions("connector", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListConnectors(ctx, userPermalink, int64(pageSize), pageToken, view == ViewBasic, filter, uidAllowList, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, int32(totalSize), nextPageToken, err

}

func (s *service) CreateNamespaceConnector(ctx context.Context, ns resource.Namespace, authUser *AuthUser, connector *pipelinePB.Connector) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.Permalink()

	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		if granted, err := s.aclClient.CheckPermission("organization", ns.NsUID, authUser.GetACLType(), authUser.UID, s.getCode(ctx), "member"); err != nil {
			return nil, err
		} else if !granted {
			return nil, ErrNoPermission
		}
	} else {
		if ns.NsUID != authUser.UID {
			return nil, ErrNoPermission
		}
	}

	connDefResp, err := s.connector.GetConnectorDefinitionByID(strings.Split(connector.ConnectorDefinitionName, "/")[1], nil, nil)
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
		Owner:                  ns.Permalink(),
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

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, dbConnectorToCreate.ID, false)
	if err != nil {
		return nil, err
	}
	ownerType := string(ns.NsType)[0 : len(string(ns.NsType))-1]
	ownerUID := ns.NsUID
	err = s.aclClient.SetOwner("connector", dbConnector.UID, ownerType, ownerUID)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, ViewFull, true)

}

func (s *service) ListNamespaceConnectors(ctx context.Context, ns resource.Namespace, authUser *AuthUser, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error) {

	uidAllowList, err := s.aclClient.ListPermissions("connector", authUser.GetACLType(), authUser.UID, "reader")
	if err != nil {
		return nil, 0, "", err
	}

	ownerPermalink := ns.Permalink()

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListNamespaceConnectors(ctx, ownerPermalink, int64(pageSize), pageToken, view == ViewBasic, filter, uidAllowList, showDeleted)

	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, int32(totalSize), nextPageToken, err

}

func (s *service) ListConnectorsAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Connector, int32, string, error) {

	dbConnectors, totalSize, nextPageToken, err := s.repository.ListConnectorsAdmin(ctx, int64(pageSize), pageToken, view == ViewBasic, filter, showDeleted)
	if err != nil {
		return nil, 0, "", err
	}

	pbConnectors, err := s.convertDatamodelArrayToProtoArray(ctx, dbConnectors, view, true)
	return pbConnectors, int32(totalSize), nextPageToken, err
}

func (s *service) GetNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, view View, credentialMask bool) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.Permalink()

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, id, view == ViewBasic)
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

	dbConnector, err := s.repository.GetConnectorByUIDAdmin(ctx, uid, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, view, true)
}

func (s *service) UpdateNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, connector *pipelinePB.Connector) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.Permalink()

	dbConnectorToUpdate, err := s.convertProtoToDatamodel(ctx, ns, connector)
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

	dbConnector, err := s.repository.GetNamespaceConnectorByID(ctx, ownerPermalink, dbConnectorToUpdate.ID, false)
	if err != nil {
		return nil, err
	}

	return s.convertDatamodelToProto(ctx, dbConnector, ViewFull, true)

}

func (s *service) DeleteNamespaceConnectorByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string) error {
	// logger, _ := logger.GetZapLogger(ctx)

	ownerPermalink := ns.Permalink()

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

	err = s.aclClient.Purge("connector", dbConnector.UID)
	if err != nil {
		return err
	}

	return s.repository.DeleteNamespaceConnectorByID(ctx, ownerPermalink, id)
}

func (s *service) UpdateNamespaceConnectorStateByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, state pipelinePB.Connector_State) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.Permalink()

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

	case pipelinePB.Connector_STATE_DISCONNECTED:

		if err := s.repository.UpdateNamespaceConnectorStateByID(ctx, ownerPermalink, id, datamodel.ConnectorState(pipelinePB.Connector_STATE_DISCONNECTED)); err != nil {
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

	return s.convertDatamodelToProto(ctx, dbConnector, ViewFull, true)
}

func (s *service) UpdateNamespaceConnectorIDByID(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, newID string) (*pipelinePB.Connector, error) {

	ownerPermalink := ns.Permalink()

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

	return s.convertDatamodelToProto(ctx, dbConnector, ViewFull, true)

}

func (s *service) Execute(ctx context.Context, ns resource.Namespace, authUser *AuthUser, id string, task string, inputs []*structpb.Struct) ([]*structpb.Struct, error) {

	logger, _ := logger.GetZapLogger(ctx)
	ownerPermalink := ns.Permalink()

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
		return pipelinePB.Connector_STATE_CONNECTED.Enum(), nil
	case pipelinePB.Connector_STATE_ERROR:
		return pipelinePB.Connector_STATE_ERROR.Enum(), nil
	default:
		return pipelinePB.Connector_STATE_ERROR.Enum(), nil
	}

}
