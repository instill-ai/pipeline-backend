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
	componentIdMap := make(map[string]*datamodel.Component)
	exp := "^[A-Za-z0-9]([A-Za-z0-9-_]{0,62}[A-Za-z0-9])?$"
	r, _ := regexp.Compile(exp)

	for idx := range recipeRscName.Components {
		if match := r.MatchString(recipeRscName.Components[idx].Id); !match {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, fmt.Sprintf("[pipeline-backend] component `id` needs to be within ASCII-only 64 characters following with a regexp (%s)", exp))
		}
		if componentIdSet[recipeRscName.Components[idx].Id] {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[pipeline-backend] component `id` duplicated")
		}
		componentIdSet[recipeRscName.Components[idx].Id] = true
		componentIdMap[recipeRscName.Components[idx].Id] = recipeRscName.Components[idx]
	}

	for idx := range recipeRscName.Components {

		connResp, err := s.connectorPublicServiceClient.GetConnector(utils.InjectOwnerToContext(ctx, owner),
			&connectorPB.GetConnectorRequest{
				Name: recipeRscName.Components[idx].ResourceName,
			})
		if err != nil {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
					"GetConnector", recipeRscName.Components[idx].ResourceName, err.Error())

		}

		watchResp, err := s.connectorPublicServiceClient.WatchConnector(
			utils.InjectOwnerToContext(ctx, owner),
			&connectorPB.WatchConnectorRequest{
				Name: recipeRscName.Components[idx].ResourceName,
			},
		)

		if err != nil {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
					"WatchConnector", recipeRscName.Components[idx].ResourceName, err.Error())

		}

		if watchResp.State != connectorPB.Connector_STATE_CONNECTED {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[connector-backend] %s is not connected", recipeRscName.Components[idx].ResourceName)
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

		switch recipeRscName.Components[idx].Type {

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

	dag := utils.NewDAG(recipeRscName.Components)
	for _, component := range recipeRscName.Components {
		parents, _, err := utils.ParseDependency(component.Dependencies)
		if err != nil {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "dependencies error")
		}
		for idx := range parents {

			dag.AddEdge(componentIdMap[parents[idx]], component)
		}
	}
	_, err = dag.TopoloicalSort()
	if err != nil {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.InvalidArgument, "[pipeline-backend] The recipe is not a valid single source DAG: %v", err.Error())
	}

	if srcCategory == Http && len(dstConnDefIDs) == 1 && dstHasHttp {
		if len(recipeRscName.Components) > 4 {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have more than 4 components in sync pipeline")
		}
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
	} else if srcCategory == Grpc && len(dstConnDefIDs) == 1 && dstHasGrpc {
		if len(recipeRscName.Components) > 4 {
			return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
				status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have more than 4 components in sync pipeline")
		}
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
