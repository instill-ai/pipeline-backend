package convert000040

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

type RenameFormatToType struct {
	convert.Basic
}

func (c *RenameFormatToType) Migrate() error {
	if err := c.migratePipeline(); err != nil {
		return err
	}

	return c.migratePipelineRelease()
}

func (c *RenameFormatToType) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			var node yaml.Node
			if p.Recipe != nil {
				// Update recipe_yaml content
				if err := yaml.Unmarshal([]byte(p.RecipeYAML), &node); err != nil {
					return fmt.Errorf("unmarshaling recipe yaml: %w", err)
				}

				// Find and update the variable section
				if len(node.Content) > 0 && len(node.Content[0].Content) > 0 {
					for i := 0; i < len(node.Content[0].Content); i += 2 {
						if node.Content[0].Content[i].Value == "variable" {
							variableNode := node.Content[0].Content[i+1]
							for j := 0; j < len(variableNode.Content); j += 2 {
								varContent := variableNode.Content[j+1]
								for k := 0; k < len(varContent.Content); k += 2 {
									if varContent.Content[k].Value == "format" {
										varContent.Content[k].Value = "type"
										isRecipeUpdated = true
									}
								}
							}
							break
						}
					}
				}
			}

			if isRecipeUpdated {
				buf := bytes.Buffer{}
				encoder := yaml.NewEncoder(&buf)
				encoder.SetIndent(2)
				err := encoder.Encode(&node.Content[0])
				if err != nil {
					return fmt.Errorf("marshalling recipe: %w", err)
				}
				recipeYAML := buf.String()
				result := tx.Model(p).Where("uid = ?", p.UID).Update("recipe_yaml", recipeYAML)
				if result.Error != nil {
					l.Error("Failed to update pipeline.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *RenameFormatToType) migratePipelineRelease() error {
	releases := make([]*datamodel.PipelineRelease, 0, batchSize)
	return c.DB.Select("uid", "recipe_yaml").FindInBatches(&releases, batchSize, func(tx *gorm.DB, _ int) error {
		for _, r := range releases {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", r.UID.String()))

			var node yaml.Node
			if r.Recipe != nil {
				// Update recipe_yaml content
				if err := yaml.Unmarshal([]byte(r.RecipeYAML), &node); err != nil {
					return fmt.Errorf("unmarshaling recipe yaml: %w", err)
				}

				// Find and update the variable section
				if len(node.Content) > 0 && len(node.Content[0].Content) > 0 {
					for i := 0; i < len(node.Content[0].Content); i += 2 {
						if node.Content[0].Content[i].Value == "variable" {
							variableNode := node.Content[0].Content[i+1]
							for j := 0; j < len(variableNode.Content); j += 2 {
								varContent := variableNode.Content[j+1]
								for k := 0; k < len(varContent.Content); k += 2 {
									if varContent.Content[k].Value == "format" {
										varContent.Content[k].Value = "type"
										isRecipeUpdated = true
									}
								}
							}
							break
						}
					}
				}
			}

			if isRecipeUpdated {
				buf := bytes.Buffer{}
				encoder := yaml.NewEncoder(&buf)
				encoder.SetIndent(2)
				err := encoder.Encode(&node.Content[0])
				if err != nil {
					return fmt.Errorf("marshalling recipe: %w", err)
				}
				recipeYAML := buf.String()
				result := tx.Model(r).Where("uid = ?", r.UID).Update("recipe_yaml", recipeYAML)
				if result.Error != nil {
					l.Error("Failed to update pipeline release.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}
