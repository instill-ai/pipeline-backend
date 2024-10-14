package convert000032

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

const batchSize = 100

type ConvertToWeb struct {
	DB     *gorm.DB
	Logger *zap.Logger
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
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			recipeYAML := p.RecipeYAML

			updatedRecipeYAML, isUpdate := c.updateRecipeYAML(recipeYAML)

			l.Info("Updated Recipe YAML: \n", zap.String("updatedRecipeYAML", updatedRecipeYAML))

			l.Info("Is Update: ", zap.Bool("isUpdate", isUpdate))

			if isUpdate {
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", updatedRecipeYAML)
				l.Info("=== Updating Pipeline ===")

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
	pipelineReleases := make([]*datamodel.PipelineRelease, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelineReleases, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelineReleases {
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			recipeYAML := p.RecipeYAML

			updatedRecipeYAML, isUpdate := c.updateRecipeYAML(recipeYAML)

			if isUpdate {
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", updatedRecipeYAML)
				l.Info("=== Updating PipelineRelease ===")
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *ConvertToWeb) updateRecipeYAML(recipeYAML string) (updatedRecipeYAML string, isUpdate bool) {

	if !strings.Contains(recipeYAML, "TASK_CRAWL_WEBSITE") && !strings.Contains(recipeYAML, "TASK_SCRAPE_WEBPAGE") {
		return "", false
	}

	if strings.Contains(recipeYAML, "TASK_CRAWL_WEBSITE") {
		// Insert the description of the recipe needs to be updated in the top of the YAML file
		updatedRecipeYAML = fmt.Sprintf("# Recipe updated to replace TASK_CRAWL_WEBSITE with TASK_CRAWL_SITE and deprecated some fields. \n# Please check the latest document to update your recipe.\n%s", recipeYAML)

		updatedRecipeYAML = strings.ReplaceAll(updatedRecipeYAML, "TASK_CRAWL_WEBSITE", "TASK_CRAWL_SITE")
		isUpdate = true
	}

	if strings.Contains(recipeYAML, "TASK_SCRAPE_WEBPAGE") {
		updatedRecipeYAML = strings.ReplaceAll(recipeYAML, "TASK_SCRAPE_WEBPAGE", "TASK_SCRAPE_PAGE")
		isUpdate = true
	}

	return updatedRecipeYAML, isUpdate
}
