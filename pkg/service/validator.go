package service

import (
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

type SourceCategory int64

const (
	Unspecified SourceCategory = 0
	HTTP        SourceCategory = 1
	Grpc        SourceCategory = 2
	Pull        SourceCategory = 3
)

func (s *service) checkRecipe(ownerPermalink string, recipePermalink *datamodel.Recipe) error {

	return nil
	// disable this temporarily

	// startCnt := 0
	// endCnt := 0

	// componentIDMap := make(map[string]*datamodel.Component)
	// exp := "^[a-z_][-a-z_0-9]{0,31}$"
	// r, _ := regexp.Compile(exp)

	// for idx := range recipePermalink.Components {
	// 	if match := r.MatchString(recipePermalink.Components[idx].ID); !match {
	// 		return fmt.Errorf("component `id` needs to be started with a letter (uppercase or lowercase) or an underscore, followed by zero or more alphanumeric characters or underscores")
	// 	}
	// 	if _, ok := componentIDMap[recipePermalink.Components[idx].ID]; ok {
	// 		return fmt.Errorf("component `id` can not be duplicated")
	// 	}
	// 	componentIDMap[recipePermalink.Components[idx].ID] = recipePermalink.Components[idx]
	// }

	// startOpDef, err := s.operator.GetOperatorDefinitionByID("start", nil)
	// if err != nil {
	// 	return fmt.Errorf("operator-definitions/start not found")
	// }
	// endOpDef, err := s.operator.GetOperatorDefinitionByID("end", nil)
	// if err != nil {
	// 	return fmt.Errorf("operator-definitions/end not found")
	// }

	// for idx := range recipePermalink.Components {

	// 	if recipePermalink.Components[idx].DefinitionName == fmt.Sprintf("operator-definitions/%s", startOpDef.Uid) {
	// 		startCnt += 1
	// 	}
	// 	if recipePermalink.Components[idx].DefinitionName == fmt.Sprintf("operator-definitions/%s", endOpDef.Uid) {
	// 		endCnt += 1
	// 	}

	// 	var compJSONSchema []byte
	// 	if utils.IsConnectorDefinition(recipePermalink.Components[idx].DefinitionName) {

	// 		uid, err := resource.GetRscPermalinkUID(recipePermalink.Components[idx].DefinitionName)
	// 		if err != nil {
	// 			return fmt.Errorf("operator definition for component %s is not found", recipePermalink.Components[idx].ID)
	// 		}

	// 		def, err := s.connector.GetConnectorDefinitionByUID(uid, nil, nil)
	// 		if err != nil {
	// 			return fmt.Errorf("operator definition for component %s is not found", recipePermalink.Components[idx].ID)
	// 		}

	// 		compJSONSchema, err = protojson.Marshal(def.Spec.ComponentSpecification)

	// 		if err != nil {
	// 			return fmt.Errorf("connector definition for component %s is wrong", recipePermalink.Components[idx].ID)
	// 		}

	// 	}
	// 	if utils.IsOperatorDefinition(recipePermalink.Components[idx].DefinitionName) {

	// 		uid, err := resource.GetRscPermalinkUID(recipePermalink.Components[idx].DefinitionName)
	// 		if err != nil {
	// 			return fmt.Errorf("operator definition for component %s is not found", recipePermalink.Components[idx].ID)
	// 		}

	// 		def, err := s.operator.GetOperatorDefinitionByUID(uid, nil)
	// 		if err != nil {
	// 			return fmt.Errorf("operator definition for component %s is not found", recipePermalink.Components[idx].ID)
	// 		}

	// 		compJSONSchema, err = protojson.Marshal(def.Spec.ComponentSpecification)
	// 		if err != nil {
	// 			return fmt.Errorf("operator definition for component %s is wrong", recipePermalink.Components[idx].ID)
	// 		}
	// 	}

	// 	configJSON, err := protojson.Marshal(recipePermalink.Components[idx].Configuration)
	// 	if err != nil {
	// 		return fmt.Errorf("configuration for component %s is wrong %w", recipePermalink.Components[idx].ID, err)
	// 	}

	// 	sch, err := jsonschema.CompileString("schema.json", string(compJSONSchema))
	// 	if err != nil {
	// 		return err
	// 	}

	// 	var v interface{}
	// 	if err := json.Unmarshal(configJSON, &v); err != nil {
	// 		return err
	// 	}

	// 	if err = sch.Validate(v); err != nil {
	// 		return fmt.Errorf("configuration for component %s is wrong %w", recipePermalink.Components[idx].ID, err)
	// 	}

	// }

	// if startCnt != 1 {
	// 	return fmt.Errorf("need to have exactly one start operator")
	// }
	// if endCnt != 1 {
	// 	return fmt.Errorf("need to have exactly one end operator")
	// }

	// dag, err := utils.GenerateDAG(recipePermalink.Components)
	// if err != nil {
	// 	return err
	// }

	// _, err = dag.TopologicalSort()
	// if err != nil {
	// 	return err
	// }

	// return nil
}
