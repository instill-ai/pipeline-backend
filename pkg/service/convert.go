package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) recipeNameToPermalink(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) (*datamodel.Recipe, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = InjectOwnerToContext(ctx, owner)
	recipePermalink := &datamodel.Recipe{Version: recipeRscName.Version}
	for _, component := range recipeRscName.Components {
		componentPermalink := &datamodel.Component{
			Id:             component.Id,
			Metadata:       component.Metadata,
			Dependencies:   component.Dependencies,
			ResourceDetail: component.ResourceDetail,
		}

		permalink := ""
		var err error
		switch GetDefinitionType(component) {
		case SourceConnector:
			permalink, err = s.sourceConnectorNameToPermalink(ctx, component.ResourceName)
		case DestinationConnector:
			permalink, err = s.destinationConnectorNameToPermalink(ctx, component.ResourceName)
		case Model:
			permalink, err = s.modelNameToPermalink(ctx, component.ResourceName)
		}
		if err != nil {
			return nil, err
		}
		componentPermalink.ResourceName = permalink
		recipePermalink.Components = append(recipePermalink.Components, componentPermalink)
	}
	return recipePermalink, nil
}

func (s *service) recipePermalinkToName(recipePermalink *datamodel.Recipe) (*datamodel.Recipe, error) {

	recipe := &datamodel.Recipe{Version: recipePermalink.Version}

	for _, componentPermalink := range recipePermalink.Components {
		component := &datamodel.Component{
			Id:             componentPermalink.Id,
			Metadata:       componentPermalink.Metadata,
			Dependencies:   componentPermalink.Dependencies,
			ResourceDetail: componentPermalink.ResourceDetail,
		}

		name := ""
		var err error
		switch GetDefinitionType(componentPermalink) {
		case SourceConnector:
			name, err = s.sourceConnectorPermalinkToName(componentPermalink.ResourceName)
		case DestinationConnector:
			name, err = s.destinationConnectorPermalinkToName(componentPermalink.ResourceName)
		case Model:
			name, err = s.modelPermalinkToName(componentPermalink.ResourceName)
		}
		if err != nil {
			return nil, err
		}
		component.ResourceName = name
		recipe.Components = append(recipe.Components, component)
	}
	return recipe, nil
}

func (s *service) sourceConnectorNameToPermalink(ctx context.Context, name string) (string, error) {

	getSrcConnResp, err := s.connectorPublicServiceClient.GetSourceConnector(ctx,
		&connectorPB.GetSourceConnectorRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetSourceConnector", name, err)
	}

	srcColID, err := resource.GetCollectionID(name)
	if err != nil {
		return "", err
	}

	return srcColID + "/" + getSrcConnResp.GetSourceConnector().GetUid(), nil
}

func (s *service) destinationConnectorNameToPermalink(ctx context.Context, name string) (string, error) {

	getDstConnResp, err := s.connectorPublicServiceClient.GetDestinationConnector(ctx,
		&connectorPB.GetDestinationConnectorRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetDestinationConnector", name, err)
	}

	destColID, err := resource.GetCollectionID(name)
	if err != nil {
		return "", err
	}

	return destColID + "/" + getDstConnResp.GetDestinationConnector().GetUid(), nil
}

func (s *service) modelNameToPermalink(ctx context.Context, name string) (string, error) {

	getModelResp, err := s.modelPublicServiceClient.GetModel(ctx,
		&modelPB.GetModelRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[model-backend] Error %s at %s: %s", "GetModel", name, err)
	}

	modelColID, err := resource.GetCollectionID(name)
	if err != nil {
		return "", err
	}

	return modelColID + "/" + getModelResp.GetModel().GetUid(), nil
}

