package convert000036

import (
	"fmt"
	"slices"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
)

const batchSize = 100

type ConvertInstillModel struct {
	convert.Basic
}

// This migration mainly focuses on updating the following pipelines and the pipelines are similar to the following pipelines:
// https://instill.tech/abrc/pipelines/stomavision/playground?version=v1.0.0
// https://instill.tech/leochen5/pipelines/index-preprocess-img-desc/playground?version=v1.0.0
// https://instill.tech/instill-wombat/pipelines/jumbotron-visual-understanding/preview?version=v4.0.0
// https://instill.tech/leochen5/pipelines/vlm-text-extraction-id/playground?view=BE8zvSpABUSjscJOGp1nRUaSOuHBUusr
// https://instill.tech/leochen5/pipelines/image-quality-assureance/playground?view=ruuafgHXWiZtTHZiHVsMiUITK1BeWIA4

func (c *ConvertInstillModel) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}
	return c.migratePipelineRelease()
}

func (c *ConvertInstillModel) getComponentTypeQuery(compType string) *gorm.DB {
	pattern := fmt.Sprintf(`type:\s+%s`, compType)
	return c.DB.Select("uid", "recipe_yaml", "recipe").
		Where("recipe_yaml ~ ?", pattern).
		Where("delete_time IS NULL")
}

func (c *ConvertInstillModel) migratePipeline() error {
	q := c.getComponentTypeQuery("instill-model")

	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return q.FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			if p.Recipe == nil {
				continue
			}

			var instillModelImageRelatedComponentNames []string
			var instillModelChatRelatedComponentNames []string

			for id, comp := range p.Recipe.Component {
				isComponentUpdated, err := c.updateInstillModelImageRelatedTasks(id, comp, &instillModelImageRelatedComponentNames)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}

				isComponentUpdated_, err := c.updateInstillModelChatRelatedTasks(id, comp, &instillModelChatRelatedComponentNames)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}

				isRecipeUpdated = isComponentUpdated || isRecipeUpdated || isComponentUpdated_
			}

			if !isRecipeUpdated {
				continue
			}

			recipeYAML, err := yaml.Marshal(p.Recipe)
			if err != nil {
				return fmt.Errorf("marshalling recipe: %w", err)
			}

			if len(instillModelImageRelatedComponentNames) == 0 && len(instillModelChatRelatedComponentNames) == 0 {
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", string(recipeYAML))
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
				continue
			}

			var updatedRecipe = string(recipeYAML)
			if len(instillModelImageRelatedComponentNames) > 0 {
				updatedRecipe, err = c.updateInstillModelImageRelatedOutput(updatedRecipe, instillModelImageRelatedComponentNames)

				if err != nil {
					l.Error("Failed to update pipeline output receiver.")
					return fmt.Errorf("updating pipeline output receiver: %w", err)
				}
			}

			if len(instillModelChatRelatedComponentNames) > 0 {
				updatedRecipe, err = c.updateInstillModelChatRelatedOutput(updatedRecipe, instillModelChatRelatedComponentNames)

				if err != nil {
					l.Error("Failed to update pipeline output receiver.")
					return fmt.Errorf("updating pipeline output receiver: %w", err)
				}
			}

			result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", updatedRecipe)
			if result.Error != nil {
				l.Error("Failed to update pipeline release.")
				return fmt.Errorf("updating pipeline recipe: %w", result.Error)
			}

		}

		return nil
	}).Error
}

func (c *ConvertInstillModel) migratePipelineRelease() error {
	q := c.getComponentTypeQuery("instill-model")

	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return q.FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", p.UID.String()))

			if p.Recipe == nil {
				continue
			}

			var instillModelImageRelatedComponentNames []string
			var instillModelChatRelatedComponentNames []string

			for id, comp := range p.Recipe.Component {
				isComponentUpdated, err := c.updateInstillModelImageRelatedTasks(id, comp, &instillModelImageRelatedComponentNames)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}

				isComponentUpdated_, err := c.updateInstillModelChatRelatedTasks(id, comp, &instillModelChatRelatedComponentNames)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}

				isRecipeUpdated = isComponentUpdated || isRecipeUpdated || isComponentUpdated_
			}

			if !isRecipeUpdated {
				continue
			}

			recipeYAML, err := yaml.Marshal(p.Recipe)
			if err != nil {
				return fmt.Errorf("marshalling recipe: %w", err)
			}

			if len(instillModelImageRelatedComponentNames) == 0 && len(instillModelChatRelatedComponentNames) == 0 {
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", string(recipeYAML))
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
				continue
			}

			var updatedRecipe = string(recipeYAML)
			if len(instillModelImageRelatedComponentNames) > 0 {
				updatedRecipe, err = c.updateInstillModelImageRelatedOutput(updatedRecipe, instillModelImageRelatedComponentNames)

				if err != nil {
					l.Error("Failed to update pipeline output receiver.")
					return fmt.Errorf("updating pipeline output receiver: %w", err)
				}
			}

			if len(instillModelChatRelatedComponentNames) > 0 {
				updatedRecipe, err = c.updateInstillModelChatRelatedOutput(updatedRecipe, instillModelChatRelatedComponentNames)

				if err != nil {
					l.Error("Failed to update pipeline output receiver.")
					return fmt.Errorf("updating pipeline output receiver: %w", err)
				}
			}

			result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", updatedRecipe)
			if result.Error != nil {
				l.Error("Failed to update pipeline release.")
				return fmt.Errorf("updating pipeline recipe: %w", result.Error)
			}
		}

		return nil
	}).Error
}

func (c *ConvertInstillModel) updateInstillModelImageRelatedTasks(compName string, comp *datamodel.Component, instillModelComponentNames *[]string) (bool, error) {

	if comp.Type == "iterator" {
		isComponentUpdated := false
		for compNameInIterator, comp := range comp.Component {
			isUpdated, err := c.updateInstillModelImageRelatedTasks(compNameInIterator, comp, instillModelComponentNames)
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

	imageRelatedTasks := []string{
		"TASK_INSTANCE_SEGMENTATION",
		"TASK_DETECTION",
		"TASK_CLASSIFICATION",
		"TASK_KEYPOINT",
		"TASK_OCR",
		"TASK_SEMANTIC_SEGMENTATION",
	}
	if slices.Contains(imageRelatedTasks, comp.Task) {

		if v, ok := input["model-name"]; ok {
			input["data"] = map[string]interface{}{
				"model": v,
			}
			delete(input, "model-name")
		}

		if v, ok := input["image-base64"]; ok {
			input["data"] = map[string]interface{}{
				"image-base64": v,
				"type":         "image-base64",
			}
			delete(input, "image-base64")
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

		if v, ok := input["system-message"]; ok {
			messages := input["data"].(map[string]interface{})["messages"].([]map[string]interface{})

			newMessage := map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"text": v,
						"type": "text",
					},
				},
				"role": "system",
			}

			messages = append(messages, newMessage)

			input["data"].(map[string]interface{})["messages"] = messages

			delete(input, "system-message")
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

	return updatedRecipe, nil
}
