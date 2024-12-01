package service

import (
	"context"
	"testing"

	"github.com/frankban/quicktest"
	"github.com/go-redis/redismock/v9"
	"github.com/gofrs/uuid"
	"github.com/gojuno/minimock/v3"
	"go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/mock"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
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

	compStore := componentstore.Init(nil, config.Config.Component.Secrets, nil, external.NewBinaryFetcher())

	workerUID, _ := uuid.NewV4()
	service := NewService(
		ServiceConfig{
			Repository:               repo,
			RedisClient:              redisClient,
			TemporalClient:           temporalClient,
			ACLClient:                aclClient,
			Converter:                converter,
			MgmtPrivateServiceClient: mgmtPrivateClient,
			MinioClient:              nil,
			ComponentStore:           compStore,
			Memory:                   memory.NewMemoryStore(),
			WorkerUID:                workerUID,
			RetentionHandler:         nil,
		},
	)

	aclClient.CheckPermissionMock.Return(true, nil)
	uid, err := uuid.NewV4()
	c.Assert(err, quicktest.IsNil)
	dataPipeline := newDataPipeline(uid, [2]string{"tag1", "tag3"})
	newDataPipeline := newDataPipeline(uid, [2]string{"tag1", "tag2"})

	repo.GetNamespacePipelineByIDMock.Return(&dataPipeline, nil)
	repo.UpdateNamespacePipelineByUIDMock.Return(nil)
	repo.DeletePipelineTagsMock.Expect(ctx, uid, []string{"tag3"}).Return(nil)
	repo.CreatePipelineTagsMock.Expect(ctx, uid, []string{"tag2"}).Return(nil)
	repo.ListPipelineRunOnsMock.Expect(ctx, uid).Return(repository.PipelineRunOnList{}, nil)
	repo.DeletePipelineRunOnMock.Expect(ctx, uid).Return(nil)

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
