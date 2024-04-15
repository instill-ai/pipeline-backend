package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
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

	component "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/component/pkg/connector"
	operator "github.com/instill-ai/component/pkg/operator"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// Service interface
type Service interface {
	GetOperatorDefinitionByID(ctx context.Context, defID string) (*pipelinePB.OperatorDefinition, error)
	ListOperatorDefinitions(context.Context, *pipelinePB.ListOperatorDefinitionsRequest) (*pipelinePB.ListOperatorDefinitionsResponse, error)

	ListPipelines(ctx context.Context, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetPipelineByUID(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)
	CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, pipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	ListNamespacePipelines(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, view View) (*pipelinePB.Pipeline, error)
	UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, updatedPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error)
	UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, id string, newID string) (*pipelinePB.Pipeline, error)
	DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) error
	ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) (*pipelinePB.Pipeline, error)
	GetNamespacePipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, id string) (uuid.UUID, error)
	CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, id string, targetNS resource.Namespace, targetID string) (*pipelinePB.Pipeline, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error)

	CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error)
	GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error)
	UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, updatedPipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error)
	DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error
	RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error
	UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error)

	CreateNamespaceSecret(ctx context.Context, ns resource.Namespace, secret *pipelinePB.Secret) (*pipelinePB.Secret, error)
	ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pipelinePB.Secret, int32, string, error)
	GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pipelinePB.Secret, error)
	UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pipelinePB.Secret) (*pipelinePB.Secret, error)
	DeleteNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) error

	// Influx API

	TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)

	TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)

	WriteNewPipelineDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error

	GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error)
	GetRscNamespaceAndNameID(ctx context.Context, path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(ctx context.Context, path string) (resource.Namespace, uuid.UUID, error)
	GetRscNamespaceAndNameIDAndReleaseID(ctx context.Context, path string) (resource.Namespace, string, string, error)
	convertOwnerPermalinkToName(ctx context.Context, permalink string) (string, error)
	convertOwnerNameToPermalink(ctx context.Context, name string) (string, error)

	PBToDBPipeline(ctx context.Context, ns resource.Namespace, pbPipeline *pipelinePB.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, view View, checkPermission bool) (*pipelinePB.Pipeline, error)
	DBToPBPipelines(ctx context.Context, dbPipeline []*datamodel.Pipeline, view View, checkPermission bool) ([]*pipelinePB.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease *datamodel.PipelineRelease, view View) (*pipelinePB.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view View) ([]*pipelinePB.PipelineRelease, error)

	PBToDBSecret(ctx context.Context, ns resource.Namespace, pbSecret *pipelinePB.Secret) (*datamodel.Secret, error)
	DBToPBSecret(ctx context.Context, dbSecret *datamodel.Secret) (*pipelinePB.Secret, error)
	DBToPBSecrets(ctx context.Context, dbSecrets []*datamodel.Secret) ([]*pipelinePB.Secret, error)

	ListComponentDefinitions(context.Context, *pipelinePB.ListComponentDefinitionsRequest) (*pipelinePB.ListComponentDefinitionsResponse, error)

	ListConnectorDefinitions(context.Context, *pipelinePB.ListConnectorDefinitionsRequest) (*pipelinePB.ListConnectorDefinitionsResponse, error)
	GetConnectorDefinitionByID(ctx context.Context, id string, view View) (*pipelinePB.ConnectorDefinition, error)

	// Influx API
	WriteNewConnectorDataPoint(ctx context.Context, data utils.ConnectorUsageMetricData, pipelineMetadata *structpb.Value) error

	// Helper functions
	RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct)
	KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct)
}

type service struct {
	repository               repository.Repository
	mgmtPrivateServiceClient mgmtPB.MgmtPrivateServiceClient
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
	rc *redis.Client,
	t client.Client,
	i api.WriteAPI,
	acl *acl.ACLClient,
) Service {
	logger, _ := logger.GetZapLogger(context.Background())
	return &service{
		repository:               r,
		mgmtPrivateServiceClient: u,
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

func generateShareCode() string {
	return randomStrWithCharset(32, charset)
}

// Note: Currently, we don't allow changing the owner ID. We are safe to use a cache with a longer TTL for this function.
func (s *service) convertOwnerPermalinkToName(ctx context.Context, permalink string) (string, error) {

	splits := strings.Split(permalink, "/")
	nsType := splits[0]
	uid := splits[1]
	key := fmt.Sprintf("user:%s:uid_to_id", uid)
	if id, err := s.redisClient.Get(ctx, key).Result(); err != redis.Nil {
		return fmt.Sprintf("%s/%s", nsType, id), nil
	}

	if nsType == "users" {
		userResp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		s.redisClient.Set(ctx, key, userResp.User.Id, 24*time.Hour)
		return fmt.Sprintf("users/%s", userResp.User.Id), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return "", fmt.Errorf("ConvertNamespaceToOwnerPath error")
		}
		s.redisClient.Set(ctx, key, orgResp.Organization.Id, 24*time.Hour)
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Id), nil
	}
}

