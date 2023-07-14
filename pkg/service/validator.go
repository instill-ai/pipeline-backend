package service

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	mgmtPB "github.com/instill-ai/protogen-go/base/mgmt/v1alpha"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

type SourceCategory int64

const (
	Unspecified SourceCategory = 0
	Http        SourceCategory = 1
	Grpc        SourceCategory = 2
	Pull        SourceCategory = 3
)

func (s *service) checkRecipe(owner *mgmtPB.User, recipeRscName *datamodel.Recipe) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.IncludeConnectorTypeInRecipeByName(recipeRscName, owner)
	if err != nil {
		return err
	}
	triggerCnt := 0
	responseCnt := 0

	componentIdMap := make(map[string]*datamodel.Component)
	exp := "^[A-Za-z0-9]([A-Za-z0-9-_]{0,62}[A-Za-z0-9])?$"
	r, _ := regexp.Compile(exp)

	for idx := range recipeRscName.Components {
		if match := r.MatchString(recipeRscName.Components[idx].Id); !match {
			return status.Errorf(codes.InvalidArgument,
				fmt.Sprintf("[pipeline-backend] component `id` needs to be within ASCII-only 64 characters following with a regexp (%s)", exp))
		}
		if _, ok := componentIdMap[recipeRscName.Components[idx].Id]; ok {
			return status.Errorf(codes.InvalidArgument,
				"[pipeline-backend] component `id` duplicated")
		}
		componentIdMap[recipeRscName.Components[idx].Id] = recipeRscName.Components[idx]
	}

	for idx := range recipeRscName.Components {

		if recipeRscName.Components[idx].ResourceName == fmt.Sprintf("connectors/%s", constant.TriggerConnectorId) {
			triggerCnt += 1
		}
		if recipeRscName.Components[idx].ResourceName == fmt.Sprintf("connectors/%s", constant.ResponseConnectorId) {
			responseCnt += 1
		}
		watchResp, err := s.connectorPublicServiceClient.WatchConnector(
			utils.InjectOwnerToContext(ctx, owner),
			&connectorPB.WatchConnectorRequest{
				Name: recipeRscName.Components[idx].ResourceName,
			},
		)

		if err != nil {
			return status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
				"WatchConnector", recipeRscName.Components[idx].ResourceName, err.Error())

		}

		if watchResp.State != connectorPB.Connector_STATE_CONNECTED {
			return status.Errorf(codes.InvalidArgument, "[connector-backend] %s is not connected", recipeRscName.Components[idx].ResourceName)
		}

	}
	if triggerCnt != 1 {
		return status.Errorf(codes.InvalidArgument, "[pipeline-backend] need to have exactly one trigger connector")
	}
	if responseCnt > 1 {
		return status.Errorf(codes.InvalidArgument, "[pipeline-backend] need to have less or equal one response connector")
	}

	dag := utils.NewDAG(recipeRscName.Components)
	for _, component := range recipeRscName.Components {
		if component.Dependencies != nil {
			parents, _, err := utils.ParseDependency(component.Dependencies)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "dependencies error")
			}
			for idx := range parents {

				dag.AddEdge(componentIdMap[parents[idx]], component)
			}
		}

	}
	_, err = dag.TopoloicalSort()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "[pipeline-backend] The recipe is not legal: %v", err.Error())
	}

	return nil
}
