package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamic) BeforeCreate(db *gorm.DB) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	db.Statement.SetColumn("ID", uuid)
	return nil
}

// Pipeline is the data model of the pipeline table
type Pipeline struct {
	BaseDynamic
	OwnerID     uuid.UUID
	Name        string
	Description sql.NullString
	Mode        PipelineMode
	Status      PipelineStatus
	Recipe      *Recipe `gorm:"type:jsonb"`

	// Output-only field
	FullName string `gorm:"-"`
}

// PipelineMode is an alias type for Protobuf enum Pipeline.Mode
type PipelineMode pipelinePB.Pipeline_Mode

// Scan function for custom GORM type PipelineStatus
func (c *PipelineMode) Scan(value interface{}) error {
	*c = PipelineMode(pipelinePB.Pipeline_Mode_value[value.(string)])
	return nil
}

// Value function for custom GORM type PipelineStatus
func (c PipelineMode) Value() (driver.Value, error) {
	return pipelinePB.Pipeline_Mode(c).String(), nil
}

// PipelineStatus is an alias type for Protobuf enum Pipeline.Status
type PipelineStatus pipelinePB.Pipeline_Status

// Scan function for custom GORM type PipelineStatus
func (c *PipelineStatus) Scan(value interface{}) error {
	*c = PipelineStatus(pipelinePB.Pipeline_Status_value[value.(string)])
	return nil
}

// Value function for custom GORM type PipelineStatus
func (c PipelineStatus) Value() (driver.Value, error) {
	return pipelinePB.Pipeline_Status(c).String(), nil
}

// Recipe is the data model of the pipeline recipe
type Recipe struct {
	Source      *Source      `json:"source,omitempty"`
	Destination *Destination `json:"destination,omitempty"`
	Models      []*Model     `json:"models,omitempty"`
	Logics      []*Logic     `json:"logics,omitempty"`
}

// Source is the data model of source connector
type Source struct {
	Name string `json:"name"`
}

// Destination is the data model of destination connector
type Destination struct {
	Name string `json:"name"`
}

// Model is the data model of model
type Model struct {
	Name         string `json:"name"`
	InstanceName string `json:"instance_name"`
}

// Logic is the data model of logic operator
type Logic struct {
}

// Scan function for custom GORM type Recipe
func (r *Recipe) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal Recipe value:", value))
	}

	if err := json.Unmarshal(bytes, &r); err != nil {
		return err
	}

	return nil
}

// Value function for custom GORM type Recipe
func (r *Recipe) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

type TriggerPipeline struct {
	Name     string                    `json:"name,omitempty"`
	Contents []*TriggerPipelineContent `json:"contents,omitempty"`
}

type TriggerPipelineContent struct {
	Url    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
	Chunk  []byte `json:"chunk,omitempty"`
}
