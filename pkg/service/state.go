package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) checkState(recipeRscName *datamodel.Recipe) (datamodel.PipelineState, error) {

	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var srcConnState int
	if srcResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: fmt.Sprintf("resources/%s/types/source-connectors", strings.Split(recipeRscName.Source, "/")[1]),
	}); err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[Controller] Error %s at source-connectors/%s: %v", "GetResourceState", recipeRscName.Source, err.Error())
	} else {
		srcConnState = int(srcResource.GetResource().GetConnectorState().Number())
	}

	var dstConnState int
	if dstResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
		Name: fmt.Sprintf("resources/%s/types/destination-connectors", strings.Split(recipeRscName.Destination, "/")[1]),
	}); err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[Controller] Error %s at source-connectors/%s: %v", "GetResourceState", recipeRscName.Source, err.Error())
	} else {
		dstConnState = int(dstResource.GetResource().GetConnectorState().Number())
	}

	modelStates := make([]int, len(recipeRscName.Models))
	for idx, model := range recipeRscName.Models {
		if modelResource, err := s.controllerClient.GetResource(ctx, &controllerPB.GetResourceRequest{
			Name: fmt.Sprintf("resources/%s/types/models", strings.Split(model, "/")[1]),
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
