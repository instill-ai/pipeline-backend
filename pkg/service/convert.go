package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func IsConnector(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connectors/")
}

func IsConnectorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "connector-definitions/")
}

func IsOperatorDefinition(resourceName string) bool {
	return strings.HasPrefix(resourceName, "operator-definitions/")
}

func (s *service) recipeNameToPermalink(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) (*datamodel.Recipe, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = utils.InjectOwnerToContext(ctx, owner)
	recipePermalink := &datamodel.Recipe{Version: recipeRscName.Version}
	for _, component := range recipeRscName.Components {
		componentPermalink := &datamodel.Component{
			Id:               component.Id,
			Configuration:    component.Configuration,
			ResourceDetail:   component.ResourceDetail,
			DefinitionDetail: component.DefinitionDetail,
		}

		permalink := ""
		var err error
		if IsConnector(component.ResourceName) {
			permalink, err = s.connectorNameToPermalink(ctx, component.ResourceName)
			if err != nil {
				return nil, err
			}
			componentPermalink.ResourceName = permalink
		}
		if IsConnectorDefinition(component.DefinitionName) {
			permalink, err = s.connectorDefinitionNameToPermalink(ctx, component.DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.DefinitionName = permalink
		} else if IsOperatorDefinition(component.DefinitionName) {
			permalink, err = s.operatorDefinitionNameToPermalink(ctx, component.DefinitionName)
			if err != nil {
				return nil, err
			}
			componentPermalink.DefinitionName = permalink
		}

		recipePermalink.Components = append(recipePermalink.Components, componentPermalink)
	}
	return recipePermalink, nil
}

func (s *service) recipePermalinkToName(recipePermalink *datamodel.Recipe) (*datamodel.Recipe, error) {

	recipe := &datamodel.Recipe{Version: recipePermalink.Version}

	for _, componentPermalink := range recipePermalink.Components {
		component := &datamodel.Component{
			Id:               componentPermalink.Id,
			Configuration:    componentPermalink.Configuration,
			ResourceDetail:   componentPermalink.ResourceDetail,
			DefinitionDetail: componentPermalink.DefinitionDetail,
		}

		if IsConnector(componentPermalink.ResourceName) {
			name, err := s.connectorPermalinkToName(componentPermalink.ResourceName)
			if err != nil {
				return nil, err
			}
			component.ResourceName = name
		}
		if IsConnectorDefinition(componentPermalink.DefinitionName) {
			name, err := s.connectorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		} else if IsOperatorDefinition(componentPermalink.DefinitionName) {
			name, err := s.operatorDefinitionPermalinkToName(componentPermalink.DefinitionName)
			if err != nil {
				return nil, err
			}
			component.DefinitionName = name
		}

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

func (s *service) connectorDefinitionNameToPermalink(ctx context.Context, name string) (string, error) {

	resp, err := s.connectorPublicServiceClient.GetConnectorDefinition(ctx,
		&connectorPB.GetConnectorDefinitionRequest{
			Name: name,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "GetConnectorDefinition", name, err)
	}

	colId, err := resource.GetCollectionID(name)
	if err != nil {
		return "", err
	}

	return colId + "/" + resp.GetConnectorDefinition().GetUid(), nil
}

func (s *service) connectorDefinitionPermalinkToName(permalink string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := s.connectorPrivateServiceClient.LookUpConnectorDefinitionAdmin(ctx,
		&connectorPB.LookUpConnectorDefinitionAdminRequest{
			Permalink: permalink,
		})
	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpConnectorDefinitionAdmin", permalink, err)
	}

	colId, err := resource.GetCollectionID(permalink)
	if err != nil {
		return "", err
	}

	return colId + "/" + resp.GetConnectorDefinition().GetId(), nil
}

func (s *service) operatorDefinitionNameToPermalink(ctx context.Context, name string) (string, error) {
	id, err := resource.GetRscNameID(name)
	if err != nil {
		return "", err
	}
	def, err := s.operator.GetOperatorDefinitionById(id)
	if err != nil {
		return "", err
	}

	colId, err := resource.GetCollectionID(name)
	if err != nil {
		return "", err
	}

	return colId + "/" + def.Uid, nil
}

func (s *service) operatorDefinitionPermalinkToName(permalink string) (string, error) {
	uid, err := resource.GetPermalinkUID(permalink)
	if err != nil {
		return "", err
	}
	def, err := s.operator.GetOperatorDefinitionByUid(uuid.FromStringOrNil(uid))
	if err != nil {
		return "", err
	}

	if err != nil {
		return "", fmt.Errorf("[connector-backend] Error %s at %s: %s", "LookUpOperatorDefinitionAdmin", permalink, err)
	}

	colId, err := resource.GetCollectionID(permalink)
	if err != nil {
		return "", err
	}

	return colId + "/" + def.Id, nil
}

func (s *service) excludeDetailFromRecipe(recipe *datamodel.Recipe) error {

	for idx := range recipe.Components {
		recipe.Components[idx].ResourceDetail = &structpb.Struct{}
		recipe.Components[idx].DefinitionDetail = &structpb.Struct{}
	}
	return nil
}

func (s *service) includeDetailInRecipe(recipe *datamodel.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for idx := range recipe.Components {

		if IsConnector(recipe.Components[idx].ResourceName) {
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
		if IsConnectorDefinition(recipe.Components[idx].DefinitionName) {
			resp, err := s.connectorPrivateServiceClient.LookUpConnectorDefinitionAdmin(ctx, &connectorPB.LookUpConnectorDefinitionAdminRequest{
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

			recipe.Components[idx].DefinitionDetail = detail
		}
		if IsOperatorDefinition(recipe.Components[idx].DefinitionName) {
			uid, err := resource.GetPermalinkUID(recipe.Components[idx].DefinitionName)
			if err != nil {
				return err
			}
			def, err := s.operator.GetOperatorDefinitionByUid(uuid.FromStringOrNil(uid))
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

			recipe.Components[idx].DefinitionDetail = detail
		}

	}
	return nil
}

func ConvertResourceUIDToControllerResourcePermalink(resourceUID string, resourceType string) string {
	resourcePermalink := fmt.Sprintf("resources/%s/types/%s", resourceUID, resourceType)

	return resourcePermalink
}