func (s *service) sourceConnectorPermalinkToName(permalink string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lookUpSrcConnResp, err := s.connectorPrivateServiceClient.LookUpSourceConnectorAdmin(ctx,
		&connectorPB.LookUpSourceConnectorAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpSourceConnectorAdmin", permalink, err)
	}

	srcColID, err := resource.GetCollectionID(permalink)
	if err != nil {
		return "", err
	}

	return srcColID + "/" + lookUpSrcConnResp.GetSourceConnector().GetId(), nil
}

func (s *service) destinationConnectorPermalinkToName(permalink string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lookUpDstConnResp, err := s.connectorPrivateServiceClient.LookUpDestinationConnectorAdmin(ctx,
		&connectorPB.LookUpDestinationConnectorAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpDestinationConnectorAdmin", permalink, err)
	}

	dstColID, err := resource.GetCollectionID(permalink)
	if err != nil {
		return "", err
	}

	return dstColID + "/" + lookUpDstConnResp.GetDestinationConnector().GetId(), nil
}

func (s *service) modelPermalinkToName(permalink string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lookUpModelResp, err := s.modelPrivateServiceClient.LookUpModelAdmin(ctx,
		&modelPB.LookUpModelAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[model-backend] Error %s at %s: %s", "LookUpModelAdmin", permalink, err)
	}

	modelColID, err := resource.GetCollectionID(permalink)
	if err != nil {
		return "", err
	}

	return modelColID + "/" + lookUpModelResp.Model.GetId(), nil
}

func (s *service) excludeResourceDetailFromRecipe(recipe *datamodel.Recipe) error {

	for idx := range recipe.Components {
		recipe.Components[idx].ResourceDetail = &structpb.Struct{}
	}
	return nil
}

func (s *service) includeResourceDetailInRecipe(recipe *datamodel.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {
		switch GetDefinitionType(recipe.Components[idx]) {
		case SourceConnector:
			resp, err := s.connectorPrivateServiceClient.LookUpSourceConnectorAdmin(ctx, &connectorPB.LookUpSourceConnectorAdminRequest{
				Permalink: recipe.Components[idx].ResourceName,
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetSourceConnector", recipe.Components[idx].ResourceName, err)
			}

			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetSourceConnector())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			defResp, err := s.connectorPublicServiceClient.GetSourceConnectorDefinition(ctx, &connectorPB.GetSourceConnectorDefinitionRequest{
				Name: resp.GetSourceConnector().GetSourceConnectorDefinition(),
			})
			if err != nil {
				return err
			}
			defDetail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			defJson, defMarshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(defResp.GetSourceConnectorDefinition())
			if defMarshalErr != nil {
				return defMarshalErr
			}
			defUnmarshalErr := defDetail.UnmarshalJSON(defJson)
			if defUnmarshalErr != nil {
				return defUnmarshalErr
			}
			detail.GetFields()["source_connector_definition_detail"] = structpb.NewStructValue(defDetail)
			recipe.Components[idx].ResourceDetail = detail

		case DestinationConnector:
			resp, err := s.connectorPrivateServiceClient.LookUpDestinationConnectorAdmin(ctx, &connectorPB.LookUpDestinationConnectorAdminRequest{
				Permalink: recipe.Components[idx].ResourceName,
			})
			if err != nil {
				return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetDestinationConnector", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetDestinationConnector())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			defResp, err := s.connectorPublicServiceClient.GetDestinationConnectorDefinition(ctx, &connectorPB.GetDestinationConnectorDefinitionRequest{
				Name: resp.GetDestinationConnector().GetDestinationConnectorDefinition(),
			})
			if err != nil {
				return err
			}
			defDetail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			defJson, defMarshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(defResp.GetDestinationConnectorDefinition())
			if defMarshalErr != nil {
				return defMarshalErr
			}
			defUnmarshalErr := defDetail.UnmarshalJSON(defJson)
			if defUnmarshalErr != nil {
				return defUnmarshalErr
			}
			detail.GetFields()["destination_connector_definition_detail"] = structpb.NewStructValue(defDetail)
			recipe.Components[idx].ResourceDetail = detail

		case Model:
			resp, err := s.modelPrivateServiceClient.LookUpModelAdmin(ctx, &modelPB.LookUpModelAdminRequest{
				Permalink: recipe.Components[idx].ResourceName,
			})
			if err != nil {
				return fmt.Errorf("[model-backend] Error %s at %s: %s", "GetModel", recipe.Components[idx].ResourceName, err)
			}
			detail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetModel())
			if marshalErr != nil {
				return marshalErr
			}
			unmarshalErr := detail.UnmarshalJSON(json)
			if unmarshalErr != nil {
				return unmarshalErr
			}

			defResp, err := s.modelPublicServiceClient.GetModelDefinition(ctx, &modelPB.GetModelDefinitionRequest{
				Name: resp.Model.GetModelDefinition(),
			})
			if err != nil {
				return err
			}
			defDetail := &structpb.Struct{}
			// Note: need to deal with camelCase or under_score for grpc in future
			defJson, defMarshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(defResp.GetModelDefinition())
			if defMarshalErr != nil {
				return defMarshalErr
			}
			defUnmarshalErr := defDetail.UnmarshalJSON(defJson)
			if defUnmarshalErr != nil {
				return defUnmarshalErr
			}
			detail.GetFields()["model_definition_detail"] = structpb.NewStructValue(defDetail)
			recipe.Components[idx].ResourceDetail = detail
		}

	}
	return nil
}

