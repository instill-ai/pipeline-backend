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

func (c *ConvertWebFields) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}
	return c.migratePipelineRelease()
}

func (c *ConvertWebFields) getComponentTypeQuery(compType string) *gorm.DB {
	pattern := fmt.Sprintf(`type:\s+%s`, compType)
	return c.DB.Select("uid", "recipe_yaml", "recipe").
		Where("recipe_yaml ~ ?", pattern).
		Where("delete_time IS NULL")
}

func (c *ConvertWebFields) migratePipeline() error {
	q := c.getComponentTypeQuery("web")

	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return q.FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			if p.Recipe == nil {
				continue
			}

			var webComponentNames []string

			for id, comp := range p.Recipe.Component {
				isComponentUpdated, err := c.updateWebInput(id, comp, &webComponentNames)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}

				isRecipeUpdated = isComponentUpdated || isRecipeUpdated
			}

			if !isRecipeUpdated {
				continue
			}

			recipeYAML, err := yaml.Marshal(p.Recipe)
			if err != nil {
				return fmt.Errorf("marshalling recipe: %w", err)
			}

			if len(webComponentNames) == 0 {
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", string(recipeYAML))
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
				continue
			}

			updatedRecipe, err := c.updateWebOutputReceiver(recipeYAML, webComponentNames)

			if err != nil {
				return fmt.Errorf("updating pipeline output receiver: %w", err)
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

func (c *ConvertWebFields) migratePipelineRelease() error {
	q := c.getComponentTypeQuery("web")

	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return q.FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", p.UID.String()))

			if p.Recipe == nil {
				continue
			}

			var webComponentNames []string

			for id, comp := range p.Recipe.Component {
				isComponentUpdated, err := c.updateWebInput(id, comp, &webComponentNames)
				if err != nil {
					l.With(zap.String("componentID", id), zap.Error(err)).
						Error("Failed to update pipeline.")

					return fmt.Errorf("updating pipeline component: %w", err)
				}
				isRecipeUpdated = isComponentUpdated || isRecipeUpdated
			}

			if !isRecipeUpdated {
				continue
			}

			recipeYAML, err := yaml.Marshal(p.Recipe)
			if err != nil {
				return fmt.Errorf("marshalling recipe: %w", err)
			}

			if len(webComponentNames) == 0 {
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", string(recipeYAML))
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
				continue
			}

			updatedRecipe, err := c.updateWebOutputReceiver(recipeYAML, webComponentNames)

			if err != nil {
				return fmt.Errorf("updating pipeline output receiver: %w", err)
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

func (c *ConvertWebFields) updateWebInput(compName string, comp *datamodel.Component, webComponentNames *[]string) (bool, error) {

	if comp.Type == "iterator" {
		isComponentUpdated := false
		for compNameInIterator, comp := range comp.Component {
			isUpdated, err := c.updateWebInput(compNameInIterator, comp, webComponentNames)
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

	timeout, timeoutFound := input["timeout"]

	timeoutType := fmt.Sprintf("%T", timeout)

	if timeoutFound && timeout != nil && timeoutType == "int" && timeout.(int) > 0 {
		input["scrape-method"] = "chrome-simulator"
	}

	_, scrapeMethodFound := input["scrape-method"]

	if !scrapeMethodFound {
		input["scrape-method"] = "http"
	}

	*webComponentNames = append(*webComponentNames, compName)

	return true, nil
}

func (c *ConvertWebFields) updateWebOutputReceiver(recipeYAML []byte, webComponentNames []string) (string, error) {
	originalFields := []string{
		"content",
		"markdown",
		"html",
		"metadata",
		"links-on-page",
	}
	recipeString := string(recipeYAML)

	updatedRecipe := recipeString
	for _, compName := range webComponentNames {
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

	return updatedRecipe, nil
}
