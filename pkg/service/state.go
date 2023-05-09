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

	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var srcConnState int
	sourceConnectorUID, err := resource.GetPermalinkUID(recipePermalink.Source)
	if err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED), err
	}
	if srcResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: ConvertResourceUIDToControllerResourcePermalink(sourceConnectorUID, "source-connectors"),
	}); err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[Controller] Error %s at %s: %v", "GetResourceState", recipePermalink.Source, err.Error())
	} else {
		srcConnState = int(srcResource.GetResource().GetConnectorState().Number())
	}

	var dstConnState int
	destinationConnectorUID, err := resource.GetPermalinkUID(recipePermalink.Destination)
	if err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED), err
	}
	if dstResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		ResourcePermalink: ConvertResourceUIDToControllerResourcePermalink(destinationConnectorUID, "destination-connectors"),
	}); err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[Controller] Error %s at %s: %v", "GetResourceState", recipePermalink.Destination, err.Error())
	} else {
		dstConnState = int(dstResource.GetResource().GetConnectorState().Number())
	}

	modelStates := make([]int, len(recipePermalink.Models))
	for idx, model := range recipePermalink.Models {
		modelUID, err := resource.GetPermalinkUID(model)
		if err != nil {
			return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED), err
		}
		if modelResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
			ResourcePermalink: ConvertResourceUIDToControllerResourcePermalink(modelUID, "models"),
		}); err != nil {
			return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
				status.Errorf(codes.Internal, "[Controller] Error %s at %dth model %s: %v", "GetResourceState", idx, model, err.Error())
		} else{
			modelStates[idx] = int(modelResource.GetResource().GetModelState().Number())
		}
	}

	// State precedence rule (i.e., enum_number state logic) : 3 error (any of) > 0 unspecified (any of) > 1 negative (any of) > 2 positive (all of)
	states := []int{srcConnState, dstConnState}
	states = append(states, modelStates...)

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
