package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

type SourceCategory int64

const (
	Unspecified SourceCategory = 0
	Http        SourceCategory = 1
	Grpc        SourceCategory = 2
	Pull        SourceCategory = 3
)

func (s *service) checkRecipe(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) (datamodel.PipelineMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.IncludeConnectorTypeInRecipeByName(recipeRscName, owner)
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED), err
	}
	srcCnt := 0
	srcConnDefID := ""
	srcCategory := Unspecified

	dstConnDefIDs := []string{}
	dstHasHttp := false
	dstHasGrpc := false

	aiCnt := 0
	blockchainCnt := 0

	componentIdSet := make(map[string]bool)
	exp := "^[A-Za-z0-9]([A-Za-z0-9-_]{0,62}[A-Za-z0-9])?$"
	r, _ := regexp.Compile(exp)
	for _, component := range recipeRscName.Components {
		if match := r.MatchString(component.Id); !match {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, fmt.Sprintf("[pipeline-backend] component `id` needs to be within ASCII-only 64 characters following with a regexp (%s)", exp))
		}
		if componentIdSet[component.Id] {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[pipeline-backend] component `id` duplicated")
		}
		componentIdSet[component.Id] = true
	}

	for _, component := range recipeRscName.Components {

		connResp, err := s.connectorPublicServiceClient.GetConnector(utils.InjectOwnerToContext(ctx, owner),
			&connectorPB.GetConnectorRequest{
				Name: component.ResourceName,
			})
		if err != nil {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
					"GetConnector", component.ResourceName, err.Error())

		}

		watchResp, err := s.connectorPublicServiceClient.WatchConnector(
			utils.InjectOwnerToContext(ctx, owner),
			&connectorPB.WatchConnectorRequest{
				Name: component.ResourceName,
			},
		)

		if err != nil {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
					"WatchConnector", component.ResourceName, err.Error())

		}

		if watchResp.State != connectorPB.Connector_STATE_CONNECTED {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] %s is not connected", component.ResourceName)
		}

		connDefResp, err := s.connectorPublicServiceClient.GetConnectorDefinition(utils.InjectOwnerToContext(ctx, owner),
			&connectorPB.GetConnectorDefinitionRequest{
				Name: connResp.GetConnector().GetConnectorDefinitionName(),
			})

		if err != nil {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
					"GetConnectorDefinition", connResp.GetConnector().GetConnectorDefinition(), err.Error())
		}

		switch component.Type {

		case connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE.String():

			srcCnt += 1

			if srcCnt > 1 {
				return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
					status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have more than one source connector.")
			}

			srcConnDefID = connDefResp.GetConnectorDefinition().GetId()
			if strings.Contains(srcConnDefID, "source-http") {
				srcCategory = Http
			} else if strings.Contains(srcConnDefID, "source-grpc") {
				srcCategory = Grpc
			}

		case connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION.String():

			dstConnDefID := connDefResp.GetConnectorDefinition().GetId()
			dstConnDefIDs = append(dstConnDefIDs, dstConnDefID)
			if strings.Contains(dstConnDefID, "destination-http") {
				dstHasHttp = true
			}
			if strings.Contains(dstConnDefID, "destination-grpc") {
				dstHasGrpc = true
			}

		case connectorPB.ConnectorType_CONNECTOR_TYPE_AI.String():
			aiCnt += 1
		case connectorPB.ConnectorType_CONNECTOR_TYPE_BLOCKCHAIN.String():
			blockchainCnt += 1
		}
	}

	// Temporary Constraint
	// if aiCnt != 1 {
	// 	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
	// 		status.Errorf(codes.InvalidArgument, "[pipeline-backend] Need to have exactly one model")
	// }
	// Temporary Constraint
	// if len(dstConnDefIDs) != 1 {
	// 	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
	// 		status.Errorf(codes.InvalidArgument, "[pipeline-backend] Need to have exactly one destination connector")
	// }

	// if srcCnt == 0 {
	// 	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
	// 		status.Errorf(codes.InvalidArgument, "[pipeline-backend] Need to have one source connector")
	// }

	// if len(dstConnDefIDs) == 0 {
	// 	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
	// 		status.Errorf(codes.InvalidArgument, "[pipeline-backend] Need to have at least one destination connector")
	// }

	if srcCategory == Http && len(dstConnDefIDs) == 1 && dstHasHttp {
		// if aiCnt != 1 {
		// 	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
		// 		status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have more than one model for sync pipeline")
		// }
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
	} else if srcCategory == Grpc && len(dstConnDefIDs) == 1 && dstHasGrpc {
		// if aiCnt != 1 {
		// 	return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
		// 		status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have more than one model for sync pipeline")
		// }
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
	} else if srcCategory == Http && dstHasGrpc {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have http source connector with grpc destination connector")
	} else if srcCategory == Grpc && dstHasHttp {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have grpc source connector with http destination connector")
	} else if len(dstConnDefIDs) > 1 && dstHasHttp || dstHasGrpc {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can only have one destination connector for sync pipeline")
	} else {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_ASYNC), nil
	}

}
