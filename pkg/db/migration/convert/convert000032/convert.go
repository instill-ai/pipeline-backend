package convert000032

import (
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
)

const batchSize = 100

type ConvertToWeb struct {
	convert.Basic
}

func (c *ConvertToWeb) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *ConvertToWeb) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			if p.Recipe != nil {
				for id, comp := range p.Recipe.Component {
					isComponentUpdated, err := c.update(comp)
					if err != nil {
						l.With(zap.String("componentID", id), zap.Error(err)).
							Error("Failed to update pipeline.")

						return fmt.Errorf("updating pipeline component: %w", err)
					}

					isRecipeUpdated = isComponentUpdated || isRecipeUpdated
				}
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

func (c *ConvertToWeb) migratePipelineRelease() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", p.UID.String()))

			if p.Recipe != nil {
				for id, comp := range p.Recipe.Component {
					isComponentUpdated, err := c.update(comp)
					if err != nil {
						l.With(zap.String("componentID", id), zap.Error(err)).
							Error("Failed to update pipeline.")

						return fmt.Errorf("updating pipeline component: %w", err)
					}

					isRecipeUpdated = isComponentUpdated || isRecipeUpdated
				}
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

func (c *ConvertToWeb) update(comp *datamodel.Component) (bool, error) {

	if comp.Type == "iterator" {
		isComponentUpdated := false
		for _, comp := range comp.Component {
			isUpdated, err := c.update(comp)
			if err != nil {
				return false, err
			}
			isComponentUpdated = isUpdated || isComponentUpdated
		}
		return isComponentUpdated, nil
	}

	if comp.Type != "web" {
		return false, nil
	}

	if comp.Task != "TASK_CRAWL_WEBSITE" && comp.Task != "TASK_SCRAPE_WEBPAGE" {
		return false, nil
	}

	if comp.Task == "TASK_CRAWL_WEBSITE" {
		comp.Task = "TASK_CRAWL_SITE"
		toRemoveFields := []string{
			"include-link-html",
			"include-link-text",
			"only-main-content",
			"remove-tags",
			"only-include-tags",
		}

		input, isMap := comp.Input.(map[string]interface{})

		if !isMap {
			return false, nil
		}

		for _, field := range toRemoveFields {
			delete(input, field)
		}

		if v, ok := input["target-url"]; ok {
			input["url"] = v
			delete(input, "target-url")
		}
		return true, nil
	}

	if comp.Task == "TASK_SCRAPE_WEBPAGE" {
		comp.Task = "TASK_SCRAPE_PAGE"
		return true, nil
	}

	return false, nil

}
