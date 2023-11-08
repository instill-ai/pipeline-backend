package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
)

func (s *service) checkState(recipePermalink *datamodel.Recipe) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	logger, _ := logger.GetZapLogger(ctx)
	defer cancel()

	states := []int{}
	for _, component := range recipePermalink.Components {

		if utils.IsConnectorWithNamespace(component.ResourceName) {
			connectorUID, err := resource.GetRscPermalinkUID(component.ResourceName)
			if err != nil {
				return err
			}
			if dstResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
				ResourcePermalink: ConvertResourceUIDToControllerResourcePermalink(connectorUID, "connectors"),
			}); err != nil {
				return status.Errorf(codes.Internal, "[Controller] Error %s at %s: %v", "GetResourceState", component.ResourceName, err.Error())
			} else {
				states = append(states, int(dstResource.GetResource().GetConnectorState().Number()))
			}
		}

	}

	// State precedence rule (i.e., enum_number state logic) : 3 error (any of) > 0 unspecified (any of) > 1 negative (any of) > 2 positive (all of)
	if contains(states, 3) {
		logger.Info(fmt.Sprintf("component state: %v", states))
		return fmt.Errorf("component state: %v", states)
	}

	if contains(states, 0) {
		logger.Info(fmt.Sprintf("component state: %v", states))
		return fmt.Errorf("component state: %v", states)
	}

	if contains(states, 1) {
		logger.Info(fmt.Sprintf("component state: %v", states))
		return fmt.Errorf("component state: %v", states)
	}

	return nil
}

func contains(slice interface{}, elem interface{}) bool {
	for _, v := range slice.([]int) {
		if v == elem {
			return true
		}
	}
	return false
}
