package pipelinelogger

import (
	"context"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

type PipelineLogger struct {
	db *gorm.DB
}

// NewPipelineLogger creates a new instance of PipelineLogger
func NewPipelineLogger(db *gorm.DB) *PipelineLogger {
	return &PipelineLogger{
		db: db,
	}
}

// LogPipelineRun creates or updates a pipeline run log
func (l *PipelineLogger) LogPipelineRun(ctx context.Context, runData *datamodel.PipelineRun) (uuid.UUID, error) {
	if runData.UID == uuid.Nil {
		// New run, create it
		return runData.UID, l.db.Create(runData).Error
	}

	// Existing run, update it
	return runData.UID, l.db.Model(runData).Updates(runData).Error
}

// LogComponentRun creates or updates a component run log
func (l *PipelineLogger) LogComponentRun(ctx context.Context, componentData *datamodel.RunComponent) error {
	if componentData.UID == uuid.Nil {
		// New component run, create it
		return l.db.Create(componentData).Error
	}

	// Existing component run, update it
	return l.db.Model(componentData).Updates(componentData).Error
}

// GetPipelineRun retrieves a pipeline run by its UID
func (l *PipelineLogger) GetPipelineRun(ctx context.Context, runUID uuid.UUID) (*datamodel.PipelineRun, error) {
	var run datamodel.PipelineRun
	result := l.db.Preload("Components").First(&run, "uid = ?", runUID)
	if result.Error != nil {
		return nil, result.Error
	}
	return &run, nil
}

// ListPipelineRuns retrieves a list of pipeline runs with pagination
func (l *PipelineLogger) ListPipelineRuns(ctx context.Context, pipelineUID uuid.UUID, limit int, offset int) ([]datamodel.PipelineRun, error) {
	var runs []datamodel.PipelineRun
	result := l.db.Where("pipeline_uid = ?", pipelineUID).
		Order("triggered_time DESC").
		Limit(limit).
		Offset(offset).
		Find(&runs)
	if result.Error != nil {
		return nil, result.Error
	}
	return runs, nil
}
