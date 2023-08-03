package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
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
		if componentPermalink.Dependencies != nil {
			if _, ok := componentPermalink.Dependencies["audios"]; !ok {
				componentPermalink.Dependencies["audios"] = "[]"
			}
		}

		permalink := ""
		var err error
		permalink, err = s.connectorNameToPermalink(ctx, component.ResourceName)
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
		if component.Dependencies != nil {
			if _, ok := component.Dependencies["audios"]; !ok {
				component.Dependencies["audios"] = "[]"
			}
		}

		name, err := s.connectorPermalinkToName(componentPermalink.ResourceName)
		if err != nil {
			return nil, err
		}
		component.ResourceName = name
		recipe.Components = append(recipe.Components, component)
	}
	return recipe, nil
}

func (s *service) connectorNameToPermalink(ctx context.Context, name string) (string, error) {

	getSrcConnResp, err := s.connectorPublicServiceClient.GetConnector(ctx,
		&connectorPB.GetConnectorRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", name, err)
	}

	srcColID, err := resource.GetCollectionID(name)
	if err != nil {
		return "", err
	}

	return srcColID + "/" + getSrcConnResp.GetConnector().GetUid(), nil
}

func (s *service) connectorPermalinkToName(permalink string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lookUpSrcConnResp, err := s.connectorPrivateServiceClient.LookUpConnectorAdmin(ctx,
		&connectorPB.LookUpConnectorAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpConnectorAdmin", permalink, err)
	}

	srcColID, err := resource.GetCollectionID(permalink)
	if err != nil {
		return "", err
	}

	return srcColID + "/" + lookUpSrcConnResp.GetConnector().GetId(), nil
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

		resp, err := s.connectorPrivateServiceClient.LookUpConnectorAdmin(ctx, &connectorPB.LookUpConnectorAdminRequest{
			Permalink: recipe.Components[idx].ResourceName,
			View:      connectorPB.View_VIEW_FULL.Enum(),
		})
		if err != nil {
			return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
		}
		detail := &structpb.Struct{}
		// Note: need to deal with camelCase or under_score for grpc in future
		json, marshalErr := protojson.MarshalOptions{UseProtoNames: true}.Marshal(resp.GetConnector())
		if marshalErr != nil {
			return marshalErr
		}
		unmarshalErr := detail.UnmarshalJSON(json)
		if unmarshalErr != nil {
			return unmarshalErr
		}

		recipe.Components[idx].ResourceDetail = detail

	}
	return nil
}

func (s *service) IncludeConnectorTypeInRecipeByPermalink(recipe *datamodel.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		resp, err := s.connectorPrivateServiceClient.LookUpConnectorAdmin(ctx, &connectorPB.LookUpConnectorAdminRequest{
			Permalink: recipe.Components[idx].ResourceName,
		})
		if err != nil {
			return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
		}

		recipe.Components[idx].Type = resp.Connector.ConnectorType.String()

	}
	return nil
}

func (s *service) IncludeConnectorTypeInRecipeByName(recipe *datamodel.Recipe, owner *mgmtPB.User) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = utils.InjectOwnerToContext(ctx, owner)
	for idx := range recipe.Components {

		resp, err := s.connectorPublicServiceClient.GetConnector(ctx, &connectorPB.GetConnectorRequest{
			Name: recipe.Components[idx].ResourceName,
		})
		if err != nil {
			return fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnector", recipe.Components[idx].ResourceName, err)
		}

		recipe.Components[idx].Type = resp.Connector.ConnectorType.String()

	}
	return nil
}

func ConvertResourceUIDToControllerResourcePermalink(resourceUID string, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID, resourceType)

	return resourcePermalink
}
