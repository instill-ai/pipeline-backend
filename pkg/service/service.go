package service

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	component "github.com/instill-ai/component/pkg/base"
	connector "github.com/instill-ai/component/pkg/connector"
	operator "github.com/instill-ai/component/pkg/operator"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// Service interface
type Service interface {
	GetOperatorDefinitionByID(ctx context.Context, defID string) (*pb.OperatorDefinition, error)
	ListOperatorDefinitions(context.Context, *pb.ListOperatorDefinitionsRequest) (*pb.ListOperatorDefinitionsResponse, error)

	ListPipelines(ctx context.Context, pageSize int32, pageToken string, view pb.Pipeline_View, visibility *pb.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pb.Pipeline, int32, string, error)
	GetPipelineByUID(ctx context.Context, uid uuid.UUID, view pb.Pipeline_View) (*pb.Pipeline, error)
	CreateNamespacePipeline(ctx context.Context, ns resource.Namespace, pipeline *pb.Pipeline) (*pb.Pipeline, error)
	ListNamespacePipelines(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view pb.Pipeline_View, visibility *pb.Pipeline_Visibility, filter filtering.Filter, showDeleted bool) ([]*pb.Pipeline, int32, string, error)
	GetNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, view pb.Pipeline_View) (*pb.Pipeline, error)
	UpdateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, updatedPipeline *pb.Pipeline) (*pb.Pipeline, error)
	UpdateNamespacePipelineIDByID(ctx context.Context, ns resource.Namespace, id string, newID string) (*pb.Pipeline, error)
	DeleteNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) error
	ValidateNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string) (*pb.Pipeline, error)
	GetNamespacePipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, id string) (uuid.UUID, error)
	CloneNamespacePipeline(ctx context.Context, ns resource.Namespace, id string, targetNS resource.Namespace, targetID string) (*pb.Pipeline, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view pb.Pipeline_View, filter filtering.Filter, showDeleted bool) ([]*pb.Pipeline, int32, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view pb.Pipeline_View) (*pb.Pipeline, error)

	CreateNamespacePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pipelineRelease *pb.PipelineRelease) (*pb.PipelineRelease, error)
	ListNamespacePipelineReleases(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pageSize int32, pageToken string, view pb.Pipeline_View, filter filtering.Filter, showDeleted bool) ([]*pb.PipelineRelease, int32, string, error)
	GetNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, view pb.Pipeline_View) (*pb.PipelineRelease, error)
	UpdateNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, updatedPipelineRelease *pb.PipelineRelease) (*pb.PipelineRelease, error)
	DeleteNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error
	RestoreNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error
	UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, newID string) (*pb.PipelineRelease, error)

	CreateNamespaceSecret(ctx context.Context, ns resource.Namespace, secret *pb.Secret) (*pb.Secret, error)
	ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pb.Secret, int32, string, error)
	GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pb.Secret, error)
	UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pb.Secret) (*pb.Secret, error)
	DeleteNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) error

	// Influx API

	TriggerNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pb.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineByID(ctx context.Context, ns resource.Namespace, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)

	TriggerNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pb.TriggerMetadata, error)
	TriggerAsyncNamespacePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, req []*structpb.Struct, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)

	WriteNewPipelineDataPoint(ctx context.Context, data utils.PipelineUsageMetricData) error

	GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error)
	GetRscNamespaceAndNameID(ctx context.Context, path string) (resource.Namespace, string, error)
	GetRscNamespaceAndPermalinkUID(ctx context.Context, path string) (resource.Namespace, uuid.UUID, error)
	GetRscNamespaceAndNameIDAndReleaseID(ctx context.Context, path string) (resource.Namespace, string, string, error)
	convertOwnerPermalinkToName(ctx context.Context, permalink string) (string, error)
	convertOwnerNameToPermalink(ctx context.Context, name string) (string, error)

	PBToDBPipeline(ctx context.Context, ns resource.Namespace, pbPipeline *pb.Pipeline) (*datamodel.Pipeline, error)
	DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) (*pb.Pipeline, error)
	DBToPBPipelines(ctx context.Context, dbPipeline []*datamodel.Pipeline, view pb.Pipeline_View, checkPermission bool) ([]*pb.Pipeline, error)

	PBToDBPipelineRelease(ctx context.Context, pipelineUID uuid.UUID, pbPipelineRelease *pb.PipelineRelease) (*datamodel.PipelineRelease, error)
	DBToPBPipelineRelease(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease *datamodel.PipelineRelease, view pb.Pipeline_View) (*pb.PipelineRelease, error)
	DBToPBPipelineReleases(ctx context.Context, dbPipeline *datamodel.Pipeline, dbPipelineRelease []*datamodel.PipelineRelease, view pb.Pipeline_View) ([]*pb.PipelineRelease, error)

	PBToDBSecret(ctx context.Context, ns resource.Namespace, pbSecret *pb.Secret) (*datamodel.Secret, error)
	DBToPBSecret(ctx context.Context, dbSecret *datamodel.Secret) (*pb.Secret, error)
	DBToPBSecrets(ctx context.Context, dbSecrets []*datamodel.Secret) ([]*pb.Secret, error)

	ListComponentDefinitions(context.Context, *pb.ListComponentDefinitionsRequest) (*pb.ListComponentDefinitionsResponse, error)

	ListConnectorDefinitions(context.Context, *pb.ListConnectorDefinitionsRequest) (*pb.ListConnectorDefinitionsResponse, error)
	GetConnectorDefinitionByID(ctx context.Context, id string, view pb.ComponentDefinition_View) (*pb.ConnectorDefinition, error)

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
