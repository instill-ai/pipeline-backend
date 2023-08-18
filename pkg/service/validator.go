package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/internal/resource"
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

func (s *service) checkRecipe(owner *mgmtPB.User, recipePermalink *datamodel.Recipe) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startCnt := 0
	endCnt := 0

	componentIdMap := make(map[string]*datamodel.Component)
	exp := "^[a-zA-Z_][a-zA-Z_0-9]*$"
	r, _ := regexp.Compile(exp)

	for idx := range recipePermalink.Components {
		if match := r.MatchString(recipePermalink.Components[idx].Id); !match {
			return status.Errorf(codes.InvalidArgument,
				fmt.Sprintf("[pipeline-backend] component `id` needs to be following with a regexp (%s)", exp))
		}
		if _, ok := componentIdMap[recipePermalink.Components[idx].Id]; ok {
			return status.Errorf(codes.InvalidArgument,
				"[pipeline-backend] component `id` duplicated")
		}
		componentIdMap[recipePermalink.Components[idx].Id] = recipePermalink.Components[idx]
	}

	startOpDef, err := s.operator.GetOperatorDefinitionById("start-operator")
	if err != nil {
		return err
	}
	endOpDef, err := s.operator.GetOperatorDefinitionById("end-operator")
	if err != nil {
		return err
	}

	for idx := range recipePermalink.Components {

		if recipePermalink.Components[idx].DefinitionName == fmt.Sprintf("operator-definitions/%s", startOpDef.Uid) {
			startCnt += 1
		}
		if recipePermalink.Components[idx].DefinitionName == fmt.Sprintf("operator-definitions/%s", endOpDef.Uid) {
			endCnt += 1
		}
		if IsConnector(recipePermalink.Components[idx].ResourceName) {

			checkResp, err := s.connectorPrivateServiceClient.CheckConnectorResource(
				utils.InjectOwnerToContext(ctx, owner),
				&connectorPB.CheckConnectorResourceRequest{
					Permalink: recipePermalink.Components[idx].ResourceName,
				},
			)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "[connector-backend] Error %s at %s: %v",
					"CheckConnector", recipePermalink.Components[idx].ResourceName, err.Error())

			}
			if checkResp.State != connectorPB.ConnectorResource_STATE_CONNECTED {
				return status.Errorf(codes.InvalidArgument, "[connector-backend] %s is not connected", recipePermalink.Components[idx].ResourceName)
			}
		}

		var compJsonSchema []byte
		if IsConnectorDefinition(recipePermalink.Components[idx].DefinitionName) {

			resp, err := s.connectorPrivateServiceClient.LookUpConnectorDefinitionAdmin(ctx, &connectorPB.LookUpConnectorDefinitionAdminRequest{
				Permalink: recipePermalink.Components[idx].DefinitionName,
				View:      connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return nil
			}

			def := resp.ConnectorDefinition

			compJsonSchema, err = protojson.Marshal(def.Spec.ComponentSpecification)

			if err != nil {
				return err
			}

		}
		if IsOperatorDefinition(recipePermalink.Components[idx].DefinitionName) {

			uid, err := resource.GetPermalinkUID(recipePermalink.Components[idx].DefinitionName)
			if err != nil {
				return err
			}

			def, err := s.operator.GetOperatorDefinitionByUid(uuid.FromStringOrNil(uid))
			if err != nil {
				return err
			}

			compJsonSchema, err = protojson.Marshal(def.Spec.ComponentSpecification)
			if err != nil {
				return err
			}
		}

		configJson, err := protojson.Marshal(recipePermalink.Components[idx].Configuration)
		if err != nil {
			return err
		}

		sch, err := jsonschema.CompileString("schema.json", string(compJsonSchema))
		if err != nil {
			return err
		}

		var v interface{}
		if err := json.Unmarshal(configJson, &v); err != nil {
			return err
		}

		if err = sch.Validate(v); err != nil {
			return err
		}

	}

	if startCnt != 1 {
		return status.Errorf(codes.InvalidArgument, "[pipeline-backend] need to have exactly one start operator")
	}
	if endCnt != 1 {
		return status.Errorf(codes.InvalidArgument, "[pipeline-backend] need to have exactly one end operator")
	}

	dag, err := utils.GenerateDAG(recipePermalink.Components)
	if err != nil {
		return err
	}

	_, err = dag.TopologicalSort()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "[pipeline-backend] The recipe is not legal: %v", err.Error())
	}

	return nil
}