func (s *service) fetchOwnerByPermalink(ctx context.Context, permalink string) (*mgmtPB.Owner, error) {

	key := fmt.Sprintf("owner_profile:%s", permalink)
	if b, err := s.redisClient.Get(ctx, key).Bytes(); err == nil {
		owner := &mgmtPB.Owner{}
		if protojson.Unmarshal(b, owner) == nil {
			return owner, nil
		}
	}

	if strings.HasPrefix(permalink, "users") {
		resp, err := s.mgmtPrivateServiceClient.LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_User{User: resp.User}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil
	} else {
		resp, err := s.mgmtPrivateServiceClient.LookUpOrganizationAdmin(ctx, &mgmtPB.LookUpOrganizationAdminRequest{Permalink: permalink})
		if err != nil {
			return nil, fmt.Errorf("fetchOwnerByPermalink error")
		}
		owner := &mgmtPB.Owner{Owner: &mgmtPB.Owner_Organization{Organization: resp.Organization}}
		if b, err := protojson.Marshal(owner); err == nil {
			s.redisClient.Set(ctx, key, b, 5*time.Minute)
		}
		return owner, nil

	}
}

// Note: Currently, we don't allow changing the owner ID. We are safe to use a cache with a longer TTL for this function.
func (s *service) convertOwnerNameToPermalink(ctx context.Context, name string) (string, error) {

	splits := strings.Split(name, "/")
	nsType := splits[0]
	id := splits[1]
	key := fmt.Sprintf("user:%s:id_to_uid", id)
	if uid, err := s.redisClient.Get(ctx, key).Result(); err != redis.Nil {
		return fmt.Sprintf("%s/%s", nsType, uid), nil
	}

	if nsType == "users" {
		userResp, err := s.mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("convertOwnerNameToPermalink error %w", err)
		}
		s.redisClient.Set(ctx, key, *userResp.User.Uid, 24*time.Hour)
		return fmt.Sprintf("users/%s", *userResp.User.Uid), nil
	} else {
		orgResp, err := s.mgmtPrivateServiceClient.GetOrganizationAdmin(ctx, &mgmtPB.GetOrganizationAdminRequest{Name: name})
		if err != nil {
			return "", fmt.Errorf("convertOwnerNameToPermalink error %w", err)
		}
		s.redisClient.Set(ctx, key, orgResp.Organization.Uid, 24*time.Hour)
		return fmt.Sprintf("organizations/%s", orgResp.Organization.Uid), nil
	}
}

func (s *service) checkNamespacePermission(ctx context.Context, ns resource.Namespace) error {
	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return err
		}
		if !granted {
			return ErrNoPermission
		}
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			return ErrNoPermission
		}
	}
	return nil
}

func (s *service) GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error) {

	uid := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	name, err := s.convertOwnerPermalinkToName(ctx, fmt.Sprintf("users/%s", uid))
	if err != nil {
		return resource.Namespace{}, fmt.Errorf("namespace error")
	}
	// TODO: optimize the flow to get namespace
	return resource.Namespace{
		NsType: resource.NamespaceType("users"),
		NsID:   strings.Split(name, "/")[1],
		NsUID:  uid,
	}, nil
}
func (s *service) GetRscNamespaceAndNameID(ctx context.Context, path string) (resource.Namespace, string, error) {

	if strings.HasPrefix(path, "user/") {

		uid := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
		splits := strings.Split(path, "/")

		name, err := s.convertOwnerPermalinkToName(ctx, fmt.Sprintf("users/%s", uid))
		if err != nil {
			return resource.Namespace{}, "", fmt.Errorf("namespace error")
		}

		return resource.Namespace{
			NsType: resource.NamespaceType("users"),
			NsID:   strings.Split(name, "/")[1],
			NsUID:  uid,
		}, splits[2], nil
	}

	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, "", fmt.Errorf("namespace error")
	}
	uidStr, err := s.convertOwnerNameToPermalink(ctx, fmt.Sprintf("%s/%s", splits[0], splits[1]))

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

