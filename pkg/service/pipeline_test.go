package service

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

const NAMESPACE = "local-user"

func TestPipelineService_CreatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			Namespace:   NAMESPACE,
		}
		mockPipelineRepository := NewMockOperations(ctrl)
		mockPipelineRepository.
			EXPECT().
			GetPipelineByName(gomock.Eq(NAMESPACE), gomock.Eq(normalPipeline.Name)).
			Return(datamodel.Pipeline{}, nil).
			Times(2)
		mockPipelineRepository.
			EXPECT().
			CreatePipeline(normalPipeline).
			Return(nil)

		rpcModelClient := NewMockModelClient(ctrl)

		pipelineService := pipelineService{
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

		normalPipeline := datamodel.Pipeline{
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

		pipelineService := pipelineService{
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

		var recipeModels []*datamodel.Model
		recipeModels = append(recipeModels, &datamodel.Model{
			Name:    "yolov4",
			Version: 1,
		})

		normalPipeline := datamodel.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			Namespace:   NAMESPACE,
			Recipe: &datamodel.Recipe{
				Source: &datamodel.Source{
					Type: "direct",
				},
				Model: recipeModels,
				Destination: &datamodel.Destination{
					Type: "direct",
				},
			},
		}

		var modelInputs []*modelPB.Input
		modelInputs = append(modelInputs, &modelPB.Input{
			Type: &modelPB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/dog.jpg"},
		})

		mockPipelineRepository := NewMockOperations(ctrl)
		rpcModelClient := NewMockModelClient(ctrl)

		rpcModelClient.EXPECT().TriggerModel(gomock.Any(), gomock.Eq(&modelPB.TriggerModelRequest{
			Name:    "yolov4",
			Version: 1,
			Inputs:  modelInputs,
		}))

		var pipelineInputs []*pipelinePB.Input
		pipelineInputs = append(pipelineInputs, &pipelinePB.Input{
			Type: &pipelinePB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/dog.jpg"},
		})

		pipelineService := pipelineService{
			PipelineRepository: mockPipelineRepository,
			ModelServiceClient: rpcModelClient,
		}

		_, err := pipelineService.TriggerPipeline(NAMESPACE, &pipelinePB.TriggerPipelineRequest{Inputs: pipelineInputs}, normalPipeline)

		assert.NoError(t, err)
	})
}
