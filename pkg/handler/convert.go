package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// PBToDBPipeline converts protobuf data model to db data model
func PBToDBPipeline(ctx context.Context, owner string, pbPipeline *pipelinePB.Pipeline) *datamodel.Pipeline {
	logger, _ := logger.GetZapLogger(ctx)

	return &datamodel.Pipeline{
		Owner: owner,
		ID:    pbPipeline.GetId(),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipeline.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipeline.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipeline.GetCreateTime() != nil {
					return pbPipeline.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipeline.GetUpdateTime() != nil {
					return pbPipeline.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipeline.GetDescription(),
			Valid:  true,
		},

		Recipe: func() *datamodel.Recipe {
			if pbPipeline.GetRecipe() != nil {
				b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.GetRecipe())
				if err != nil {
					logger.Error(err.Error())
				}

				recipe := datamodel.Recipe{}
				if err := json.Unmarshal(b, &recipe); err != nil {
					logger.Error(err.Error())
				}
				return &recipe
			}
			return nil
		}(),
	}
}

// DBToPBPipeline converts db data model to protobuf data model
func DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline) *pipelinePB.Pipeline {
	logger, _ := logger.GetZapLogger(ctx)

	pbPipeline := pipelinePB.Pipeline{
		Name:        fmt.Sprintf("pipelines/%s", dbPipeline.ID),
		Uid:         dbPipeline.BaseDynamic.UID.String(),
		Id:          dbPipeline.ID,
		CreateTime:  timestamppb.New(dbPipeline.CreateTime),
		UpdateTime:  timestamppb.New(dbPipeline.UpdateTime),
		Description: &dbPipeline.Description.String,

		Recipe: func() *pipelinePB.Recipe {
			if dbPipeline.Recipe != nil {
				b, err := json.Marshal(dbPipeline.Recipe)
				if err != nil {
					logger.Error(err.Error())
				}
				pbRecipe := pipelinePB.Recipe{}

				err = protojson.Unmarshal(b, &pbRecipe)
				if err != nil {
					logger.Error(err.Error())
				}

				for i := range pbRecipe.Components {
					// TODO: use enum
					if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions/") {
						if pbRecipe.Components[i].Resource != nil {
							switch pbRecipe.Components[i].Resource.ConnectorType {
							case connectorPB.ConnectorType_CONNECTOR_TYPE_AI:
								pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
							case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
								pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
							case connectorPB.ConnectorType_CONNECTOR_TYPE_DATA:
								pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
							}
						}
					} else if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions/") {
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
					}
				}

				return &pbRecipe
			}
			return nil
		}(),
	}

	if strings.HasPrefix(dbPipeline.Owner, "users/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_User{User: dbPipeline.Owner}
	} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_Org{Org: dbPipeline.Owner}
	}

	return &pbPipeline
}

