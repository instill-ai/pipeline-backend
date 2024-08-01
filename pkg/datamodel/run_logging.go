package datamodel

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
)

// PipelineRun represents a single execution instance of a pipeline
type PipelineRun struct {
	BaseDynamic
	// PipelineUID is the unique identifier of the pipeline that was executed
	PipelineUID uuid.UUID `gorm:"type:uuid;index" json:"pipeline-uid"`

	// PipelineVersion is the version of the pipeline that was executed
	PipelineVersion string `gorm:"type:varchar(255)" json:"pipeline-version"`

	// Status represents the current state of the pipeline run (e.g., "Queued", "Running", "Completed", "Failed")
	Status string `gorm:"type:varchar(50);index" json:"status"`

	// Source indicates how the pipeline run was initiated (e.g., "API", "WebUI", "Scheduled")
	Source string `gorm:"type:varchar(50)" json:"source"`

	// TotalDuration is the total time taken for the pipeline run, in milliseconds
	TotalDuration int64 `gorm:"type:bigint" json:"total-duration"`

	// TriggeredBy identifies the user or system that initiated the pipeline run
	TriggeredBy string `gorm:"type:varchar(255)" json:"triggered-by"`

	// Credits represents the computational or service credits consumed by this run
	Credits int `gorm:"type:int" json:"credits"`

	// Inputs stores metadata about the input files or data for the pipeline run
	Inputs datatypes.JSON `gorm:"type:jsonb" json:"inputs"`

	// Outputs stores metadata about the output files or data produced by the pipeline run
	Outputs datatypes.JSON `gorm:"type:jsonb" json:"outputs"`

	// Components contains detailed execution information for each component in the pipeline
	Components []RunComponent `gorm:"foreignKey:RunUID" json:"components"`

	// RecipeSnapshot is a JSON representation of the pipeline configuration at the time of execution
	RecipeSnapshot datatypes.JSON `gorm:"type:jsonb" json:"recipe-snapshot"`

	// TriggeredTime is when the pipeline run was initially requested or scheduled
	TriggeredTime time.Time `gorm:"type:timestamp with time zone;not null;index" json:"triggered-time"`

	// StartedTime is when the pipeline run began execution (may be after TriggeredTime due to queueing)
	StartedTime *time.Time `gorm:"type:timestamp with time zone;index" json:"started-time,omitempty"`

	// CompletedTime is when the pipeline run finished execution (successfully or with failure)
	CompletedTime *time.Time `gorm:"type:timestamp with time zone;index" json:"completed-time,omitempty"`

	// ErrorMsg contains any error message if the pipeline run failed
	ErrorMsg string `gorm:"type:text" json:"error-msg"`
}

// RunComponent represents the execution details of a single component within a pipeline run
type RunComponent struct {
	BaseDynamic
	// RunUID references the UID of the PipelineRun this component execution belongs to
	RunUID uuid.UUID `gorm:"type:uuid;index" json:"run-uid"`

	// ComponentID is the identifier of the component within the pipeline
	ComponentID string `gorm:"type:varchar(255);index" json:"component-id"`

	// Status represents the execution status of this component (e.g., "Completed", "Failed")
	Status string `gorm:"type:varchar(50);index" json:"status"`

	// TotalDuration is the time taken for this component's execution, in milliseconds
	TotalDuration int64 `gorm:"type:bigint" json:"total-duration"`

	// StartedTime is when this component began execution
	StartedTime time.Time `gorm:"type:timestamp with time zone;index" json:"started-time"`

	// CompletedTime is when this component finished execution
	CompletedTime time.Time `gorm:"type:timestamp with time zone;index" json:"completed-time"`

	// Credits represents the computational or service credits consumed by this component
	Credits int `gorm:"type:int" json:"credits"`

	// ErrorMsg contains any error message if the component execution failed
	ErrorMsg string `gorm:"type:text" json:"error-msg"`

	// Inputs stores metadata about the input files or data for this component
	Inputs datatypes.JSON `gorm:"type:jsonb" json:"inputs"`

	// Outputs stores metadata about the output files or data produced by this component
	Outputs datatypes.JSON `gorm:"type:jsonb" json:"outputs"`
}

// FileReference represents metadata for a file, to be stored in JSON fields
type FileReference struct {
	// Name is the filename
	Name string `json:"name"`

	// Type is the MIME type of the file
	Type string `json:"type"`

	// Size is the file size in bytes
	Size int64 `json:"size"`

	// URL is the S3 or other storage URL where the file can be accessed
	URL string `json:"url"`
}

// TableName overrides the table name used by GORM
func (PipelineRun) TableName() string {
	return "pipeline_runs"
}

// TableName overrides the table name used by GORM
func (RunComponent) TableName() string {
	return "run_components"
}
