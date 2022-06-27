package service

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) checkState(recipeRscName *datamodel.Recipe) (datamodel.PipelineState, error) {

	logger, _ := logger.GetZapLogger()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srcConnResp, err := s.connectorServiceClient.GetSourceConnector(ctx, &connectorPB.GetSourceConnectorRequest{
		Name: recipeRscName.Source,
	})
	if err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connectors/%s: %v", "GetDestinationConnector", recipeRscName.Source, err.Error())
	}

	srcConnState := int(srcConnResp.GetSourceConnector().GetConnector().GetState().Number())

	dstConnResp, err := s.connectorServiceClient.GetDestinationConnector(ctx, &connectorPB.GetDestinationConnectorRequest{
		Name: recipeRscName.Destination,
	})
	if err != nil {
		return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at destination-connectors/%s: %v", "GetDestinationConnector", recipeRscName.Destination, err.Error())
	}

	dstConnState := int(dstConnResp.GetDestinationConnector().GetConnector().GetState().Number())

	modelInstStates := make([]int, len(recipeRscName.ModelInstances))
	for idx, modelInst := range recipeRscName.ModelInstances {
		modelInstResp, err := s.modelServiceClient.GetModelInstance(ctx, &modelPB.GetModelInstanceRequest{
			Name: modelInst,
		})
		if err != nil {
			return datamodel.PipelineState(pipelinePB.Pipeline_STATE_UNSPECIFIED),
				status.Errorf(codes.Internal, "[model-backend] Error %s at %dth model instance %s: %v", "GetModelInstance", idx, modelInst, err.Error())
		}
		modelInstStates[idx] = int(modelInstResp.Instance.State.Number())
	}

	// State precedence rule (i.e., enum_number state logic) : 3 error (any of) > 0 unspecified (any of) > 1 negative (any of) > 2 positive (all of)
	states := []int{srcConnState, dstConnState}
	states = append(states, modelInstStates...)

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
