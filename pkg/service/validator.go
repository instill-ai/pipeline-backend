package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/utils"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

type SourceCategory int64

const (
	Unspecified SourceCategory = 0
	Http        SourceCategory = 1
	Grpc        SourceCategory = 2
	Pull        SourceCategory = 3
)

func (s *service) checkRecipe(ownerPermalink string, recipePermalink *datamodel.Recipe) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startCnt := 0
	endCnt := 0

	componentIdMap := make(map[string]*datamodel.Component)
	exp := "^[a-zA-Z_][a-zA-Z_0-9]*$"
	r, _ := regexp.Compile(exp)

	for idx := range recipePermalink.Components {
		if match := r.MatchString(recipePermalink.Components[idx].Id); !match {
			return fmt.Errorf("component `id` needs to be started with a letter (uppercase or lowercase) or an underscore, followed by zero or more alphanumeric characters or underscores")
		}
		if _, ok := componentIdMap[recipePermalink.Components[idx].Id]; ok {
			return fmt.Errorf("component `id` can not be duplicated")
		}
		componentIdMap[recipePermalink.Components[idx].Id] = recipePermalink.Components[idx]
	}

	startOpDef, err := s.operator.GetOperatorDefinitionByID("op-start")
	if err != nil {
		return fmt.Errorf("operator-definitions/op-start not found")
	}
	endOpDef, err := s.operator.GetOperatorDefinitionByID("op-end")
	if err != nil {
		return fmt.Errorf("operator-definitions/op-end not found")
	}

	for idx := range recipePermalink.Components {

		if recipePermalink.Components[idx].DefinitionName == fmt.Sprintf("operator-definitions/%s", startOpDef.Uid) {
			startCnt += 1
		}
		if recipePermalink.Components[idx].DefinitionName == fmt.Sprintf("operator-definitions/%s", endOpDef.Uid) {
			endCnt += 1
		}

		var compJsonSchema []byte
		if utils.IsConnectorDefinition(recipePermalink.Components[idx].DefinitionName) {

			resp, err := s.connectorPrivateServiceClient.LookUpConnectorDefinitionAdmin(ctx, &connectorPB.LookUpConnectorDefinitionAdminRequest{
				Permalink: recipePermalink.Components[idx].DefinitionName,
				View:      connectorPB.View_VIEW_FULL.Enum(),
			})
			if err != nil {
				return fmt.Errorf("connector definition for component %s is not found", recipePermalink.Components[idx].Id)
			}

			def := resp.ConnectorDefinition

			compJsonSchema, err = protojson.Marshal(def.Spec.ComponentSpecification)

			if err != nil {
				return fmt.Errorf("connector definition for component %s is wrong", recipePermalink.Components[idx].Id)
			}

		}
		if utils.IsOperatorDefinition(recipePermalink.Components[idx].DefinitionName) {

			uid, err := resource.GetRscPermalinkUID(recipePermalink.Components[idx].DefinitionName)
			if err != nil {
				return fmt.Errorf("operator definition for component %s is not found", recipePermalink.Components[idx].Id)
			}

			def, err := s.operator.GetOperatorDefinitionByUID(uid)
			if err != nil {
				return fmt.Errorf("operator definition for component %s is not found", recipePermalink.Components[idx].Id)
			}

			compJsonSchema, err = protojson.Marshal(def.Spec.ComponentSpecification)
			if err != nil {
				return fmt.Errorf("operator definition for component %s is wrong", recipePermalink.Components[idx].Id)
			}
		}

		configJson, err := protojson.Marshal(recipePermalink.Components[idx].Configuration)
		if err != nil {
			return fmt.Errorf("configuration for component %s is wrong", recipePermalink.Components[idx].Id)
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
			return fmt.Errorf("configuration for component %s is wrong", recipePermalink.Components[idx].Id)
		}

	}

	if startCnt != 1 {
		return fmt.Errorf("need to have exactly one start operator")
	}
	if endCnt != 1 {
		return fmt.Errorf("need to have exactly one end operator")
	}

	dag, err := utils.GenerateDAG(recipePermalink.Components)
	if err != nil {
		return err
	}

	_, err = dag.TopologicalSort()
	if err != nil {
		return err
	}

	return nil
}
