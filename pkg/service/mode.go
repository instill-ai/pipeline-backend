package service

import (
	"context"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"google.golang.org/grpc/codes"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func (s *service) checkMode(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) (datamodel.PipelineMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srcConnRscName := recipeRscName.Source
	dstConnRscName := recipeRscName.Destination

	srcConnResp, err := s.connectorPublicServiceClient.GetSourceConnector(InjectOwnerToContext(ctx, owner),
		&connectorPB.GetSourceConnectorRequest{
			Name: srcConnRscName,
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connectors/%s: %v",
				"GetSourceConnector", srcConnRscName, err.Error())

	}

	srcConnDefResp, err := s.connectorPublicServiceClient.GetSourceConnectorDefinition(InjectOwnerToContext(ctx, owner),
		&connectorPB.GetSourceConnectorDefinitionRequest{
			Name: srcConnResp.GetSourceConnector().GetSourceConnectorDefinition(),
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connector-definitions/%s: %v",
				"GetSourceConnectorDefinition", srcConnResp.GetSourceConnector().GetSourceConnectorDefinition(), err.Error())
	}

	srcConnDefID := srcConnDefResp.GetSourceConnectorDefinition().GetId()

	dstConnResp, err := s.connectorPublicServiceClient.GetDestinationConnector(InjectOwnerToContext(ctx, owner),
		&connectorPB.GetDestinationConnectorRequest{
			Name: dstConnRscName,
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at destination-connectors/%s: %v",
				"GetDestinationConnector", dstConnRscName, err.Error())
	}

	dstConnDefResp, err := s.connectorPublicServiceClient.GetDestinationConnectorDefinition(InjectOwnerToContext(ctx, owner),
		&connectorPB.GetDestinationConnectorDefinitionRequest{
			Name: dstConnResp.GetDestinationConnector().GetDestinationConnectorDefinition(),
		})
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.Internal, "[connector-backend] Error %s at source-connector-definitions/%s: %v",
				"GetDestinationConnectorDefinitionRequest", dstConnResp.GetDestinationConnector().GetDestinationConnectorDefinition(), err.Error())
	}

	dstConnDefID := dstConnDefResp.GetDestinationConnectorDefinition().GetId()

	switch {
	case strings.Contains(srcConnDefID, "http") && strings.Contains(dstConnDefID, "http"):
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
	case strings.Contains(srcConnDefID, "grpc") && strings.Contains(dstConnDefID, "grpc"):
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
	default:
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC), nil
	}

}
