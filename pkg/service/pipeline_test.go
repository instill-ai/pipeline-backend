package service

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/go-redis/redismock/v9"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"
	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/mock"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/minio"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

func fakeNamespace() resource.Namespace {
	return resource.Namespace{
		NsType: resource.User,
	}
}

func newDataPipeline(uid uuid.UUID, tagNames [2]string) datamodel.Pipeline {
	fakeRecipe := fakeRecipe()
	return datamodel.Pipeline{
		ID: "pipelineID",
		BaseDynamic: datamodel.BaseDynamic{
			UID: uid,
		},
		Recipe: &fakeRecipe,
		Owner:  "ChunHao/haha",
		Tags: []*datamodel.Tag{
			{
				TagName: tagNames[0],
			},
			{
				TagName: tagNames[1],
			},
		},
	}
}

func fakeRecipe() datamodel.Recipe {
	return datamodel.Recipe{}
}

func TestService_UpdateNamespacePipelineByID(t *testing.T) {
	c := quicktest.New(t)
	mc := minimock.NewController(t)
	ctx := context.Background()

	repo := mock.NewRepositoryMock(mc)
	redisClient, _ := redismock.NewClientMock()
	var temporalClient client.Client

	aclClient := mock.NewACLClientInterfaceMock(mc)
	converter := mock.NewConverterMock(mc)
	mgmtPrivateClient := mock.NewMgmtPrivateServiceClientMock(mc)

	compStore := componentstore.Init(componentstore.InitParams{
		Secrets:       config.Config.Component.Secrets,
		BinaryFetcher: external.NewBinaryFetcher(),
	})

	workerUID, _ := uuid.NewV4()
	service := newService(
		serviceConfig{
			repository:               repo,
			redisClient:              redisClient,
			temporalClient:           temporalClient,
			aCLClient:                aclClient,
			converter:                converter,
			mgmtPrivateServiceClient: mgmtPrivateClient,
			componentStore:           compStore,
			memory:                   memory.NewStore(nil),
			workerUID:                workerUID,
		},
	)

	aclClient.CheckPermissionMock.Return(true, nil)
	uid, err := uuid.NewV4()
	c.Assert(err, quicktest.IsNil)
	dataPipeline := newDataPipeline(uid, [2]string{"tag1", "tag3"})
	newDataPipeline := newDataPipeline(uid, [2]string{"tag1", "tag2"})

	repo.GetNamespacePipelineByIDMock.Return(&dataPipeline, nil)
	repo.UpdateNamespacePipelineByUIDMock.Return(nil)
	repo.GetPipelineByUIDMock.Return(&dataPipeline, nil)
	repo.DeletePipelineTagsMock.Expect(ctx, uid, []string{"tag3"}).Return(nil)
	repo.CreatePipelineTagsMock.Expect(ctx, uid, []string{"tag2"}).Return(nil)
	repo.ListPipelineRunOnsMock.Expect(ctx, uid).Return(repository.PipelineRunOnList{}, nil)

	converter.ConvertPipelineToDBMock.Return(&newDataPipeline, nil)

	pbPipeline := pb.Pipeline{
		Id:   "pipelineID",
		Name: "pipelineName",
		Tags: []string{"tag1", "tag2"},
	}
	converter.ConvertPipelineToPBMock.Return(&pbPipeline, nil)

	updatedPbPipeline, err := service.UpdateNamespacePipelineByID(
		ctx,
		fakeNamespace(),
		"pipelineID",
		&pbPipeline,
	)

	c.Assert(err, quicktest.IsNil)
	c.Assert(updatedPbPipeline, quicktest.IsNotNil)
}

type serviceConfig struct {
	repository                   repository.Repository
	redisClient                  *redis.Client
	temporalClient               client.Client
	aCLClient                    acl.ACLClientInterface
	converter                    Converter
	mgmtPublicServiceClient      mgmtpb.MgmtPublicServiceClient
	mgmtPrivateServiceClient     mgmtpb.MgmtPrivateServiceClient
	minioClient                  minio.Client
	componentStore               *componentstore.Store
	memory                       memory.Store
	workerUID                    uuid.UUID
	retentionHandler             MetadataRetentionHandler
	binaryFetcher                external.BinaryFetcher
	artifactPublicServiceClient  artifactpb.ArtifactPublicServiceClient
	artifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
}

// newService is a compact helper to instantiate a new service, which allows us
// to only define the dependencies we'll use in a test. This approach shouldn't
// be used in production code, where we want every dependency to be injected
// every time, so we rely on the compiler to guard us against missing
// dependency injections.
func newService(cfg serviceConfig) Service {
	return NewService(
		cfg.repository,
		cfg.redisClient,
		cfg.temporalClient,
		cfg.aCLClient,
		cfg.converter,
		cfg.mgmtPublicServiceClient,
		cfg.mgmtPrivateServiceClient,
		cfg.minioClient,
		cfg.componentStore,
		cfg.memory,
		cfg.workerUID,
		cfg.retentionHandler,
		cfg.binaryFetcher,
		cfg.artifactPublicServiceClient,
		cfg.artifactPrivateServiceClient,
	)
}
