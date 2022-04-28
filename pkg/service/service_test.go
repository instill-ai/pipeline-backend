package service_test

//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/pipeline-backend/pkg/repository Repository
//go:generate mockgen -destination mock_model_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/model/v1alpha ModelServiceClient

import (
	"testing"

	uuid "github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

var OwnerID = uuid.UUID{}

func TestCreatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			OwnerID:     OwnerID,
			Recipe:      &datamodel.Recipe{},
		}

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetPipeline(gomock.Eq(normalPipeline.OwnerID), gomock.Eq(normalPipeline.Name)).
			Return(&normalPipeline, nil).
			Times(1)
		mockRepository.
			EXPECT().
			CreatePipeline(gomock.Eq(&normalPipeline)).
			Return(nil)

		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		s := service.NewService(mockRepository, mockModelServiceClient)

		_, err := s.CreatePipeline(&normalPipeline)

		assert.NoError(t, err)
	})
}

func TestUpdatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			OwnerID:     OwnerID,
		}
		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetPipeline(gomock.Eq(OwnerID), gomock.Eq(normalPipeline.Name)).
			Return(&normalPipeline, nil).
			Times(2)
		mockRepository.
			EXPECT().
			UpdatePipeline(gomock.Eq(OwnerID), gomock.Eq(normalPipeline.Name), gomock.Eq(&normalPipeline)).
			Return(nil)

		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		s := service.NewService(mockRepository, mockModelServiceClient)

		_, err := s.UpdatePipeline(OwnerID, normalPipeline.Name, &normalPipeline)

		assert.NoError(t, err)
	})
}

func TestTriggerPipeline(t *testing.T) {
	t.Run("normal-url", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		var recipeModels []*datamodel.Model
		recipeModels = append(recipeModels, &datamodel.Model{
			ModelName:    "yolov4",
			InstanceName: "latest",
		})

		normalPipeline := datamodel.Pipeline{
			Name:        "awesome",
			Description: "awesome pipeline",
			OwnerID:     OwnerID,
			Recipe: &datamodel.Recipe{
				Source: &datamodel.Source{
					Name: "HTTP",
				},
				Models: recipeModels,
				Destination: &datamodel.Destination{
					Name: "HTTP",
				},
			},
		}

		var modelInputs []*modelPB.Input
		modelInputs = append(modelInputs, &modelPB.Input{
			Type: &modelPB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/dog.jpg"},
		})

		mockRepository := NewMockRepository(ctrl)
		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		mockModelServiceClient.EXPECT().TriggerModel(gomock.Any(), gomock.Eq(&modelPB.TriggerModelRequest{
			ModelName:    "yolov4",
			InstanceName: "latest",
			Inputs:       modelInputs,
		}))

		var pipelineInputs []*pipelinePB.Input
		pipelineInputs = append(pipelineInputs, &pipelinePB.Input{
			Type: &pipelinePB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/dog.jpg"},
		})

		s := service.NewService(mockRepository, mockModelServiceClient)

		_, err := s.TriggerPipeline(OwnerID, &pipelinePB.TriggerPipelineRequest{Inputs: pipelineInputs}, &normalPipeline)

		assert.NoError(t, err)
	})
}
