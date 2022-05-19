package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

func (s *service) ownerNameToPermalink(owner *string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if strings.Split(*owner, "/")[0] == "users" {
		user, err := s.userServiceClient.GetUser(ctx, &mgmtPB.GetUserRequest{Name: *owner})
		if err != nil {
			return fmt.Errorf("[mgmt-backend] %s", err)
		}
		*owner = "users/" + user.User.GetUid()
	} else if strings.Split(*owner, "/")[0] == "orgs" { //nolint
		// TODO: implement orgs case
	}

	return nil
}

func (s *service) ownerPermalinkToName(owner *string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if strings.Split(*owner, "/")[0] == "users" {
		user, err := s.userServiceClient.LookUpUser(ctx, &mgmtPB.LookUpUserRequest{Permalink: *owner})
		if err != nil {
			return fmt.Errorf("[mgmt-backend] %s", err)
		}
		*owner = "users/" + user.User.GetId()
	} else if strings.Split(*owner, "/")[0] == "orgs" { //nolint
		// TODO: implement orgs case
	}

	return nil
}

func (s *service) recipeNameToPermalink(recipe *datamodel.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Source connector
	getSrcConnResp, err := s.connectorServiceClient.GetSourceConnector(ctx,
		&connectorPB.GetSourceConnectorRequest{
			Name: recipe.Source,
		})
	if err != nil {
		return fmt.Errorf("[connector-backend: GetSourceConnector - Name: %s] %s", recipe.Source, err)
	}

	srcColID, err := resource.GetResourceCollectionID(recipe.Source)
	if err != nil {
		return err
	}

	recipe.Source = srcColID + "/" + getSrcConnResp.GetSourceConnector().GetUid()

	// Destination connector
	getDstConnResp, err := s.connectorServiceClient.GetDestinationConnector(ctx,
		&connectorPB.GetDestinationConnectorRequest{
			Name: recipe.Destination,
		})
	if err != nil {
		return fmt.Errorf("[connector-backend: GetDestinationConnector - Name: %s] %s", recipe.Destination, err)
	}

	dstColID, err := resource.GetResourceCollectionID(recipe.Destination)
	if err != nil {
		return err
	}

	recipe.Destination = dstColID + "/" + getDstConnResp.GetDestinationConnector().GetUid()

	// Model instances
	for idx, modelInstanceRscName := range recipe.ModelInstances {

		getModelInstResp, err := s.modelServiceClient.GetModelInstance(ctx,
			&modelPB.GetModelInstanceRequest{
				Name: modelInstanceRscName,
			})
		if err != nil {
			return fmt.Errorf("[model-backend: GetModelInstance - Name: %s] %s", modelInstanceRscName, err)
		}

		modelInstColID, err := resource.GetResourceCollectionID(modelInstanceRscName)
		if err != nil {
			return err
		}

		modelInstID, err := resource.GetResourceNameID(modelInstanceRscName)
		if err != nil {
			return err
		}

		modelRscName := strings.TrimSuffix(modelInstanceRscName, "/"+modelInstColID+"/"+modelInstID)

		getModelResp, err := s.modelServiceClient.GetModel(ctx,
			&modelPB.GetModelRequest{
				Name: modelRscName,
			})
		if err != nil {
			return fmt.Errorf("[model-backend: GetModel - Name: %s] %s", modelRscName, err)
		}

		modelColID, err := resource.GetResourceCollectionID(modelRscName)
		if err != nil {
			return err
		}

		recipe.ModelInstances[idx] = modelColID + "/" + getModelResp.GetModel().GetUid() + "/" + modelInstColID + "/" + getModelInstResp.GetInstance().GetUid()
	}

	return nil
}

func (s *service) recipePermalinkToName(recipe *datamodel.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Source connector
	lookUpSrcConnResp, err := s.connectorServiceClient.LookUpSourceConnector(ctx,
		&connectorPB.LookUpSourceConnectorRequest{
			Permalink: recipe.Source,
		})
	if err != nil {
		return fmt.Errorf("[connector-backend: LookUpSourceConnector - Permalink: %s] %s", recipe.Source, err)
	}

	srcColID, err := resource.GetResourceCollectionID(recipe.Source)
	if err != nil {
		return err
	}

	recipe.Source = srcColID + "/" + lookUpSrcConnResp.GetSourceConnector().GetId()

	// Destination connector
	lookUpDstConnResp, err := s.connectorServiceClient.LookUpDestinationConnector(ctx,
		&connectorPB.LookUpDestinationConnectorRequest{
			Permalink: recipe.Destination,
		})
	if err != nil {
		return fmt.Errorf("[connector-backend: LookUpDestinationConnector - Permalink: %s] %s", recipe.Destination, err)
	}

	dstColID, err := resource.GetResourceCollectionID(recipe.Destination)
	if err != nil {
		return err
	}

	recipe.Destination = dstColID + "/" + lookUpDstConnResp.GetDestinationConnector().GetId()

	// Model instances
	for idx, modelInstanceRscPermalink := range recipe.ModelInstances {

		lookUpModelInstResp, err := s.modelServiceClient.LookUpModelInstance(ctx,
			&modelPB.LookUpModelInstanceRequest{
				Permalink: modelInstanceRscPermalink,
			})
		if err != nil {
			return fmt.Errorf("[model-backend: LookUpModelInstance - Permalink: %s] %s", modelInstanceRscPermalink, err)
		}

		modelInstUID, err := resource.GetResourcePermalinkUID(modelInstanceRscPermalink)
		if err != nil {
			return err
		}

		modelInstColID, err := resource.GetResourceCollectionID(modelInstanceRscPermalink)
		if err != nil {
			return err
		}

		modelRscPermalink := strings.TrimSuffix(modelInstanceRscPermalink, "/"+modelInstColID+"/"+modelInstUID)
		lookUpModelResp, err := s.modelServiceClient.LookUpModel(ctx,
			&modelPB.LookUpModelRequest{
				Permalink: modelRscPermalink,
			})
		if err != nil {
			return fmt.Errorf("[model-backend: LookUpModel - Permalink: %s] %s", modelRscPermalink, err)
		}

		modelColID, err := resource.GetResourceCollectionID(modelRscPermalink)
		if err != nil {
			return err
		}

		recipe.ModelInstances[idx] = modelColID + "/" + lookUpModelResp.Model.GetId() + "/" + modelInstColID + "/" + lookUpModelInstResp.GetInstance().GetId()
	}

	return nil
}
