package service

import (
	"context"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/minio"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	artifactpb "github.com/instill-ai/protogen-go/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
	logx "github.com/instill-ai/x/log"
)

// Service interface
type Service interface {
	GetHubStats(ctx context.Context) (*pipelinepb.GetHubStatsResponse, error)
	ListPublicPipelines(ctx context.Context, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, visibility *pipelinepb.Pipeline_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*pipelinepb.Pipeline, int32, string, error)
	GetPipelineByUID(ctx context.Context, uid uuid.UUID, view pipelinepb.Pipeline_View) (*pipelinepb.Pipeline, error)
	CreatePipeline(ctx context.Context, ns resource.Namespace, pipeline *pipelinepb.Pipeline) (*pipelinepb.Pipeline, error)
	ListPipelines(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, visibility *pipelinepb.Pipeline_Visibility, filter filtering.Filter, showDeleted bool, order ordering.OrderBy) ([]*pipelinepb.Pipeline, int32, string, error)
	GetPipelineByID(ctx context.Context, ns resource.Namespace, id string, view pipelinepb.Pipeline_View) (*pipelinepb.Pipeline, error)
	GetPipelineUIDByID(ctx context.Context, ns resource.Namespace, id string) (uuid.UUID, error)
	UpdatePipelineByID(ctx context.Context, ns resource.Namespace, id string, updatedPipeline *pipelinepb.Pipeline) (*pipelinepb.Pipeline, error)
	UpdatePipelineIDByID(ctx context.Context, ns resource.Namespace, id string, newID string) (*pipelinepb.Pipeline, error)
	DeletePipelineByID(ctx context.Context, ns resource.Namespace, id string) error
	ValidatePipelineByID(ctx context.Context, ns resource.Namespace, id string) ([]*pipelinepb.ErrPipelineValidation, error)
	GetPipelineLatestReleaseUID(ctx context.Context, ns resource.Namespace, id string) (uuid.UUID, error)
	ClonePipeline(ctx context.Context, ns resource.Namespace, id, targetNamespaceID, targetPipelineID, description string, sharing *pipelinepb.Sharing) (*pipelinepb.Pipeline, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, filter filtering.Filter, showDeleted bool) ([]*pipelinepb.Pipeline, int32, string, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, view pipelinepb.Pipeline_View) (*pipelinepb.Pipeline, error)

	CreatePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pipelineRelease *pipelinepb.PipelineRelease) (*pipelinepb.PipelineRelease, error)
	ListPipelineReleases(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, pageSize int32, pageToken string, view pipelinepb.Pipeline_View, filter filtering.Filter, showDeleted bool) ([]*pipelinepb.PipelineRelease, int32, string, error)
	GetPipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, view pipelinepb.Pipeline_View) (*pipelinepb.PipelineRelease, error)
	UpdatePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, updatedPipelineRelease *pipelinepb.PipelineRelease) (*pipelinepb.PipelineRelease, error)
	DeletePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error
	RestorePipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string) error
	UpdatePipelineReleaseIDByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, newID string) (*pipelinepb.PipelineRelease, error)
	ClonePipelineRelease(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id, targetNamespaceID, targetPipelineID, description string, sharing *pipelinepb.Sharing) (*pipelinepb.Pipeline, error)

	CreateNamespaceSecret(ctx context.Context, ns resource.Namespace, secret *pipelinepb.Secret) (*pipelinepb.Secret, error)
	ListNamespaceSecrets(ctx context.Context, ns resource.Namespace, pageSize int32, pageToken string, filter filtering.Filter) ([]*pipelinepb.Secret, int32, string, error)
	GetNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) (*pipelinepb.Secret, error)
	UpdateNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string, updatedSecret *pipelinepb.Secret) (*pipelinepb.Secret, error)
	DeleteNamespaceSecretByID(ctx context.Context, ns resource.Namespace, id string) error

	TriggerPipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error)
	TriggerAsyncPipelineByID(ctx context.Context, ns resource.Namespace, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)

	DispatchPipelineWebhookEvent(ctx context.Context, params DispatchPipelineWebhookEventParams) (DispatchPipelineWebhookEventResult, error)

	TriggerPipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) ([]*structpb.Struct, *pipelinepb.TriggerMetadata, error)
	TriggerAsyncPipelineReleaseByID(ctx context.Context, ns resource.Namespace, pipelineUID uuid.UUID, id string, data []*pipelinepb.TriggerData, pipelineTriggerID string, returnTraces bool) (*longrunningpb.Operation, error)
	GetOperation(ctx context.Context, workflowID string) (*longrunningpb.Operation, error)

	GetCtxUserNamespace(ctx context.Context) (resource.Namespace, error)
	GetNamespaceByID(ctx context.Context, namespaceID string) (resource.Namespace, error)
	GetNamespaceByUID(ctx context.Context, namespaceUID uuid.UUID) (resource.Namespace, error)

	ListComponentDefinitions(context.Context, *pipelinepb.ListComponentDefinitionsRequest) (*pipelinepb.ListComponentDefinitionsResponse, error)

	ListPipelineRuns(ctx context.Context, req *pipelinepb.ListPipelineRunsRequest, filter filtering.Filter) (*pipelinepb.ListPipelineRunsResponse, error)
	ListComponentRuns(ctx context.Context, req *pipelinepb.ListComponentRunsRequest, filter filtering.Filter) (*pipelinepb.ListComponentRunsResponse, error)
	ListPipelineRunsByRequester(ctx context.Context, req *pipelinepb.ListPipelineRunsByRequesterRequest) (*pipelinepb.ListPipelineRunsByRequesterResponse, error)

	GetIntegration(_ context.Context, id string, _ pipelinepb.View) (*pipelinepb.Integration, error)
	ListIntegrations(context.Context, *pipelinepb.ListIntegrationsRequest) (*pipelinepb.ListIntegrationsResponse, error)
	CreateNamespaceConnection(context.Context, *pipelinepb.CreateNamespaceConnectionRequest) (*pipelinepb.Connection, error)
	UpdateNamespaceConnection(context.Context, *pipelinepb.UpdateNamespaceConnectionRequest) (*pipelinepb.Connection, error)
	DeleteNamespaceConnection(_ context.Context, namespaceID, id string) error
	GetNamespaceConnection(context.Context, *pipelinepb.GetNamespaceConnectionRequest) (*pipelinepb.Connection, error)
	ListNamespaceConnections(context.Context, *pipelinepb.ListNamespaceConnectionsRequest) (*pipelinepb.ListNamespaceConnectionsResponse, error)
	ListPipelineIDsByConnectionID(context.Context, *pipelinepb.ListPipelineIDsByConnectionIDRequest) (*pipelinepb.ListPipelineIDsByConnectionIDResponse, error)
	GetConnectionByUIDAdmin(context.Context, uuid.UUID, pipelinepb.View) (*pipelinepb.Connection, error)
}