func (s *service) GetRscNamespaceAndPermalinkUID(ctx context.Context, path string) (resource.Namespace, uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return resource.Namespace{}, uuid.Nil, fmt.Errorf("namespace error")
	}
	uidStr, err := s.convertOwnerNameToPermalink(ctx, fmt.Sprintf("%s/%s", splits[0], splits[1]))
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

func (s *service) GetRscNamespaceAndNameIDAndReleaseID(ctx context.Context, path string) (resource.Namespace, string, string, error) {
	ns, pipelineID, err := s.GetRscNamespaceAndNameID(ctx, path)
	if err != nil {
		return ns, pipelineID, "", err
	}
	splits := strings.Split(path, "/")

	if len(splits) < 6 {
		return ns, pipelineID, "", fmt.Errorf("path error")
	}
	return ns, pipelineID, splits[5], err
}

func (s *service) GetOperatorDefinitionByID(ctx context.Context, defID string) (*pipelinePB.OperatorDefinition, error) {
	return s.operator.GetOperatorDefinitionByID(defID, nil)
}

func (s *service) implementedOperatorDefinitions() []*pipelinePB.OperatorDefinition {
	allDefs := s.operator.ListOperatorDefinitions()

	implemented := make([]*pipelinePB.OperatorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if implementedReleaseStages[def.GetReleaseStage()] {
			implemented = append(implemented, def)
		}
	}

	return implemented
}

func (s *service) ListOperatorDefinitions(ctx context.Context, req *pipelinePB.ListOperatorDefinitionsRequest) (*pipelinePB.ListOperatorDefinitionsResponse, error) {
	pageSize := s.pageSizeInRange(req.GetPageSize())
	prevLastUID, err := s.lastUIDFromToken(req.GetPageToken())
	if err != nil {
		return nil, err
	}

	// The client of this use case is the console pipeline builder, so we want
	// to filter out the unimplemented definitions (that are present in the
	// ListComponentDefinitions method, used also for the marketing website).
	//
	// TODO we can use only the component definition list and let the clients
	// do the filtering in the query params.
	defs := s.implementedOperatorDefinitions()

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

func (s *service) ListPipelines(ctx context.Context, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	uidAllowList := []uuid.UUID{}
	var err error

	// TODO: optimize the logic
	if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbPipelines, totalSize, nextPageToken, err := s.repository.ListPipelines(ctx, int64(pageSize), pageToken, view == ViewBasic, filter, uidAllowList, showDeleted, true)
	if err != nil {
		return nil, 0, "", err
	}
	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view, true)
	return pbPipelines, int32(totalSize), nextPageToken, err

}

func (s *service) GetPipelineByUID(ctx context.Context, uid uuid.UUID, view View) (*pipelinePB.Pipeline, error) {

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", uid, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, uid, view == ViewBasic, true)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, view, true)
}

func (s *service) CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, pbPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	// TODO: optimize ACL model
	if ns.NsType == "organizations" {
		granted, err := s.aclClient.CheckPermission(ctx, "organization", ns.NsUID, "member")
		if err != nil {
			return nil, err
		}
		if !granted {
			return nil, ErrNoPermission
		}
	} else {
		if ns.NsUID != uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)) {
			return nil, ErrNoPermission
		}
	}

	dbPipeline, err := s.PBToDBPipeline(ctx, ns, pbPipeline)
	if err != nil {
		return nil, err
	}

	if dbPipeline.ShareCode == "" {
		dbPipeline.ShareCode = generateShareCode()
	}

	if err := s.repository.CreateNamespacePipeline(ctx, ownerPermalink, dbPipeline); err != nil {
		return nil, err
	}

	dbCreatedPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, dbPipeline.ID, false, true)
	if err != nil {
		return nil, err
	}
	ownerType := string(ns.NsType)[0 : len(string(ns.NsType))-1]
	ownerUID := ns.NsUID
	err = s.aclClient.SetOwner(ctx, "pipeline", dbCreatedPipeline.UID, ownerType, ownerUID)
	if err != nil {
		return nil, err
	}
	// TODO: use OpenFGA as single source of truth
	err = s.aclClient.SetPipelinePermissionMap(ctx, dbCreatedPipeline)
	if err != nil {
		return nil, err
	}

	pipeline, err := s.DBToPBPipeline(ctx, dbCreatedPipeline, ViewFull, false)
	if err != nil {
		return nil, err
	}
	pipeline.Permission = &pipelinePB.Permission{
		CanEdit:    true,
		CanTrigger: true,
	}
	return pipeline, nil
}

