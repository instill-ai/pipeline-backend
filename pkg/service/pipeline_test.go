package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	modelPB "github.com/instill-ai/protogen-go/model"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline"
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

		pipelineService := PipelineService{
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

		pipelineService := PipelineService{
			PipelineRepository: mockPipelineRepository,
			ModelServiceClient: rpcModelClient,
		}

		_, err := pipelineService.UpdatePipeline(normalPipeline)

		assert.NoError(t, err)
	})
}

func TestPipelineService_TriggerPipeline(t *testing.T) {
	t.Run("normal-url", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		var recipeModels []*model.Model
		recipeModels = append(recipeModels, &model.Model{
			Name:    "yolov4",
			Version: 1,
		})
		normalPipeline := model.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			Namespace:   NAMESPACE,
			Recipe: &model.Recipe{
				Source: &model.Source{
					Type: "direct",
				},
				Model: recipeModels,
				Destination: &model.Destination{
					Type: "direct",
				},
			},
		}

		mockPipelineRepository := NewMockOperations(ctrl)
		rpcModelClient := NewMockModelClient(ctrl)

		var triggerContents []*pipelinePB.TriggerPipelineContent
		triggerContents = append(triggerContents, &pipelinePB.TriggerPipelineContent{
			Url: "https://artifacts.instill.tech/dog.jpg",
		})
		var modelContents []*modelPB.ImageRequest
		modelContents = append(modelContents, &modelPB.ImageRequest{
			Url: "https://artifacts.instill.tech/dog.jpg",
		})

		rpcModelClient.
			EXPECT().
			PredictModel(gomock.Any(), gomock.Eq(&modelPB.PredictModelImageRequest{
				Name:     "yolov4",
				Version:  1,
				Contents: modelContents,
			}))

		pipelineService := PipelineService{
			PipelineRepository: mockPipelineRepository,
			ModelServiceClient: rpcModelClient,
		}

		_, err := pipelineService.TriggerPipeline(NAMESPACE, &pipelinePB.TriggerPipelineRequest{Contents: triggerContents}, normalPipeline)

		assert.NoError(t, err)
	})
}
