package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// BaseDynamic contains common columns for all tables with dynamic UUID as primary key generated when creating
type BaseDynamic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
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
	db.Statement.SetColumn("UID", uuid)
	return nil
}

// Pipeline is the data model of the pipeline table
type Pipeline struct {
	BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	Mode        PipelineMode
	State       PipelineState
	Recipe      *Recipe `gorm:"type:jsonb"`
}

// PipelineMode is an alias type for Protobuf enum Pipeline.Mode
type PipelineMode pipelinePB.Pipeline_Mode

// Scan function for custom GORM type PipelineMode
func (p *PipelineMode) Scan(value interface{}) error {
	*p = PipelineMode(pipelinePB.Pipeline_Mode_value[value.(string)])
	return nil
}

// Value function for custom GORM type PipelineMode
func (p PipelineMode) Value() (driver.Value, error) {
	return pipelinePB.Pipeline_Mode(p).String(), nil
}

// PipelineState is an alias type for Protobuf enum Pipeline.State
type PipelineState pipelinePB.Pipeline_State

// Scan function for custom GORM type PipelineState
func (p *PipelineState) Scan(value interface{}) error {
	*p = PipelineState(pipelinePB.Pipeline_State_value[value.(string)])
	return nil
}

// Value function for custom GORM type PipelineState
func (p PipelineState) Value() (driver.Value, error) {
	return pipelinePB.Pipeline_State(p).String(), nil
}

// Recipe is the data model of the pipeline recipe
type Recipe struct {
	Version    string       `json:"version,omitempty"`
	Components []*Component `json:"components,omitempty"`
}

type Component struct {
	Id             string            `json:"id,omitempty"`
	ResourceName   string            `json:"resource_name,omitempty"`
	ResourceDetail *structpb.Struct  `json:"resource_detail,omitempty"`
	Metadata       *structpb.Struct  `json:"metadata,omitempty"`
	Dependencies   map[string]string `json:"dependencies,omitempty"`
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