func (s *service) ListNamespacePipelines(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view View, visibility *pipelinePB.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	ownerPermalink := ns.Permalink()

	uidAllowList := []uuid.UUID{}
	var err error

	// TODO: optimize the logic
	if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PUBLIC {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
	} else if visibility != nil && *visibility == pipelinePB.Pipeline_VISIBILITY_PRIVATE {
		allUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
		publicUIDAllowList, err := s.aclClient.ListPermissions(ctx, "pipeline", "reader", true)
		if err != nil {
			return nil, 0, "", err
		}
		for _, uid := range allUIDAllowList {
			if !slices.Contains(publicUIDAllowList, uid) {
				uidAllowList = append(uidAllowList, uid)
			}
		}
	} else {
		uidAllowList, err = s.aclClient.ListPermissions(ctx, "pipeline", "reader", false)
		if err != nil {
			return nil, 0, "", err
		}
	}

	dbPipelines, ps, pt, err := s.repository.ListNamespacePipelines(ctx, ownerPermalink, int64(pageSize), pageToken, view == ViewBasic, filter, uidAllowList, showDeleted, true)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view, true)
	return pbPipelines, int32(ps), pt, err
}

func (s *service) ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.Pipeline, int32, string, error) {

	dbPipelines, ps, pt, err := s.repository.ListPipelinesAdmin(ctx, int64(pageSize), pageToken, view == ViewBasic, filter, showDeleted, true)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelines, err := s.DBToPBPipelines(ctx, dbPipelines, view, true)
	return pbPipelines, int32(ps), pt, err

}

func (s *service) GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, view View) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, view == ViewBasic, true)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	return s.DBToPBPipeline(ctx, dbPipeline, view, true)
}

func (s *service) GetNamespacePipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, id string) (uuid.UUID, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, true)
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

	dbPipeline, err := s.repository.GetPipelineByUIDAdmin(ctx, uid, view == ViewBasic, true)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, view, true)

}

func (s *service) UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, toUpdPipeline *pipelinePB.Pipeline) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.PBToDBPipeline(ctx, ns, toUpdPipeline)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	var existingPipeline *datamodel.Pipeline
	// Validation: Pipeline existence
	if existingPipeline, _ = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, false); existingPipeline == nil {
		return nil, err
	}

	if existingPipeline.ShareCode == "" {
		dbPipeline.ShareCode = generateShareCode()
	}

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, dbPipeline.UID, dbPipeline); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, toUpdPipeline.Id, false, true)
	if err != nil {
		return nil, err
	}

	oldSharing, _ := json.Marshal(existingPipeline.Sharing)
	newSharing, _ := json.Marshal(dbPipeline.Sharing)

	// TODO: use OpenFGA as single source of truth
	if string(oldSharing) != string(newSharing) {
		err = s.aclClient.SetPipelinePermissionMap(ctx, dbPipeline)
		if err != nil {
			return nil, err
		}
	}

	dbPipelineUpdated, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, err
	}
	pipeline, err := s.DBToPBPipeline(ctx, dbPipelineUpdated, ViewFull, false)
	if err != nil {
		return nil, err
	}
	pipeline.Permission = &pipelinePB.Permission{
		CanEdit:    true,
		CanTrigger: true,
	}
	return pipeline, nil
}

func (s *service) DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) error {
	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	// TODO: pagination
	pipelineReleases, _, _, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, dbPipeline.UID, 1000, "", false, filtering.Filter{}, false, false)
	if err != nil {
		return err
	}

	ch := make(chan error)
	var wg sync.WaitGroup
	wg.Add(len(pipelineReleases))

	for idx := range pipelineReleases {
		go func(r *datamodel.PipelineRelease) {
			defer wg.Done()
			err := s.DeleteNamespacePipelineReleaseByID(ctx, ns, dbPipeline.UID, r.ID)
			ch <- err
		}(pipelineReleases[idx])
	}
	for range pipelineReleases {
		err = <-ch
		if err != nil {
			return err
		}
	}

	err = s.aclClient.Purge(ctx, "pipeline", dbPipeline.UID)
	if err != nil {
		return err
	}
	return s.repository.DeleteNamespacePipelineByID(ctx, ownerPermalink, id)
}

