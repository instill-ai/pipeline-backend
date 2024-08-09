package convert000022

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const batchSize = 100

type ConvertWebsiteToWebConverter struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

func (c *ConvertWebsiteToWebConverter) Migrate() error {

	if err := c.migratePipeline(); err != nil {
		return err
	}

	return nil
}

func (c *ConvertWebsiteToWebConverter) migratePipeline() error {
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

func (c *ConvertWebsiteToWebConverter) updateTask(comp *datamodel.Component) (bool, error) {

	isConverted, err := c.convertTypeAndTask(comp)
	if err != nil {
		return false, fmt.Errorf("converting type from website to web and task from TASK_SCRAPE_WEBSITE to TASK_CRAWL_WEBSITE: %w", err)
	}

	return isConverted, nil
}

func (c *ConvertWebsiteToWebConverter) convertTypeAndTask(comp *datamodel.Component) (bool, error) {
	if comp.Type == "iterator" {
		isComponentUpdated := false
		for _, comp := range comp.Component {
			isSubComponentUpdated, err := c.convertTypeAndTask(comp)
			if err != nil {
				return false, fmt.Errorf("updating iterator component: %w", err)
			}
			isComponentUpdated = isSubComponentUpdated || isComponentUpdated
		}

		return isComponentUpdated, nil
	}
	isTypeUpdated := false
	if comp.Type == "website" {
		isTypeUpdated = true
		comp.Type = "web"
	}

	isTaskUpdated := false
	if comp.Task == "TASK_SCRAPE_WEBSITE" {
		isTaskUpdated = true
		comp.Task = "TASK_CRAWL_WEBSITE"
	}

	return isTypeUpdated || isTaskUpdated, nil

}
