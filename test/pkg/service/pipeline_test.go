package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	service "github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/stretchr/testify/assert"
)

const NAMESPACE = "local-user"

func TestPipelineService_CreatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := model.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			Namespace:   NAMESPACE,
		}
		mockPipelineRepository := NewMockOperations(ctrl)
		mockPipelineRepository.
			EXPECT().
			GetPipelineByName(gomock.Eq(NAMESPACE), gomock.Eq(normalPipeline.Name)).
			Return(model.Pipeline{}, nil).
			Times(2)
		mockPipelineRepository.
			EXPECT().
			CreatePipeline(normalPipeline).
			Return(nil)

		rpcModelClient := NewMockModelClient(ctrl)

		pipelineService := service.PipelineService{
			PipelineRepository: mockPipelineRepository,
			ModelServiceClient: rpcModelClient,
		}

		_, err := pipelineService.CreatePipeline(normalPipeline)

		assert.NoError(t, err)
	})
}

func TestPipelineService_UpdatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := model.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			Namespace:   NAMESPACE,
		}
		mockPipelineRepository := NewMockOperations(ctrl)
		mockPipelineRepository.
			EXPECT().
			GetPipelineByName(gomock.Eq(NAMESPACE), gomock.Eq(normalPipeline.Name)).
			Return(normalPipeline, nil).
			Times(2)
		mockPipelineRepository.
			EXPECT().
			UpdatePipeline(gomock.Eq(normalPipeline)).
			Return(nil)

		rpcModelClient := NewMockModelClient(ctrl)

		pipelineService := service.PipelineService{
			PipelineRepository: mockPipelineRepository,
			ModelServiceClient: rpcModelClient,
		}

		_, err := pipelineService.UpdatePipeline(normalPipeline)

		assert.NoError(t, err)
	})
}
