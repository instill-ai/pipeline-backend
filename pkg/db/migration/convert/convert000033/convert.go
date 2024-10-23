package convert000033

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
)

const batchSize = 100

type ConvertWebFields struct {
	convert.Basic
}

var webCompNames []string

func (c *ConvertWebFields) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	if err := c.migratePipelineRelease(); err != nil {
		return err
	}

	if len(webCompNames) > 0 {
		return fmt.Errorf("web components' output are not updated: %v", webCompNames)
	}

	return nil
}

func (c *ConvertWebFields) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			if p.Recipe != nil {
				for id, comp := range p.Recipe.Component {
					isComponentUpdated, err := c.updateWebInput(id, comp)
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

				updatedRecipe, err := c.updateWebOutputReceiver(recipeYAML)

				if err != nil {
					return fmt.Errorf("updating pipeline output receiver: %w", err)
				}

				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", updatedRecipe)
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *ConvertWebFields) migratePipelineRelease() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml", "recipe").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", p.UID.String()))

			if p.Recipe != nil {
				for id, comp := range p.Recipe.Component {
					isComponentUpdated, err := c.updateWebInput(id, comp)
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

				updatedRecipe, err := c.updateWebOutputReceiver(recipeYAML)

				if err != nil {
					return fmt.Errorf("updating pipeline output receiver: %w", err)
				}

				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", updatedRecipe)
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *ConvertWebFields) updateWebInput(compName string, comp *datamodel.Component) (bool, error) {

	if comp.Type == "iterator" {
		isComponentUpdated := false
		for _, comp := range comp.Component {
			isUpdated, err := c.updateWebInput(compName, comp)
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

	if comp.Task != "TASK_SCRAPE_PAGE" {
		return false, nil
	}

	comp.Task = "TASK_SCRAPE_PAGES"

	input, isMap := comp.Input.(map[string]interface{})

	if !isMap {
		return false, nil
	}

	if v, ok := input["url"]; ok {
		input["urls"] = []string{v.(string)}
		delete(input, "url")
	}

	webCompNames = append(webCompNames, compName)

	return true, nil
}

func (c *ConvertWebFields) updateWebOutputReceiver(recipeYAML []byte) (string, error) {
	originalFields := []string{
		"content",
		"markdown",
		"html",
		"metadata",
		"links-on-page",
	}
	recipeString := string(recipeYAML)

	updatedRecipe := recipeString
	for _, compName := range webCompNames {
		for _, field := range originalFields {
			// Don't have to add `}` because it could be `links-on-page[0]`
			// It will be wrong if we add `}` at the end
			originalWebOutput := fmt.Sprintf("${%s.output.%s", compName, field)
			if !strings.Contains(recipeString, originalWebOutput) {
				continue
			}
			updatedRecipe = strings.ReplaceAll(updatedRecipe, originalWebOutput, fmt.Sprintf("${%s.output.pages[0].%s", compName, field))
		}
	}

	// Need to empty the slice after processing
	webCompNames = []string{}

	return updatedRecipe, nil
}
