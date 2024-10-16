package convert000031

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
)

const batchSize = 100

// SlackSetupConverter executes code along with the 19th database
// schema revision.
type SlackSetupConverter struct {
	convert.Basic
}

// Migrate updates the `TASK_JQ` input in the JSON operator to kebab-case.
func (c *SlackSetupConverter) Migrate() error {
	if err := c.migrateConnection(); err != nil {
		return fmt.Errorf("migrating connections: %w", err)
	}

	if err := c.migratePipeline(); err != nil {
		return fmt.Errorf("migrating pipelines: %w", err)
	}

	if err := c.migratePipelineRelease(); err != nil {
		return fmt.Errorf("migrating pipeline releases: %w", err)
	}

	return nil
}

func (c *SlackSetupConverter) migrateConnection() error {
	// fetch slack component ID
	cds := componentstore.Init(c.Logger, nil, nil)
	cd, err := cds.GetDefinitionByID("slack", nil, nil)
	if err != nil {
		return fmt.Errorf("fetching slack component UID")
	}

	q := c.DB.Select("uid", "setup").
		Where("integration_uid = ?", cd.Uid).
		Where("delete_time IS NULL")

	connections := make([]*datamodel.Connection, 0, batchSize)
	return q.FindInBatches(&connections, batchSize, func(tx *gorm.DB, _ int) error {
		for _, conn := range connections {
			l := c.Logger.With(zap.String("connectionUID", conn.UID.String()))

			var setup map[string]any
			if err := json.Unmarshal(conn.Setup, &setup); err != nil {
				return fmt.Errorf("unmarshalling setup: %w", err)
			}

			updated := c.updateToken(setup)
			if !updated {
				continue
			}

			j, err := json.Marshal(setup)
			if err != nil {
				return fmt.Errorf("marshalling setup: %w", err)
			}

			result := tx.Model(conn).Where("uid = ?", conn.UID).Update("setup", datatypes.JSON(j))
			if result.Error != nil {
				l.Error("Failed to update connection setup.")
				return fmt.Errorf("updating connection setup: %w", result.Error)
			}
		}

		return nil
	}).Error
}

func (c *SlackSetupConverter) migratePipeline() error {
	q := c.DB.Select("uid", "recipe_yaml", "recipe").
		Where("recipe_yaml LIKE ?", "%%type: slack%%").
		Where("delete_time IS NULL")

	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return q.FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))

			if p.Recipe != nil {
				for id, comp := range p.Recipe.Component {
					isComponentUpdated, err := c.updateSlackSetup(comp)
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
					l.Error("Failed to update pipeline recipe.")
					return fmt.Errorf("updating pipeline recipe: %w", result.Error)
				}
			}
		}

		return nil
	}).Error
}

func (c *SlackSetupConverter) migratePipelineRelease() error {
	q := c.DB.Select("uid", "recipe_yaml", "recipe").
		Where("recipe_yaml LIKE ?", "%%type: slack%%").
		Where("delete_time IS NULL")

	pipelineReleases := make([]*datamodel.PipelineRelease, 0, batchSize)
	return q.FindInBatches(&pipelineReleases, batchSize, func(tx *gorm.DB, _ int) error {
		for _, pr := range pipelineReleases {
			isRecipeUpdated := false
			l := c.Logger.With(zap.String("pipelineReleaseUID", pr.UID.String()))

			if pr.Recipe != nil {
				for id, comp := range pr.Recipe.Component {
					isComponentUpdated, err := c.updateSlackSetup(comp)
					if err != nil {
						l.With(zap.String("componentID", id), zap.Error(err)).
							Error("Failed to update pipeline release.")

						return fmt.Errorf("updating pipeline release component: %w", err)
					}

					isRecipeUpdated = isComponentUpdated || isRecipeUpdated
				}
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

func (c *SlackSetupConverter) updateSlackSetup(comp *datamodel.Component) (bool, error) {
	if comp.Type == "iterator" {
		isComponentUpdated := false
		for _, comp := range comp.Component {
			isSubComponentUpdated, err := c.updateSlackSetup(comp)
			if err != nil {
				return false, fmt.Errorf("updating iterator component: %w", err)
			}

			isComponentUpdated = isSubComponentUpdated || isComponentUpdated
		}

		return isComponentUpdated, nil
	}

	if comp.Type != "slack" {
		return false, nil
	}

	setup, isMap := comp.Setup.(map[string]any)
	if !isMap {
		// Component setup references a connection.
		return false, nil
	}

	if updated := c.updateToken(setup); !updated {
		return false, nil
	}

	comp.Setup = setup
	return true, nil
}

func (c *SlackSetupConverter) updateToken(setup map[string]any) (updated bool) {
	if _, hasToken := setup["token"]; !hasToken {
		return false
	}

	setup["bot-token"] = setup["token"]
	delete(setup, "token")

	return true
}
