package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) checkState(recipePermalink *datamodel.Recipe) (datamodel.PipelineState, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	logger, _ := logger.GetZapLogger(ctx)
	defer cancel()

	states := []int{}
	for _, component := range recipePermalink.Components {

		if IsConnector(component.ResourceName) {
			connectorUID, err := resource.GetPermalinkUID(component.ResourceName)
			if err != nil {
				return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED), err
			}
			if dstResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
				ResourcePermalink: ConvertResourceUIDToControllerResourcePermalink(connectorUID, "connectors"),
			}); err != nil {
				return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
					status.Errorf(codes.Internal, "[Controller] Error %s at %s: %v", "GetResourceState", component.ResourceName, err.Error())
			} else {
				states = append(states, int(dstResource.GetResource().GetConnectorState().Number()))
			}
		}

	}

	// State precedence rule (i.e., enum_number state logic) : 3 error (any of) > 0 unspecified (any of) > 1 negative (any of) > 2 positive (all of)
	if contains(states, 3) {
		logger.Info(fmt.Sprintf("Component state: %v", states))
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_ERROR), nil
	}

	if contains(states, 0) {
		logger.Info(fmt.Sprintf("Component state: %v", states))
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED), nil
	}

	if contains(states, 1) {
		logger.Info(fmt.Sprintf("Component state: %v", states))
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_INACTIVE), nil
	}

	return datamodel.PipelineState(pipelinePB.Pipeline_STATE_ACTIVE), nil
}

func contains(slice interface{}, elem interface{}) bool {
	for _, v := range slice.([]int) {
		if v == elem {
			return true
		}
	}
	return false
}
