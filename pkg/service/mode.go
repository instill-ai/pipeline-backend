package service

import (
	"context"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"google.golang.org/grpc/codes"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) checkMode(recipeRscName *datamodel.Recipe) (datamodel.PipelineMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srcConnRscName := recipeRscName.Source
	dstConnRscName := recipeRscName.Destination

	srcConnResp, err := s.connectorServiceClient.GetSourceConnector(ctx,
		&connectorPB.GetSourceConnectorRequest{
			Name: srcConnRscName,
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connectors/%s: %v",
				"GetSourceConnector", srcConnRscName, err.Error())

	}

	srcConnDefResp, err := s.connectorServiceClient.GetSourceConnectorDefinition(ctx,
		&connectorPB.GetSourceConnectorDefinitionRequest{
			Name: srcConnResp.GetSourceConnector().GetSourceConnectorDefinition(),
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connector-definitions/%s: %v",
				"GetSourceConnectorDefinition", srcConnResp.GetSourceConnector().GetSourceConnectorDefinition(), err.Error())
	}

	srcConnType := srcConnDefResp.GetSourceConnectorDefinition().GetConnectorDefinition().GetConnectionType()

	dstConnResp, err := s.connectorServiceClient.GetDestinationConnector(ctx,
		&connectorPB.GetDestinationConnectorRequest{
			Name: dstConnRscName,
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at destination-connectors/%s: %v",
				"GetDestinationConnector", dstConnRscName, err.Error())
	}

	dstConnDefResp, err := s.connectorServiceClient.GetDestinationConnectorDefinition(ctx,
		&connectorPB.GetDestinationConnectorDefinitionRequest{
			Name: dstConnResp.GetDestinationConnector().GetDestinationConnectorDefinition(),
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connector-definitions/%s: %v",
				"GetDestinationConnectorDefinitionRequest", dstConnResp.GetDestinationConnector().GetDestinationConnectorDefinition(), err.Error())
	}

	dstConnType := dstConnDefResp.GetDestinationConnectorDefinition().GetConnectorDefinition().GetConnectionType()

	if srcConnType == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS &&
		dstConnType == connectorPB.ConnectionType_CONNECTION_TYPE_DIRECTNESS {

		// A hardcoding naming rule "source-*" and "destination-*" for directness connectors
		if strings.Split(srcConnDefResp.GetSourceConnectorDefinition().GetId(), "-")[1] ==
			strings.Split(dstConnDefResp.GetDestinationConnectorDefinition().GetId(), "-")[1] {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
		}

		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Error(codes.InvalidArgument, "Source and destination connector definition id must be the same if they are both directness connection type")
	}

	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC), nil
}
