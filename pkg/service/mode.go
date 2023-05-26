package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"google.golang.org/grpc/codes"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
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

	srcCnt := 0
	srcConnDefID := ""
	srcCategory := Unspecified

	dstConnDefIDs := []string{}
	dstHasHttp := false
	dstHasGrpc := false

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
		switch utils.GetDefinitionType(component) {

		case utils.SourceConnector:

			srcCnt += 1

			if srcCnt > 1 {
				return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
					status.Errorf(codes.InvalidArgument, "[pipeline-backend] Can not have more than one source connector.")
			}

			srcConnResp, err := s.connectorPublicServiceClient.GetSourceConnector(utils.InjectOwnerToContext(ctx, owner),
				&connectorPB.GetSourceConnectorRequest{
					Name: component.ResourceName,
				})
			if err != nil {
				return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
					status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
						"GetSourceConnector", component.ResourceName, err.Error())

			}

			srcConnDefResp, err := s.connectorPublicServiceClient.GetSourceConnectorDefinition(utils.InjectOwnerToContext(ctx, owner),
				&connectorPB.GetSourceConnectorDefinitionRequest{
					Name: srcConnResp.GetSourceConnector().GetSourceConnectorDefinition(),
				})
			if err != nil {
				return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
					status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
						"GetSourceConnectorDefinition", srcConnResp.GetSourceConnector().GetSourceConnectorDefinition(), err.Error())
			}

			srcConnDefID = srcConnDefResp.GetSourceConnectorDefinition().GetId()
			if strings.Contains(srcConnDefID, "http") {
				srcCategory = Http
			} else if strings.Contains(srcConnDefID, "grpc") {
				srcCategory = Grpc
			}

		case utils.DestinationConnector:

			dstConnResp, err := s.connectorPublicServiceClient.GetDestinationConnector(utils.InjectOwnerToContext(ctx, owner),
				&connectorPB.GetDestinationConnectorRequest{
					Name: component.ResourceName,
				})
			if err != nil {
				return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
					status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
						"GetDestinationConnector", component.ResourceName, err.Error())
			}

			dstConnDefResp, err := s.connectorPublicServiceClient.GetDestinationConnectorDefinition(utils.InjectOwnerToContext(ctx, owner),
				&connectorPB.GetDestinationConnectorDefinitionRequest{
					Name: dstConnResp.GetDestinationConnector().GetDestinationConnectorDefinition(),
				})
			if err != nil {
				return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
					status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
						"GetDestinationConnectorDefinitionRequest", dstConnResp.GetDestinationConnector().GetDestinationConnectorDefinition(), err.Error())
			}
			dstConnDefID := dstConnDefResp.GetDestinationConnectorDefinition().GetId()
			dstConnDefIDs = append(dstConnDefIDs, dstConnDefID)
			if strings.Contains(dstConnDefID, "http") {
				dstHasHttp = true
			}
			if strings.Contains(dstConnDefID, "grpc") {
				dstHasGrpc = true
			}
		}
	}

	if srcCnt == 0 {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.InvalidArgument, "[pipeline-backend] Need to have one source connector")
	}

	if len(dstConnDefIDs) == 0 {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_UNSPECIFIED),
			status.Errorf(codes.InvalidArgument, "[pipeline-backend] Need to have at least one destination connector")
	}

	if srcCategory == Http && len(dstConnDefIDs) == 1 && dstHasHttp {
		return datamodel.PipelineMode(pipelinePB.Pipeline_MODE_SYNC), nil
	} else if srcCategory == Grpc && len(dstConnDefIDs) == 1 && dstHasGrpc {
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