func (s *service) CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, id string, targetNS resource.Namespace, targetID string) (*pipelinePB.Pipeline, error) {
	sourcePipeline, err := s.GetNamespacePipelineByID(ctx, ns, id, ViewRecipe)
	if err != nil {
		return nil, err
	}
	sourcePipeline.Id = targetID
	targetPipeline, err := s.CreateNamespacePipeline(ctx, targetNS, sourcePipeline)
	if err != nil {
		return nil, err
	}
	return targetPipeline, nil
}

func (s *service) ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	recipeErr := s.checkRecipe(ownerPermalink, dbPipeline.Recipe)

	if recipeErr != nil {
		return nil, recipeErr
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, ViewFull, true)

}

func (s *service) UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, id string, newID string) (*pipelinePB.Pipeline, error) {

	ownerPermalink := ns.Permalink()

	// Validation: Pipeline existence
	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, true, true)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	if err := s.repository.UpdateNamespacePipelineIDByID(ctx, ownerPermalink, id, newID); err != nil {
		return nil, err
	}

	dbPipeline, err = s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, newID, false, true)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipeline(ctx, dbPipeline, ViewFull, true)
}

func (s *service) preTriggerPipeline(ctx context.Context, isAdmin bool, ns resource.Namespace, recipe *datamodel.Recipe, pipelineInputs []*structpb.Struct) error {

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
		userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
		value, err := s.redisClient.Get(ctx, fmt.Sprintf("user_rate_limit:user:%s", userUID)).Result()
		// TODO: use a more robust way to check key exist
		if !errors.Is(err, redis.Nil) {
			requestLeft, _ := strconv.ParseInt(value, 10, 64)
			if requestLeft <= 0 {
				return ErrRateLimiting
			} else {
				_ = s.redisClient.Decr(ctx, fmt.Sprintf("user_rate_limit:user:%s", userUID))
			}
		}
	}

	var metadata []byte

	instillFormatMap := map[string]string{}

	schStruct := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	schStruct.Fields["type"] = structpb.NewStringValue("object")
	b, _ := json.Marshal(recipe.Trigger.TriggerByRequest.Fields)
	properties := &structpb.Struct{}
	_ = protojson.Unmarshal(b, properties)
	schStruct.Fields["properties"] = structpb.NewStructValue(properties)
	for k, v := range recipe.Trigger.TriggerByRequest.Fields {
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
	return s.getOperationFromWorkflowInfo(ctx, workflowExecutionRes.WorkflowExecutionInfo)
}

func (s *service) getOperationFromWorkflowInfo(ctx context.Context, workflowExecutionInfo *workflowpb.WorkflowExecutionInfo) (*longrunningpb.Operation, error) {
	operation := longrunningpb.Operation{}

	switch workflowExecutionInfo.Status {
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:

		pipelineResp := &pipelinePB.TriggerUserPipelineResponse{}

		blobRedisKey := fmt.Sprintf("async_pipeline_response:%s", workflowExecutionInfo.Execution.WorkflowId)
		blob, err := s.redisClient.Get(ctx, blobRedisKey).Bytes()
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

func (s *service) CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pipelineRelease *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, false)
	if err != nil {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	dbPipelineReleaseToCreate, err := s.PBToDBPipelineRelease(ctx, pipelineUID, pipelineRelease)
	if err != nil {
		return nil, err
	}

	dbPipelineReleaseToCreate.Recipe = dbPipeline.Recipe
	dbPipelineReleaseToCreate.Metadata = dbPipeline.Metadata

	if err := s.repository.CreateNamespacePipelineRelease(ctx, ownerPermalink, pipelineUID, dbPipelineReleaseToCreate); err != nil {
		return nil, err
	}

	dbCreatedPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, pipelineRelease.Id, false)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, dbPipeline, dbCreatedPipelineRelease, ViewFull)

}
func (s *service) ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pageSize int32, pageToken string, view View, filter filtering.Filter, showDeleted bool) ([]*pipelinePB.PipelineRelease, int32, string, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, 0, "", ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, 0, "", err
	} else if !granted {
		return nil, 0, "", ErrNotFound
	}

	dbPipelineReleases, ps, pt, err := s.repository.ListNamespacePipelineReleases(ctx, ownerPermalink, pipelineUID, int64(pageSize), pageToken, view == ViewBasic, filter, showDeleted, true)
	if err != nil {
		return nil, 0, "", err
	}

	pbPipelineReleases, err := s.DBToPBPipelineReleases(ctx, dbPipeline, dbPipelineReleases, view)
	return pbPipelineReleases, int32(ps), pt, err
}

