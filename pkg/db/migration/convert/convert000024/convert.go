package convert000024

import (
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
)

const batchSize = 100

type ConvertToTextTaskConverter struct {
	convert.Basic
}

func (c *ConvertToTextTaskConverter) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *ConvertToTextTaskConverter) migratePipeline() error {
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

func (c *ConvertToTextTaskConverter) migratePipelineRelease() error {
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

func (c *ConvertToTextTaskConverter) updateTask(comp *datamodel.Component) (bool, error) {
	mdConverted, err := c.updateDocToDocument(comp)
	if err != nil {
		return false, err
	}

	return mdConverted, nil
}

func (c *ConvertToTextTaskConverter) updateDocToDocument(comp *datamodel.Component) (bool, error) {
	if comp.Type == "iterator" {
		isComponentUpdated := false
		for _, comp := range comp.Component {
			isSubComponentUpdated, err := c.updateDocToDocument(comp)
			if err != nil {
				return false, fmt.Errorf("updating iterator component: %w", err)
			}

			isComponentUpdated = isSubComponentUpdated || isComponentUpdated
		}

		return isComponentUpdated, nil
	}

	if comp.Type != "document" || comp.Task != "TASK_CONVERT_TO_TEXT" {
		return false, nil
	}

	in, isMap := comp.Input.(map[string]any)
	if !isMap {
		return false, fmt.Errorf("invalid input type on TASK_CONVERT_TO_TEXT")
	}

	if v, ok := in["doc"]; ok {
		in["document"] = v
		delete(in, "doc")
	}

	comp.Input = in

	return true, nil
}
