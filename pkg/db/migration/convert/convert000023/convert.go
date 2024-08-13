package convert000023

import (
	"fmt"
	"log"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const batchSize = 100

type TextFieldsConverter struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

func (c *TextFieldsConverter) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *TextFieldsConverter) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)

	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			for id, comp := range p.Recipe.Component {
				isComponentUpdated, err := c.updateTask(comp)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}

				isRecipeUpdated = isComponentUpdated || isRecipeUpdated
			}

			if isRecipeUpdated {
				recipeYAML, err := yaml.Marshal(p.Recipe)
				if err != nil {
					return fmt.Errorf("marshalling recipe: %w", err)
				}
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

func (c *TextFieldsConverter) migratePipelineRelease() error {
	pipelineReleases := make([]*datamodel.PipelineRelease, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelineReleases, batchSize, func(tx *gorm.DB, _ int) error {
		for _, pr := range pipelineReleases {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", pr.UID.String()))

			for id, comp := range pr.Recipe.Component {
				isComponentUpdated, err := c.updateTask(comp)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline release.")

					return fmt.Errorf("updating pipeline release component: %w", err)
				}

				isRecipeUpdated = isComponentUpdated || isRecipeUpdated
			}

			if isRecipeUpdated {
				recipeYAML, err := yaml.Marshal(pr.Recipe)
				if err != nil {
					return fmt.Errorf("marshalling recipe: %w", err)
				}

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

func (c *TextFieldsConverter) updateTask(comp *datamodel.Component) (bool, error) {

	isConverted, err := c.convertField(comp)
	if err != nil {
		return false, fmt.Errorf("converting type from website to web and task from TASK_SCRAPE_WEBSITE to TASK_CRAWL_WEBSITE: %w", err)
	}

	return isConverted, nil
}

func (c *TextFieldsConverter) convertField(comp *datamodel.Component) (bool, error) {

	if comp.Type == "iterator" {
		isComponentUpdated := false
		for _, comp := range comp.Component {
			isSubComponentUpdated, err := c.convertField(comp)
			if err != nil {
				return false, fmt.Errorf("updating iterator component: %w", err)
			}
			isComponentUpdated = isSubComponentUpdated || isComponentUpdated
		}

		return isComponentUpdated, nil
	}

	if !(comp.Type == "text" && comp.Task == "TASK_CHUNK_TEXT") {
		return false, nil
	}

	in, isMap := comp.Input.(map[string]any)

	if !isMap {
		log.Println("Invalid input type on TASK_CHUNK_TEXT")
		return false, nil
	}

	strategyMap, ok := in["strategy"].(map[string]interface{})
	if !ok {
		log.Println("Missing field strategy on TASK_CHUNK_TEXT")
		return false, nil
	}

	settingMap, ok := strategyMap["setting"].(map[string]interface{})
	if !ok {
		log.Println("Missing field setting on TASK_CHUNK_TEXT")
		return false, nil
	}

	chunkMethod, ok := settingMap["chunk-method"].(string)
	if !ok {
		log.Println("Missing field chunk-method on TASK_CHUNK_TEXT")
		return false, nil
	}

	modelName, ok := settingMap["model-name"].(string)

	if !ok {
		log.Println("Missing field model-name on TASK_CHUNK_TEXT")
		return false, nil
	}

	if !(chunkMethod == "Recursive" || chunkMethod == "Markdown") {
		return false, nil
	}

	tokenization := map[string]interface{}{
		"choice": map[string]interface{}{
			"model":               modelName,
			"tokenization-method": "Model",
		},
	}

	in["tokenization"] = tokenization
	delete(settingMap, "model-name")

	return true, nil

}