func (s *service) GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, view View) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, view == ViewBasic)
	if err != nil {
		return nil, err
	}

	return s.DBToPBPipelineRelease(ctx, dbPipeline, dbPipelineRelease, view)

}

func (s *service) UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, toUpdPipeline *pipelinePB.PipelineRelease) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	if _, err := s.GetNamespacePipelineReleaseByID(ctx, ns, pipelineUID, id, ViewBasic); err != nil {
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

	return s.DBToPBPipelineRelease(ctx, dbPipeline, dbPipelineRelease, ViewFull)
}

func (s *service) UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, newID string) (*pipelinePB.PipelineRelease, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
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

	return s.DBToPBPipelineRelease(ctx, dbPipeline, dbPipelineRelease, ViewFull)
}

func (s *service) DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, true, false)
	if err != nil {
		return ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNoPermission
	}

	return s.repository.DeleteNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id)
}

func (s *service) RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error {
	ownerPermalink := ns.Permalink()

	pipeline, err := s.GetPipelineByUID(ctx, pipelineUID, ViewBasic)
	if err != nil {
		return ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", uuid.FromStringOrNil(pipeline.GetUid()), "admin"); err != nil {
		return err
	} else if !granted {
		return ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", uuid.FromStringOrNil(pipeline.GetUid()), "admin"); err != nil {
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
	if existingPipeline, err = s.repository.GetPipelineByUIDAdmin(ctx, pipelineUID, false, true); err != nil {
		return err
	}
	existingPipeline.Recipe = dbPipelineRelease.Recipe

	if err := s.repository.UpdateNamespacePipelineByUID(ctx, existingPipeline.UID, existingPipeline); err != nil {
		return err
	}

	return nil
}

// TODO: share the code with worker/workflow.go
func (s *service) triggerPipeline(
	ctx context.Context,
	ns resource.Namespace,
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

	err := s.preTriggerPipeline(ctx, isAdmin, ns, recipe, pipelineInputs)
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
			ctx,
			inputBlobRedisKey,
			inputJSON,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
		defer s.redisClient.Del(ctx, inputBlobRedisKey)
	}

	m := worker.WorkflowMeta{
		HeaderAuthorization: resource.GetRequestSingleHeader(ctx, "authorization"),
	}
	b, _ := json.Marshal(m)
	metadataRedisKey := fmt.Sprintf("async_pipeline_metadata:%s", pipelineTriggerID)

	s.redisClient.Set(
		ctx,
		metadataRedisKey,
		b,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)
	defer s.redisClient.Del(ctx, metadataRedisKey)

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

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))

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
			UserUID:                    userUID,
			ReturnTraces:               returnTraces,
			Mode:                       mgmtPB.Mode_MODE_SYNC,
			MetadataRedisKey:           metadataRedisKey,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("unable to execute workflow: %s", err.Error()))
		return nil, nil, err
	}

	var result *worker.TriggerPipelineWorkflowResponse
	err = we.Get(ctx, &result)
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

	blob, err := s.redisClient.Get(ctx, result.OutputBlobRedisKey).Bytes()
	if err != nil {
		return nil, nil, err
	}
	s.redisClient.Del(ctx, result.OutputBlobRedisKey)

	err = protojson.Unmarshal(blob, pipelineResp)
	if err != nil {
		return nil, nil, err
	}

	return pipelineResp.Outputs, pipelineResp.Metadata, nil
}

func (s *service) triggerAsyncPipeline(
	ctx context.Context,
	ns resource.Namespace,
	recipe *datamodel.Recipe,
	isAdmin bool,
	pipelineID string,
	pipelineUID uuid.UUID,
	pipelineReleaseID string,
	pipelineReleaseUID uuid.UUID,
	pipelineInputs []*structpb.Struct,
	pipelineTriggerID string,
	returnTraces bool) (*longrunningpb.Operation, error) {

	err := s.preTriggerPipeline(ctx, isAdmin, ns, recipe, pipelineInputs)
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
			ctx,
			inputBlobRedisKey,
			inputJSON,
			time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
		)
		inputBlobRedisKeys = append(inputBlobRedisKeys, inputBlobRedisKey)
	}

	m := worker.WorkflowMeta{
		HeaderAuthorization: resource.GetRequestSingleHeader(ctx, "authorization"),
	}
	b, _ := json.Marshal(m)
	metadataRedisKey := fmt.Sprintf("async_pipeline_metadata:%s", pipelineTriggerID)

	s.redisClient.Set(
		ctx,
		metadataRedisKey,
		b,
		time.Duration(config.Config.Server.Workflow.MaxWorkflowTimeout)*time.Second,
	)

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

	userUID := uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
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
			UserUID:                    userUID,
			ReturnTraces:               returnTraces,
			Mode:                       mgmtPB.Mode_MODE_ASYNC,
			MetadataRedisKey:           metadataRedisKey,
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

