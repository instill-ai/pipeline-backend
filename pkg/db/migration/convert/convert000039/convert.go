package convert000039

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
	"github.com/instill-ai/pipeline-backend/pkg/service"
)

const batchSize = 100

// AddExpirationDateToRuns ...
type AddExpirationDateToRuns struct {
	convert.Basic
	RetentionHandler service.MetadataRetentionHandler
}

// Migrate ...
func (c *AddExpirationDateToRuns) Migrate() error {
	return c.migratePipelineAndComponentRuns()
}

type pipelineRun struct {
	RequesterUID uuid.UUID `gorm:"type:uuid;primary_key;<-:create"`
}

func (pipelineRun) TableName() string {
	return "pipeline_run"
}

func (c *AddExpirationDateToRuns) migratePipelineAndComponentRuns() error {
	c.DB = c.DB.Debug()

	pipelineRuns := make([]*pipelineRun, 0, batchSize)
	return c.DB.Distinct("requester_uid").FindInBatches(&pipelineRuns, batchSize, func(tx *gorm.DB, _ int) error {
		for _, pr := range pipelineRuns {
			log := c.Logger.With(zap.String("requesterUID", pr.RequesterUID.String()))

			expiryRule, err := c.RetentionHandler.GetExpiryRuleByNamespace(context.Background(), pr.RequesterUID)
			if err != nil {
				log.Error("Failed to fetch expiry rule", zap.Error(err))
				return fmt.Errorf("fetching expiry rule: %w", err)
			}

			if expiryRule.ExpirationDays <= 0 {
				// Infinite expiration, blob expiration time should be NULL.
				continue
			}

			// No record with primary key specified, so the update will be done
			// in batch.
			err = c.DB.Model(&datamodel.PipelineRun{}).
				Where("blob_data_expiration_time IS ?", nil).
				Where("requester_uid = ?", pr.RequesterUID).
				Update(
					"blob_data_expiration_time",
					gorm.Expr("started_time + make_interval(days => ?)", expiryRule.ExpirationDays),
				).Error
			if err != nil {
				log.Error("Failed to update pipeline runs", zap.Error(err))
				return fmt.Errorf("updating pipeline runs: %w", err)
			}

			err = c.DB.Exec(
				`UPDATE "component_run"
					SET "blob_data_expiration_time"=component_run.started_time + make_interval(days => ?)
					FROM "pipeline_run"
					WHERE component_run.blob_data_expiration_time IS NULL
					AND pipeline_run.requester_uid = ?`,
				expiryRule.ExpirationDays,
				pr.RequesterUID,
			).Error
			if err != nil {
				log.Error("Failed to update component runs", zap.Error(err))
				return fmt.Errorf("updating component runs: %w", err)
			}
		}

		return nil
	}).Error
}
