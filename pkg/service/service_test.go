package service_test

//go:generate mockgen -destination mock_repository_test.go -package $GOPACKAGE github.com/instill-ai/pipeline-backend/pkg/repository Repository
//go:generate mockgen -destination mock_model_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/vdp/model/v1alpha ModelPublicServiceClient
//go:generate mockgen -destination mock_connector_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/vdp/connector/v1alpha ConnectorPublicServiceClient
//go:generate mockgen -destination mock_user_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha MgmtPrivateServiceClient
//go:generate mockgen -destination mock_usage_grpc_test.go -package $GOPACKAGE github.com/instill-ai/protogen-go/vdp/usage/v1alpha UsageServiceClient

import (
	"database/sql"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
)

func TestCreatePipeline(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		uid := uuid.New()
		uidstr := uid.String()
		owner := mgmtPB.User{
			Name: "users/local-user",
			Uid:  &uidstr,
		}

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

		mockMgmtPrivateServiceClient := NewMockMgmtPrivateServiceClient(ctrl)

		mockConnectorPublicServiceClient := NewMockConnectorPublicServiceClient(ctrl)
		mockConnectorPublicServiceClient.EXPECT().GetSourceConnectorDefinition(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockConnectorPublicServiceClient.EXPECT().GetDestinationConnectorDefinition(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
		mockConnectorPublicServiceClient.EXPECT().GetSourceConnector(gomock.Any(), gomock.Any()).Return(nil, nil).Times(3)
		mockConnectorPublicServiceClient.EXPECT().GetDestinationConnector(gomock.Any(), gomock.Any()).Return(nil, nil).Times(3)

		mockModelPublicServiceClient := NewMockModelPublicServiceClient(ctrl)

		s := service.NewService(mockRepository, mockMgmtPrivateServiceClient, mockConnectorPublicServiceClient, mockModelPublicServiceClient, nil)

		_, err := s.CreatePipeline(&owner, &normalPipeline)

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
// 			Type: &pipelinePB.Input_ImageUrl{ImageUrl: "https://artifacts.instill.tech/imgs/dog.jpg"},
// 		})

// 		s := service.NewService(mockRepository, mockConnectorServiceClient, mockModelServiceClient)

// 		_, err := s.TriggerPipeline(&pipelinePB.TriggerPipelineRequest{Inputs: pipelineInputs}, &normalPipeline)

// 		assert.NoError(t, err)
// 	})
// }