func (s *service) TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNoPermission
	}

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, nil, err
	}

	return s.triggerPipeline(ctx, ns, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerID, returnTraces)

}

func (s *service) TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetNamespacePipelineByID(ctx, ownerPermalink, id, false, true)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	}

	return s.triggerAsyncPipeline(ctx, ns, dbPipeline.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, "", uuid.Nil, inputs, pipelineTriggerID, returnTraces)

}

func (s *service) TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinePB.TriggerMetadata, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, true)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor"); err != nil {
		return nil, nil, err
	} else if !granted {
		return nil, nil, ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, nil, err
	}

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, nil, err
	}

	latestReleaseUID, err := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, dbPipeline.ID)
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

	return s.triggerPipeline(ctx, ns, dbPipelineRelease.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerID, returnTraces)
}

func (s *service) TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, inputs []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error) {

	ownerPermalink := ns.Permalink()

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, pipelineUID, false, true)
	if err != nil {
		return nil, ErrNotFound
	}
	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "reader"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNotFound
	}

	if granted, err := s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "executor"); err != nil {
		return nil, err
	} else if !granted {
		return nil, ErrNoPermission
	}

	dbPipelineRelease, err := s.repository.GetNamespacePipelineReleaseByID(ctx, ownerPermalink, pipelineUID, id, false)
	if err != nil {
		return nil, err
	}

	isAdmin := false
	if isAdmin, err = s.aclClient.CheckPermission(ctx, "pipeline", dbPipeline.UID, "admin"); err != nil {
		return nil, err
	}

	latestReleaseUID, err := s.GetNamespacePipelineLatestReleaseUID(ctx, ns, dbPipeline.ID)
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

	return s.triggerAsyncPipeline(ctx, ns, dbPipelineRelease.Recipe, isAdmin, dbPipeline.ID, dbPipeline.UID, dbPipelineRelease.ID, dbPipelineRelease.UID, inputs, pipelineTriggerID, returnTraces)
}

func (s *service) RemoveCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.RemoveCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
}

func (s *service) KeepCredentialFieldsWithMaskString(dbConnDefID string, config *structpb.Struct) {
	utils.KeepCredentialFieldsWithMaskString(s.connector, dbConnDefID, config)
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

func (s *service) pageInRange(page int32) int {
	if page <= 0 {
		return 0
	}

	return int(page)
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
	pageSize := s.pageSizeInRange(req.GetPageSize())
	page := s.pageInRange(req.GetPage())
	view := parseView(int32(req.GetView()))

	var compType pipelinePB.ComponentType
	var releaseStage pipelinePB.ComponentDefinition_ReleaseStage
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("q_title", filtering.TypeString),
		filtering.DeclareEnumIdent("release_stage", releaseStage.Type()),
		filtering.DeclareEnumIdent("component_type", compType.Type()),
	)
	if err != nil {
		return nil, err
	}

	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}

	p := repository.ListComponentDefinitionsParams{
		Offset: page * pageSize,
		Limit:  pageSize,
		Filter: filter,
	}

	uids, totalSize, err := s.repository.ListComponentDefinitionUIDs(ctx, p)
	if err != nil {
		return nil, err
	}

	defs := make([]*pipelinePB.ComponentDefinition, len(uids))

	for i, uid := range uids {
		d := &pipelinePB.ComponentDefinition{
			Type: pipelinePB.ComponentType(uid.ComponentType),
		}

		switch d.Type {
		case pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
			pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION,
			pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA:

			cd, err := s.connector.GetConnectorDefinitionByUID(uid.UID, nil, nil)
			if err != nil {
				return nil, err
			}

			cd = proto.Clone(cd).(*pipelinePB.ConnectorDefinition)
			s.applyViewToConnectorDefinition(cd, view)
			d.Definition = &pipelinePB.ComponentDefinition_ConnectorDefinition{
				ConnectorDefinition: cd,
			}
		case pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR:
			od, err := s.operator.GetOperatorDefinitionByUID(uid.UID, nil)
			if err != nil {
				return nil, err
			}

			od = proto.Clone(od).(*pipelinePB.OperatorDefinition)
			s.applyViewToOperatorDefinition(od, view)
			d.Definition = &pipelinePB.ComponentDefinition_OperatorDefinition{
				OperatorDefinition: od,
			}
		default:
			return nil, fmt.Errorf("invalid component definition type in database")
		}

		defs[i] = d
	}

	resp := &pipelinePB.ListComponentDefinitionsResponse{
		PageSize:             int32(pageSize),
		Page:                 int32(page),
		TotalSize:            int32(totalSize),
		ComponentDefinitions: defs,
	}

	return resp, nil
}

