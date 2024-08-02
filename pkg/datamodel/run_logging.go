package datamodel

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
)

// PipelineRun represents a single execution instance of a pipeline
type PipelineRun struct {
	BaseDynamic
	PipelineUID     uuid.UUID       `gorm:"type:uuid;index" json:"pipeline-uid"`
	PipelineVersion string          `gorm:"type:varchar(255)" json:"pipeline-version"`
	Status          string          `gorm:"type:varchar(50);index" json:"status"`
	Source          string          `gorm:"type:varchar(50)" json:"source"`
	TotalDuration   int64           `gorm:"type:bigint" json:"total-duration"`
	TriggeredBy     string          `gorm:"type:varchar(255)" json:"triggered-by"`
	Credits         int             `gorm:"type:int" json:"credits"`
	Inputs          []FileReference `gorm:"type:jsonb" json:"inputs"`
	Outputs         []FileReference `gorm:"type:jsonb" json:"outputs"`
	Components      []RunComponent  `gorm:"foreignKey:RunUID" json:"components"`
	RecipeSnapshot  datatypes.JSON  `gorm:"type:jsonb" json:"recipe-snapshot"`
	TriggeredTime   time.Time       `gorm:"type:timestamp with time zone;not null;index" json:"triggered-time"`
	StartedTime     *time.Time      `gorm:"type:timestamp with time zone;index" json:"started-time,omitempty"`
	CompletedTime   *time.Time      `gorm:"type:timestamp with time zone;index" json:"completed-time,omitempty"`
	ErrorMsg        string          `gorm:"type:text" json:"error-msg"`
}

// RunComponent represents the execution details of a single component within a pipeline run
type RunComponent struct {
	BaseDynamic
	RunUID        uuid.UUID       `gorm:"type:uuid;index" json:"run-uid"`
	ComponentID   string          `gorm:"type:varchar(255);index" json:"component-id"`
	Status        string          `gorm:"type:varchar(50);index" json:"status"`
	TotalDuration int64           `gorm:"type:bigint" json:"total-duration"`
	StartedTime   time.Time       `gorm:"type:timestamp with time zone;index" json:"started-time"`
	CompletedTime time.Time       `gorm:"type:timestamp with time zone;index" json:"completed-time"`
	Credits       int             `gorm:"type:int" json:"credits"`
	ErrorMsg      string          `gorm:"type:text" json:"error-msg"`
	Inputs        []FileReference `gorm:"type:jsonb" json:"inputs"`
	Outputs       []FileReference `gorm:"type:jsonb" json:"outputs"`
}

// FileReference represents metadata for a file, to be stored in JSON fields
type FileReference struct {
	Name string `gorm:"type:varchar(255)" json:"name"`
	Type string `gorm:"type:varchar(255)" json:"type"`
	Size int64  `gorm:"type:bigint" json:"size"`
	URL  string `gorm:"type:text" json:"url"`
}

// TableName overrides the table name used by GORM
func (PipelineRun) TableName() string {
	return "pipeline_runs"
}

// TableName overrides the table name used by GORM
func (RunComponent) TableName() string {
	return "run_components"
}