// PBToDBPipelineRelease converts protobuf data model to db data model
func PBToDBPipelineRelease(ctx context.Context, owner string, pipelineUid uuid.UUID, pbPipelineRelease *pipelinePB.PipelineRelease) *datamodel.PipelineRelease {
	logger, _ := logger.GetZapLogger(ctx)

	return &datamodel.PipelineRelease{
		ID: pbPipelineRelease.GetId(),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipelineRelease.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipelineRelease.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipelineRelease.GetCreateTime() != nil {
					return pbPipelineRelease.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipelineRelease.GetUpdateTime() != nil {
					return pbPipelineRelease.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipelineRelease.GetDescription(),
			Valid:  true,
		},

		PipelineUID: pipelineUid,
	}
}

// DBToPBPipelineRelease converts db data model to protobuf data model
func DBToPBPipelineRelease(ctx context.Context, pipelineId string, dbPipelineRelease *datamodel.PipelineRelease) *pipelinePB.PipelineRelease {
	logger, _ := logger.GetZapLogger(ctx)

	pbPipelineRelease := pipelinePB.PipelineRelease{
		Name:        fmt.Sprintf("pipelines/%s/releases/%s", pipelineId, dbPipelineRelease.ID),
		Uid:         dbPipelineRelease.BaseDynamic.UID.String(),
		Id:          dbPipelineRelease.ID,
		CreateTime:  timestamppb.New(dbPipelineRelease.CreateTime),
		UpdateTime:  timestamppb.New(dbPipelineRelease.UpdateTime),
		Description: &dbPipelineRelease.Description.String,
		Recipe: func() *pipelinePB.Recipe {
			if dbPipelineRelease.Recipe != nil {
				b, err := json.Marshal(dbPipelineRelease.Recipe)
				if err != nil {
					logger.Error(err.Error())
				}
				pbRecipe := pipelinePB.Recipe{}

				err = protojson.Unmarshal(b, &pbRecipe)
				if err != nil {
					logger.Error(err.Error())
				}

				for i := range pbRecipe.Components {
					// TODO: use enum
					if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "connector-definitions/") {
						if pbRecipe.Components[i].Resource != nil {
							switch pbRecipe.Components[i].Resource.ConnectorType {
							case connectorPB.ConnectorType_CONNECTOR_TYPE_AI:
								pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI
							case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN:
								pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_BLOCKCHAIN
							case connectorPB.ConnectorType_CONNECTOR_TYPE_DATA:
								pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA
							}
						}
					} else if strings.HasPrefix(pbRecipe.Components[i].DefinitionName, "operator-definitions/") {
						pbRecipe.Components[i].Type = pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR
					}
				}

				return &pbRecipe
			}
			return nil
		}(),
	}

	return &pbPipelineRelease
}

func IncludeDetailInRecipeAdmin(recipe *pipelinePB.Recipe, s service.Service) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		if service.IsConnector(recipe.Components[idx].ResourceName) {
			resp, err := s.GetConnectorPrivateServiceClient().LookUpConnectorResourceAdmin(ctx, &connectorPB.LookUpConnectorResourceAdminRequest{
				Permalink: recipe.Components[idx].ResourceName,
				View:      connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnectorResource())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Resource = resp.ConnectorResource
		}
		if service.IsConnectorDefinition(recipe.Components[idx].DefinitionName) {
			resp, err := s.GetConnectorPrivateServiceClient().LookUpConnectorDefinitionAdmin(ctx, &connectorPB.LookUpConnectorDefinitionAdminRequest{
				Permalink: recipe.Components[idx].DefinitionName,
				View:      connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnectorDefinition())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_ConnectorDefinition{ConnectorDefinition: resp.ConnectorDefinition}
		}
		if service.IsOperatorDefinition(recipe.Components[idx].DefinitionName) {
			uid, err := resource.GetPermalinkUID(recipe.Components[idx].DefinitionName)
			if err != nil {
				return err
			}
			def, err := s.GetOperator().GetOperatorDefinitionByUid(uuid.FromStringOrNil(uid))
			if err != nil {
				return err
			}

			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(def)
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_OperatorDefinition{OperatorDefinition: def}
		}

	}
	return nil
}

func IncludeDetailInRecipe(recipe *pipelinePB.Recipe, s service.Service) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		if service.IsConnector(recipe.Components[idx].ResourceName) {
			resp, err := s.GetConnectorPublicServiceClient().GetConnectorResource(ctx, &connectorPB.GetConnectorResourceRequest{
				Name: recipe.Components[idx].ResourceName,
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnectorResource())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Resource = resp.ConnectorResource
		}
		if service.IsConnectorDefinition(recipe.Components[idx].DefinitionName) {
			resp, err := s.GetConnectorPublicServiceClient().GetConnectorDefinition(ctx, &connectorPB.GetConnectorDefinitionRequest{
				Name: recipe.Components[idx].DefinitionName,
				View: connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnectorDefinition())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_ConnectorDefinition{ConnectorDefinition: resp.ConnectorDefinition}
		}
		if service.IsOperatorDefinition(recipe.Components[idx].DefinitionName) {
			id, err := resource.GetRscNameID(recipe.Components[idx].DefinitionName)
			if err != nil {
				return err
			}
			def, err := s.GetOperator().GetOperatorDefinitionById(id)
			if err != nil {
				return err
			}

			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(def)
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			recipe.Components[idx].Definition = &pipelinePB.Component_OperatorDefinition{OperatorDefinition: def}
		}

	}
	return nil
}
