package convert000035

import (
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
)

const batchSize = 100

type RenameInstillFormat struct {
	convert.Basic
}

func (c *RenameInstillFormat) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *RenameInstillFormat) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			var node yaml.Node
			if p.Recipe != nil {
				// Update recipe_yaml content
				if err := yaml.Unmarshal([]byte(p.RecipeYAML), &node); err != nil {
					return fmt.Errorf("unmarshalling recipe yaml: %w", err)
				}

				// Find and update the variable section
				for i := 0; i < len(node.Content[0].Content); i += 2 {
					if node.Content[0].Content[i].Value == "variable" {
						variableNode := node.Content[0].Content[i+1]
						for j := 0; j < len(variableNode.Content); j += 2 {
							varContent := variableNode.Content[j+1]
							for k := 0; k < len(varContent.Content); k += 2 {
								if varContent.Content[k].Value == "instill-format" {
									varContent.Content[k].Value = "format"
									isRecipeUpdated = true
								}
							}
						}
						break
					}
				}

			}

			if isRecipeUpdated {
				recipeYAML, err := yaml.Marshal(&node.Content[0])
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

func (c *RenameInstillFormat) migratePipelineRelease() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", p.UID.String()))

			var node yaml.Node
			if p.Recipe != nil {
				// Update recipe_yaml content
				if err := yaml.Unmarshal([]byte(p.RecipeYAML), &node); err != nil {
					return fmt.Errorf("unmarshalling recipe yaml: %w", err)
				}

				// Find and update the variable section
				for i := 0; i < len(node.Content[0].Content); i += 2 {
					if node.Content[0].Content[i].Value == "variable" {
						variableNode := node.Content[0].Content[i+1]
						for j := 0; j < len(variableNode.Content); j += 2 {
							varContent := variableNode.Content[j+1]
							for k := 0; k < len(varContent.Content); k += 2 {
								if varContent.Content[k].Value == "instill-format" {
									varContent.Content[k].Value = "format"
									isRecipeUpdated = true
								}
							}
						}
						break
					}
				}

			}

			if isRecipeUpdated {
				recipeYAML, err := yaml.Marshal(&node.Content[0])
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