var implementedReleaseStages = map[pipelinePB.ComponentDefinition_ReleaseStage]bool{
	pipelinePB.ComponentDefinition_RELEASE_STAGE_ALPHA: true,
	pipelinePB.ComponentDefinition_RELEASE_STAGE_BETA:  true,
	pipelinePB.ComponentDefinition_RELEASE_STAGE_GA:    true,
}

func (s *service) implementedConnectorDefinitions() []*pipelinePB.ConnectorDefinition {
	allDefs := s.connector.ListConnectorDefinitions()

	implemented := make([]*pipelinePB.ConnectorDefinition, 0, len(allDefs))
	for _, def := range allDefs {
		if implementedReleaseStages[def.GetReleaseStage()] {
			implemented = append(implemented, def)
		}
	}

	return implemented
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

	// The client of this use case is the console pipeline builder, so we want
	// to filter out the unimplemented definitions (that are present in the
	// ListComponentDefinitions method, used also for the marketing website).
	//
	// TODO we can use only the component definition list and let the clients
	// do the filtering in the query params.
	defs := s.filterConnectorDefinitions(s.implementedConnectorDefinitions(), filter)

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

func (s *service) CreateNamespaceSecret(ctx context.Context, ns resource.Namespace, pbSecret *pipelinePB.Secret) (*pipelinePB.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	dbSecret, err := s.PBToDBSecret(ctx, ns, pbSecret)
	if err != nil {
		return nil, err
	}

	if err := s.repository.CreateNamespaceSecret(ctx, ns.Permalink(), dbSecret); err != nil {
		return nil, err
	}

	dbCreatedSecret, err := s.repository.GetNamespaceSecretByID(ctx, ns.Permalink(), dbSecret.ID)
	if err != nil {
		return nil, err
	}

	return s.DBToPBSecret(ctx, dbCreatedSecret)

}

func (s *service) ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pipelinePB.Secret, int32, string, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, 0, "", err
	}

	dbSecrets, ps, pt, err := s.repository.ListNamespaceSecrets(ctx, ns.Permalink(), int64(pageSize), pageToken, filter)
	if err != nil {
		return nil, 0, "", err
	}

	pbSecrets, err := s.DBToPBSecrets(ctx, dbSecrets)
	return pbSecrets, int32(ps), pt, err
}

func (s *service) GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pipelinePB.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.repository.GetNamespaceSecretByID(ctx, ownerPermalink, id)
	if err != nil {
		return nil, ErrNotFound
	}
	return s.DBToPBSecret(ctx, dbSecret)
}

func (s *service) UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pipelinePB.Secret) (*pipelinePB.Secret, error) {

	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return nil, err
	}

	ownerPermalink := ns.Permalink()

	dbSecret, err := s.PBToDBSecret(ctx, ns, updatedSecret)
	if err != nil {
		return nil, ErrNotFound
	}

	if _, err = s.repository.GetNamespaceSecretByID(ctx, ownerPermalink, id); err != nil {
		return nil, err
	}

	if err := s.repository.UpdateNamespaceSecretByID(ctx, ns.Permalink(), id, dbSecret); err != nil {
		return nil, err
	}

	return s.GetNamespaceSecretByID(ctx, ns, id)
}

func (s *service) DeleteNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) error {
	if err := s.checkNamespacePermission(ctx, ns); err != nil {
		return err
	}
	ownerPermalink := ns.Permalink()

	return s.repository.DeleteNamespaceSecretByID(ctx, ownerPermalink, id)
}
