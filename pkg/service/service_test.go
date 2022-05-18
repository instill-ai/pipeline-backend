package service_test

//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/pipeline-backend/pkg/repository Repository
//go:generate mockgen -destination mock_model_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/model/v1alpha ModelServiceClient
//go:generate mockgen -destination mock_connector_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/connector/v1alpha ConnectorServiceClient
//go:generate mockgen -destination mock_user_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/mgmt/v1alpha UserServiceClient

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	mgmtPB "github.com/instill-ai/protogen-go/mgmt/v1alpha"
)

func TestCreatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		normalPipeline := datamodel.Pipeline{
			ID:    "awesome",
			Owner: "users/local-user",

			Recipe: &datamodel.Recipe{
				Source:      "source-connectors/source-http",
				Destination: "destination-connectors/destination-http",
			},

			Description: sql.NullString{
				String: "awesome pipeline",
				Valid:  true,
			},
		}

		mockRepository := NewMockRepository(ctrl)
		mockRepository.
			EXPECT().
			GetPipelineByID(gomock.Eq(normalPipeline.ID), gomock.Any(), false).
			Return(&normalPipeline, nil).
			Times(1)
		mockRepository.
			EXPECT().
			CreatePipeline(gomock.Eq(&normalPipeline)).
			Return(nil)

		mockUserServiceClient := NewMockUserServiceClient(ctrl)
		mockUserServiceClient.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&mgmtPB.GetUserResponse{}, nil).Times(2)
		mockUserServiceClient.EXPECT().LookUpUser(gomock.Any(), gomock.Any()).Return(&mgmtPB.LookUpUserResponse{}, nil).Times(1)

		mockConnectorServiceClient := NewMockConnectorServiceClient(ctrl)
		mockConnectorServiceClient.EXPECT().GetSourceConnectorDefinition(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockConnectorServiceClient.EXPECT().GetDestinationConnectorDefinition(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockConnectorServiceClient.EXPECT().GetSourceConnector(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
		mockConnectorServiceClient.EXPECT().GetDestinationConnector(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)
		mockConnectorServiceClient.EXPECT().LookUpSourceConnector(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockConnectorServiceClient.EXPECT().LookUpDestinationConnector(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

		mockModelServiceClient := NewMockModelServiceClient(ctrl)

		s := service.NewService(mockRepository, mockUserServiceClient, mockConnectorServiceClient, mockModelServiceClient)

		_, err := s.CreatePipeline(&normalPipeline)

		assert.NoError(t, err)
	})
}

// func TestUpdatePipeline(t *testing.T) {
// 	t.Run("normal", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		normalPipeline := datamodel.Pipeline{
// 			ID:    "awesome",
// 			Owner: Owner,

// 			Description: sql.NullString{
// 				String: "awesome pipeline",
// 				Valid:  true,
// 			},
// 		}
// 		mockRepository := NewMockRepository(ctrl)
// 		mockRepository.
// 			EXPECT().
// 			GetPipelineByID(gomock.Eq(ID), gomock.Eq(normalPipeline.Owner), false).
// 			Return(&normalPipeline, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			GetPipelineByID(gomock.Eq(normalPipeline.ID), gomock.Eq(normalPipeline.Owner), false).
// 			Return(&normalPipeline, nil).
// 			Times(1)
// 		mockRepository.
// 			EXPECT().
// 			UpdatePipeline(gomock.Eq(ID), gomock.Eq(Owner), gomock.Eq(&normalPipeline)).
// 			Return(nil)

// 		mockModelServiceClient := NewMockModelServiceClient(ctrl)
// 		mockConnectorServiceClient := NewMockConnectorServiceClient(ctrl)

// 		s := service.NewService(mockRepository, mockConnectorServiceClient, mockModelServiceClient)

// 		_, err := s.UpdatePipeline(ID, Owner, &normalPipeline)

// 		assert.NoError(t, err)
// 	})
// }

// func TestTriggerPipeline(t *testing.T) {
// 	t.Run("normal-url", func(t *testing.T) {
// 		ctrl := gomock.NewController(t)

// 		normalPipeline := datamodel.Pipeline{
// 			ID:    "awesome",
// 			Owner: Owner,

// 			Description: sql.NullString{
// 				String: "awesome pipeline",
// 				Valid:  true,
// 			},

// 			Recipe: &datamodel.Recipe{
// 				Source:         "source-connectors/source-http",
// 				ModelInstances: []string{"models/yolov4/instances/latest"},
// 				Destination:    "destination-connectors/destination-http",
// 			},
// 		}

// 		mockRepository := NewMockRepository(ctrl)
// 		mockModelServiceClient := NewMockModelServiceClient(ctrl)
// 		mockConnectorServiceClient := NewMockConnectorServiceClient(ctrl)

// 		var pipelineInputs []*pipelinePB.Input
// 		pipelineInputs = append(pipelineInputs, &pipelinePB.Input{
// 			Type: &pipelinePB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/dog.jpg"},
// 		})

// 		s := service.NewService(mockRepository, mockConnectorServiceClient, mockModelServiceClient)

// 		_, err := s.TriggerPipeline(&pipelinePB.TriggerPipelineRequest{Inputs: pipelineInputs}, &normalPipeline)

// 		assert.NoError(t, err)
// 	})
// }
