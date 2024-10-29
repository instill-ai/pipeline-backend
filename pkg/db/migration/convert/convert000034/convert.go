package convert000034

import (
	"bytes"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
)

const batchSize = 100

type RenameHTTPComponent struct {
	convert.Basic
}

// This migration mainly focuses on updating the following pipelines and the pipelines are similar to the following pipelines:
// https://instill.tech/abrc/pipelines/stomavision/playground?version=v1.0.0
// https://instill.tech/leochen5/pipelines/index-preprocess-img-desc/playground?version=v1.0.0
// https://instill.tech/instill-wombat/pipelines/jumbotron-visual-understanding/preview?version=v4.0.0

func (c *ConvertInstillModel) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *RenameHTTPComponent) migratePipeline() error {
	q := c.DB.Select("uid", "recipe_yaml").
		Where("recipe_yaml LIKE ?", "%%type: restapi%%").
		Where("delete_time IS NULL")

	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return q.FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(p.RecipeYAML), &node); err != nil {
				return fmt.Errorf("unmarshalling recipe yaml: %w", err)
			}

			// Find and update the component section
			for i := 0; i < len(node.Content[0].Content); i += 2 {
				if node.Content[0].Content[i].Value == "component" {
					componentNode := node.Content[0].Content[i+1]
					isComponentUpdated, err := c.updateComponentNode(componentNode)
					if err != nil {
						l.Error("Failed to update component node")
						return fmt.Errorf("updating component node: %w", err)
					}
					isRecipeUpdated = isComponentUpdated || isRecipeUpdated
					break
				}
			}

			if isRecipeUpdated {
				buf := bytes.Buffer{}
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				err := enc.Encode(&node.Content[0])
				if err != nil {
					return fmt.Errorf("marshalling recipe: %w", err)
				}
				recipeYAML := buf.String()

				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", string(recipeYAML))
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *RenameHTTPComponent) migratePipelineRelease() error {
	q := c.DB.Select("uid", "recipe_yaml").
		Where("recipe_yaml LIKE ?", "%%type: restapi%%").
		Where("delete_time IS NULL")

	pipelineReleases := make([]*datamodel.PipelineRelease, 0, batchSize)
	return q.FindInBatches(&pipelineReleases, batchSize, func(tx *gorm.DB, _ int) error {
		for _, pr := range pipelineReleases {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", pr.UID.String()))

			var node yaml.Node
			if err := yaml.Unmarshal([]byte(pr.RecipeYAML), &node); err != nil {
				return fmt.Errorf("unmarshalling recipe yaml: %w", err)
			}

			// Find and update the component section
			for i := 0; i < len(node.Content[0].Content); i += 2 {
				if node.Content[0].Content[i].Value == "component" {
					componentNode := node.Content[0].Content[i+1]
					isComponentUpdated, err := c.updateComponentNode(componentNode)
					if err != nil {
						l.Error("Failed to update component node")
						return fmt.Errorf("updating component node: %w", err)
					}
					isRecipeUpdated = isComponentUpdated || isRecipeUpdated
					break
				}
			}

			if isRecipeUpdated {
				buf := bytes.Buffer{}
				enc := yaml.NewEncoder(&buf)
				enc.SetIndent(2)
				err := enc.Encode(&node.Content[0])
				if err != nil {
					return fmt.Errorf("marshalling recipe: %w", err)
				}
				recipeYAML := buf.String()
				result := tx.Model(pr).Where("uid = ?", pr.UID).Update("recipe_yaml", string(recipeYAML))
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline release recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *RenameHTTPComponent) updateComponentNode(node *yaml.Node) (bool, error) {
	isUpdated := false
	for j := 0; j < len(node.Content); j += 2 {
		compContent := node.Content[j+1]

		// Check if this is an iterator component
		isIterator := false
		for k := 0; k < len(compContent.Content); k += 2 {
			if compContent.Content[k].Value == "type" && compContent.Content[k+1].Value == "iterator" {
				isIterator = true
				break
			}
		}

		if v, ok := input["image-url"]; ok {
			input["data"] = map[string]interface{}{
				"image-url": v,
				"type":      "image-url",
			}
			delete(input, "image-url")
		}

		*instillModelComponentNames = append(*instillModelComponentNames, compName)

		return true, nil
	}

	return false, nil
}

func (c *ConvertInstillModel) updateInstillModelChatRelatedTasks(compName string, comp *datamodel.Component, instillModelComponentNames *[]string) (bool, error) {

	if comp.Type == "iterator" {
		isComponentUpdated := false
		for compNameInIterator, comp := range comp.Component {
			isUpdated, err := c.updateInstillModelChatRelatedTasks(compNameInIterator, comp, instillModelComponentNames)
			if err != nil {
				return false, err
			}
			isComponentUpdated = isUpdated || isComponentUpdated
		}
		return isComponentUpdated, nil
	}

	if comp.Type != "instill-model" {
		return false, nil
	}

	input, isMap := comp.Input.(map[string]interface{})

	if !isMap {
		return false, nil
	}

	answeringRelatedTasks := []string{
		"TASK_TEXT_GENERATION",
		"TASK_TEXT_GENERATION_CHAT",
		"TASK_VISUAL_QUESTION_ANSWERING",
	}

	if slices.Contains(answeringRelatedTasks, comp.Task) {
		if comp.Task == "TASK_TEXT_GENERATION_CHAT" || comp.Task == "TASK_VISUAL_QUESTION_ANSWERING" {
			comp.Task = "TASK_CHAT"
		}

		if comp.Task == "TASK_TEXT_GENERATION" {
			comp.Task = "TASK_COMPLETION"
		}

		if v, ok := input["model-name"]; ok {
			input["data"] = map[string]interface{}{
				"model": v,
			}
			delete(input, "model-name")
		}

		input["data"].(map[string]interface{})["messages"] = []map[string]interface{}{}

		if v, ok := input["prompt"]; ok {
			messages := input["data"].(map[string]interface{})["messages"].([]map[string]interface{})

			newMessage := map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"text": v,
						"type": "text",
					},
				},
				"role": "user",
			}

			messages = append(messages, newMessage)

			input["data"].(map[string]interface{})["messages"] = messages

			delete(input, "prompt")
		}

		if v, ok := input["prompt-images"]; ok {

			if v.(string) == "${variable.images}" {
				v = "${variable.images[0]}"
				messages := input["data"].(map[string]interface{})["messages"].([]map[string]interface{})

				newMessage := map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"image-base64": v,
							"type":         "image-base64",
						},
					},
					"role": "user",
				}
				messages = append(messages, newMessage)
				input["data"].(map[string]interface{})["messages"] = messages
			}

			if images, ok := v.([]interface{}); ok && len(images) > 0 {

				for i := range images {
					messages := input["data"].(map[string]interface{})["messages"].([]map[string]interface{})

					newMessage := map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"image-base64": images[i],
								"type":         "image-base64",
							},
						},
						"role": "user",
					}
					messages = append(messages, newMessage)
					input["data"].(map[string]interface{})["messages"] = messages
				}
			}

			delete(input, "prompt-images")
		}

		if v, ok := input["max-new-tokens"]; ok {
			input["parameter"] = map[string]interface{}{
				"max-tokens": v,
			}
			delete(input, "max-new-tokens")
		}

		paramFields := []string{
			"temperature",
			"top-k",
			"top-p",
			"seed",
			"stream",
			"n",
		}

		for _, field := range paramFields {
			if v, ok := input[field]; ok {
				input["parameter"].(map[string]interface{})[field] = v
				delete(input, field)
			}
		}

		*instillModelComponentNames = append(*instillModelComponentNames, compName)

		return true, nil
	}

	return false, nil
}

