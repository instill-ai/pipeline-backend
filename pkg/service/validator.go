package service

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func checkTask(compID, targetTask string, compSpec *structpb.Struct, compProperties map[string]any, validationErrors *[]*pb.PipelineValidationError) {
	taskMatch := false
	for _, t := range compSpec.Fields["oneOf"].GetListValue().Values {
		task := t.GetStructValue().Fields["properties"].GetStructValue().Fields["task"].GetStructValue().Fields["const"].GetStringValue()
		if task == targetTask {
			taskMatch = true
			compProperties[compID] = t
		}
	}
	if !taskMatch {
		*validationErrors = append(*validationErrors, &pb.PipelineValidationError{
			Location: "component." + compID,
			Error:    "task not correct",
		})
	}
	// return nil
}

func (s *service) checkRecipe(recipePermalink *datamodel.Recipe) ([]*pb.PipelineValidationError, error) {

	validationErrors := []*pb.PipelineValidationError{}

	schema := map[string]any{}

	_ = json.Unmarshal(recipe.RecipeSchema, &schema)

	compProperties := map[string]any{}

	for id, comp := range recipePermalink.Component {
		switch comp.Type {
		default:

			def, err := s.component.GetDefinitionByID(comp.Type, nil, nil)
			if err != nil {
				return nil, err
			}
			checkTask(id, comp.Task, def.Spec.ComponentSpecification, compProperties, &validationErrors)

		case datamodel.Iterator:
			nestedCompProperties := map[string]any{}
			nestedValidationErrors := []*pb.PipelineValidationError{}
			for nestedID, nestedComp := range comp.Component {
				if nestedComp.Type != datamodel.Iterator {
					def, err := s.component.GetDefinitionByID(nestedComp.Type, nil, nil)
					if err != nil {
						return nil, err
					}
					checkTask(nestedID, nestedComp.Task, def.Spec.ComponentSpecification, nestedCompProperties, &nestedValidationErrors)
				}

			}
			for _, e := range nestedValidationErrors {
				validationErrors = append(validationErrors, &pb.PipelineValidationError{
					Location: "component." + id + "." + e.Location,
					Error:    e.Error,
				})

			}

			compProperties[id] = map[string]any{}
			compProperties[id] = map[string]any{
				"properties": map[string]any{
					"component": map[string]any{
						"properties": nestedCompProperties,
					},
				},
			}

		}

	}

	schema["properties"].(map[string]any)["component"].(map[string]any)["properties"] = compProperties

	schemaByte, _ := json.Marshal(schema)

	c := jsonschema.NewCompiler()

	err := c.AddResource("schema.json", bytes.NewReader(schemaByte))
	if err != nil {
		return nil, err
	}
	sch, err := c.Compile("schema.json")
	if err != nil {
		return nil, err
	}

	recipeJSON, err := json.Marshal(recipePermalink)
	if err != nil {
		return nil, err
	}

	v := map[string]any{}
	_ = json.Unmarshal(recipeJSON, &v)

	err = sch.Validate(v)

	if err != nil {

		switch err := err.(type) {
		case *jsonschema.ValidationError:

			for _, detail := range err.BasicOutput().Errors {

				if detail.InstanceLocation == "" || detail.Error == "" {
					continue
				}
				loc := strings.ReplaceAll(detail.InstanceLocation, "/", ".")
				loc = loc[1:]
				validationErrors = append(validationErrors, &pb.PipelineValidationError{
					Location: loc,
					Error:    detail.Error,
				})
			}
		default:
			return nil, err
		}
	}

	return validationErrors, nil
}