// TriggerResult defines a new type to encapsulate the stream data
type TriggerResult struct {
	Struct   []*structpb.Struct
	Metadata *pipelinepb.TriggerMetadata
}

// Now, we don't need the artifact service client in the service layer.
// However, we keep it here for now because we may need it in the future.
// service is the implementation of the Service interface
type service struct {
	repository                   repository.Repository
	redisClient                  *redis.Client
	temporalClient               client.Client
	component                    *componentstore.Store
	mgmtPublicServiceClient      mgmtpb.MgmtPublicServiceClient
	mgmtPrivateServiceClient     mgmtpb.MgmtPrivateServiceClient
	aclClient                    acl.ACLClientInterface
	converter                    Converter
	minioClient                  minio.Client
	memory                       *memory.Store
	log                          *zap.Logger
	retentionHandler             MetadataRetentionHandler
	binaryFetcher                external.BinaryFetcher
	artifactPublicServiceClient  artifactpb.ArtifactPublicServiceClient
	artifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
}

// NewService initiates a service instance
func NewService(
	repository repository.Repository,
	redisClient *redis.Client,
	temporalClient client.Client,
	aclClient acl.ACLClientInterface,
	converter Converter,
	mgmtPublicServiceClient mgmtpb.MgmtPublicServiceClient,
	mgmtPrivateServiceClient mgmtpb.MgmtPrivateServiceClient,
	minioClient minio.Client,
	componentStore *componentstore.Store,
	memory *memory.Store,
	retentionHandler MetadataRetentionHandler,
	binaryFetcher external.BinaryFetcher,
	artifactPublicServiceClient artifactpb.ArtifactPublicServiceClient,
	artifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient,
) Service {
	zapLogger, _ := logx.GetZapLogger(context.Background())

	return &service{
		repository:                   repository,
		redisClient:                  redisClient,
		temporalClient:               temporalClient,
		mgmtPublicServiceClient:      mgmtPublicServiceClient,
		mgmtPrivateServiceClient:     mgmtPrivateServiceClient,
		component:                    componentStore,
		aclClient:                    aclClient,
		converter:                    converter,
		minioClient:                  minioClient,
		memory:                       memory,
		log:                          zapLogger,
		retentionHandler:             retentionHandler,
		binaryFetcher:                binaryFetcher,
		artifactPublicServiceClient:  artifactPublicServiceClient,
		artifactPrivateServiceClient: artifactPrivateServiceClient,
	}
}