func (c *ConvertInstillModel) updateInstillModelImageRelatedOutput(recipeYAML string, instillModelComponentNames []string) (string, error) {
	originalFields := []string{
		"objects",
	}

	updatedRecipe := recipeYAML
	for _, compName := range instillModelComponentNames {
		for _, field := range originalFields {
			// It will be wrong if we add `}` at the end
			originalInstillModelOutput := fmt.Sprintf("${%s.output.%s", compName, field)
			if !strings.Contains(recipeYAML, originalInstillModelOutput) {
				continue
			}
			updatedRecipe = strings.ReplaceAll(updatedRecipe, originalInstillModelOutput, fmt.Sprintf("${%s.output.data.%s", compName, field))
		}
	}

	return updatedRecipe, nil
}

func (c *ConvertInstillModel) updateInstillModelChatRelatedOutput(recipeYAML string, instillModelComponentNames []string) (string, error) {
	originalFields := []string{
		"text",
	}

	updatedRecipe := recipeYAML
	for _, compName := range instillModelComponentNames {
		for _, field := range originalFields {
			// It will be wrong if we add `}` at the end
			originalInstillModelOutput := fmt.Sprintf("${%s.output.%s", compName, field)
			if !strings.Contains(recipeYAML, originalInstillModelOutput) {
				continue
			}
			updatedRecipe = strings.ReplaceAll(updatedRecipe, originalInstillModelOutput, fmt.Sprintf("${%s.output.choices[0].message.content", compName))
		}
	}
	return isUpdated, nil
}
