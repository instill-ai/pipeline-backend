package datamodel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gopkg.in/guregu/null.v4"

	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
)

// for saving the protobuf types as string values
type (
	RunStatus runpb.RunStatus
	RunSource runpb.RunSource
)

func (v *RunStatus) Scan(value any) error {
	*v = RunStatus(runpb.RunStatus_value[value.(string)])
	return nil
}

func (v RunStatus) Value() (driver.Value, error) {
	return runpb.RunStatus(v).String(), nil
}

func (v *RunSource) Scan(value any) error {
	*v = RunSource(runpb.RunSource_value[value.(string)])
	return nil
}

func (v RunSource) Value() (driver.Value, error) {
	return runpb.RunSource(v).String(), nil
}

// FileReference represents metadata for a file, to be stored in JSON fields.
type FileReference struct {
	Name string `gorm:"type:varchar(255)" json:"name"` // Name of the file
	Type string `gorm:"type:varchar(255)" json:"type"` // Format of the file (e.g., PDF, TXT, JPG)
	Size int64  `gorm:"type:bigint" json:"size"`       // Size of the file in bytes
	URL  string `gorm:"type:text" json:"url"`          // URL of the file (e.g., S3 URL)
}

// JSONB is a custom data type to handle JSONB fields in PostgreSQL.
type JSONB []FileReference

// Value marshals the JSONB to a value.
func (j JSONB) Value() (driver.Value, error) {
	value, err := json.Marshal(j)
	return string(value), err
}

// Scan unmarshals a value into the JSONB.
func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// PipelineRun represents the metadata and execution details for each pipeline run.
// todo: use type UUID for TriggeredBy and Namespace, rename Namespace --> NamespaceUID
type PipelineRun struct {
	PipelineTriggerUID uuid.UUID      `gorm:"primaryKey" json:"pipeline-trigger-uid"`                                        // Unique identifier for each run
	PipelineUID        uuid.UUID      `gorm:"type:uuid;index" json:"pipeline-uid"`                                           // Pipeline unique ID used in the run
	Pipeline           Pipeline       `gorm:"foreignKey:PipelineUID;references:UID"`                                         // Pipeline instance referenced in the run
	PipelineVersion    string         `gorm:"type:varchar(255)" json:"pipeline-version"`                                     // Pipeline version used in the run
	Status             RunStatus      `gorm:"type:valid_trigger_status;index" json:"status"`                                 // Current status of the run (e.g., Running, Completed, Failed)
	Source             RunSource      `gorm:"type:valid_trigger_source" json:"source"`                                       // Origin of the run (e.g., Web click, API)
	TotalDuration      null.Int       `gorm:"type:bigint" json:"total-duration"`                                             // Time taken to complete the run in nanoseconds
	TriggeredBy        string         `gorm:"type:varchar(255)" json:"triggered-by"`                                         // Identity of the user who initiated the run
	Namespace          string         `gorm:"type:varchar(255)" json:"namespace"`                                            // Namespace used for the run, which is the credit owner
	Inputs             JSONB          `gorm:"type:jsonb" json:"inputs"`                                                      // Input files for the run
	Outputs            JSONB          `gorm:"type:jsonb" json:"outputs"`                                                     // Output files from the run
	RecipeSnapshot     JSONB          `gorm:"type:jsonb" json:"recipe-snapshot"`                                             // Snapshot of the pipeline recipe used for this run
	StartedTime        time.Time      `gorm:"type:timestamp with time zone;index" json:"started-time,omitempty"`             // Time when the run started execution
	CompletedTime      null.Time      `gorm:"type:timestamp with time zone;index" json:"completed-time,omitempty"`           // Time when the run completed
	Error              null.String    `gorm:"type:text" json:"error-msg"`                                                    // Error message if the run failed
	Components         []ComponentRun `gorm:"foreignKey:PipelineTriggerUID;references:PipelineTriggerUID" json:"components"` // Execution details for each component in the pipeline
}

// ComponentRun represents the execution details of a single component within a pipeline run.
type ComponentRun struct {
	PipelineTriggerUID uuid.UUID   `gorm:"type:uuid;primaryKey;index" json:"pipeline-trigger-uid"`    // Links to the parent PipelineRun
	ComponentID        string      `gorm:"type:varchar(255);primaryKey" json:"component-id"`          // Unique identifier for each pipeline component
	Status             RunStatus   `gorm:"type:varchar(50);index" json:"status"`                      // Completion status of the component (e.g., Completed, Errored)
	TotalDuration      null.Int    `gorm:"type:bigint" json:"total-duration"`                         // Time taken to execute the component in nanoseconds
	StartedTime        time.Time   `gorm:"type:timestamp with time zone;index" json:"started-time"`   // Time when the component started execution
	CompletedTime      null.Time   `gorm:"type:timestamp with time zone;index" json:"completed-time"` // Time when the component finished execution
	Error              null.String `gorm:"type:text" json:"error-msg"`                                // Error message if the component failed
	Inputs             JSONB       `gorm:"type:jsonb" json:"inputs"`                                  // Input files for the component
	Outputs            JSONB       `gorm:"type:jsonb" json:"outputs"`                                 // Output files from the component
}
