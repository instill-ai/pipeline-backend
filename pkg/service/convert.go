package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func (s *service) recipeNameToPermalink(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) (*datamodel.Recipe, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = utils.InjectOwnerToContext(ctx, owner)
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
		switch utils.GetDefinitionType(component) {
		case utils.SourceConnector:
			permalink, err = s.sourceConnectorNameToPermalink(ctx, component.ResourceName)
		case utils.DestinationConnector:
			permalink, err = s.destinationConnectorNameToPermalink(ctx, component.ResourceName)
		case utils.Model:
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
		switch utils.GetDefinitionType(componentPermalink) {
		case utils.SourceConnector:
			name, err = s.sourceConnectorPermalinkToName(componentPermalink.ResourceName)
			if err != nil {
				return nil, err
			}
		case utils.DestinationConnector:
			name, err = s.destinationConnectorPermalinkToName(componentPermalink.ResourceName)
			if err != nil {
				return nil, err
			}
		case utils.Model:
			name, err = s.modelPermalinkToName(componentPermalink.ResourceName)
			if err != nil {
				// NOTE: this is a workaround solution, need to handle the broken recipe more properly
				name = "models/_error_"
			}
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
	logger, _ := logger.GetZapLogger(ctx)

	for idx := range recipe.Components {
		switch utils.GetDefinitionType(recipe.Components[idx]) {
		case utils.SourceConnector:
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

		case utils.DestinationConnector:
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

		case utils.Model:
			resp, err := s.modelPrivateServiceClient.LookUpModelAdmin(ctx, &modelPB.LookUpModelAdminRequest{
				Permalink: recipe.Components[idx].ResourceName,
			})
			if err != nil {
				// NOTE: this is a workaround solution, need to handle the broken recipe more properly
				logger.Warn(fmt.Sprintf("[model-backend] Error %s at %s: %s", "GetModel", recipe.Components[idx].ResourceName, err))
				d, _ := structpb.NewValue(map[string]interface{}{
					"id":    "_error_",
					"name":  "models/_error_",
					"state": "STATE_ERROR",
					"task":  "TASK_UNSPECIFIED",
				})

				recipe.Components[idx].ResourceDetail = d.GetStructValue()
				continue
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

func ConvertResourceUIDToControllerResourcePermalink(resourceUID string, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID, resourceType)

	return resourcePermalink
}