func cvtModelTaskOutputToPipelineTaskOutput(modelTaskOutputs []*modelPB.TaskOutput) []*pipelinePB.TaskOutput {

	logger, _ := logger.GetZapLogger()

	var pipelineTaskOutputs []*pipelinePB.TaskOutput
	for _, taskOutput := range modelTaskOutputs {
		switch v := taskOutput.Output.(type) {
		case *modelPB.TaskOutput_Classification:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Classification{
					Classification: proto.Clone(v.Classification).(*modelPB.ClassificationOutput),
				},
			})
		case *modelPB.TaskOutput_Detection:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Detection{
					Detection: proto.Clone(v.Detection).(*modelPB.DetectionOutput),
				},
			})
		case *modelPB.TaskOutput_Keypoint:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Keypoint{
					Keypoint: proto.Clone(v.Keypoint).(*modelPB.KeypointOutput),
				},
			})
		case *modelPB.TaskOutput_Ocr:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Ocr{
					Ocr: proto.Clone(v.Ocr).(*modelPB.OcrOutput),
				},
			})
		case *modelPB.TaskOutput_InstanceSegmentation:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: proto.Clone(v.InstanceSegmentation).(*modelPB.InstanceSegmentationOutput),
				},
			})
		case *modelPB.TaskOutput_SemanticSegmentation:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: proto.Clone(v.SemanticSegmentation).(*modelPB.SemanticSegmentationOutput),
				},
			})
		case *modelPB.TaskOutput_TextToImage:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_TextToImage{
					TextToImage: proto.Clone(v.TextToImage).(*modelPB.TextToImageOutput),
				},
			})
		case *modelPB.TaskOutput_TextGeneration:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_TextGeneration{
					TextGeneration: proto.Clone(v.TextGeneration).(*modelPB.TextGenerationOutput),
				},
			})
		case *modelPB.TaskOutput_Unspecified:
			pipelineTaskOutputs = append(pipelineTaskOutputs, &pipelinePB.TaskOutput{
				Output: &pipelinePB.TaskOutput_Unspecified{
					Unspecified: proto.Clone(v.Unspecified).(*modelPB.UnspecifiedOutput),
				},
			})
		default:
			logger.Error("AI task type is not defined")
		}
	}

	return pipelineTaskOutputs
}

func ConvertResourceUIDToControllerResourcePermalink(resourceUID string, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID, resourceType)

	return resourcePermalink
}
