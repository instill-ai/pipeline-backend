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

func (c *RenameHTTPComponent) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *RenameHTTPComponent) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
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
	pipelineReleases := make([]*datamodel.PipelineRelease, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml").FindInBatches(&pipelineReleases, batchSize, func(tx *gorm.DB, _ int) error {
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

		if isIterator {
			// If it's an iterator, find and process its nested components
			for k := 0; k < len(compContent.Content); k += 2 {
				if compContent.Content[k].Value == "component" {
					isComponentUpdated, err := c.updateComponentNode(compContent.Content[k+1])
					if err != nil {
						return false, fmt.Errorf("updating iterator component: %w", err)
					}
					isUpdated = isComponentUpdated || isUpdated
				}
			}
		} else {
			// Regular component, check and update type if needed
			for k := 0; k < len(compContent.Content); k += 2 {
				if compContent.Content[k].Value == "type" && compContent.Content[k+1].Value == "restapi" {
					compContent.Content[k+1].Value = "http"
					isUpdated = true
				}
			}
		}
	}
	return isUpdated, nil
}
