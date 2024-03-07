package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/launchdarkly/go-semver"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	taskPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// BaseStatic contains common columns for all tables with static UUID as primary key
type BaseStatic struct {
	UID        uuid.UUID      `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time      `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time      `gorm:"autoUpdateTime:nano"`
	DeleteTime gorm.DeletedAt `sql:"index"`
}

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
	ID                string
	Owner             string
	Description       sql.NullString
	Recipe            *Recipe `gorm:"type:jsonb"`
	DefaultReleaseUID uuid.UUID
	Sharing           *Sharing `gorm:"type:jsonb"`
	ShareCode         string
	Metadata          datatypes.JSON `gorm:"type:jsonb"`
	Readme            string
}

// PipelineRelease is the data model of the pipeline release table
type PipelineRelease struct {
	BaseDynamic
	ID          string
	PipelineUID uuid.UUID
	Description sql.NullString
	Recipe      *Recipe        `gorm:"type:jsonb"`
	Metadata    datatypes.JSON `gorm:"type:jsonb"`
	Readme      string
}

// Recipe is the data model of the pipeline recipe
type Recipe struct {
	Version    string       `json:"version,omitempty"`
	Components []*Component `json:"components,omitempty"`
}

type Component struct {
	ID       string         `json:"id"`
	Metadata datatypes.JSON `json:"metadata"`
	// TODO: validate oneof
	StartComponent     *StartComponent     `json:"start_component,omitempty"`
	EndComponent       *EndComponent       `json:"end_component,omitempty"`
	ConnectorComponent *ConnectorComponent `json:"connector_component,omitempty"`
	OperatorComponent  *OperatorComponent  `json:"operator_component,omitempty"`
	IteratorComponent  *IteratorComponent  `json:"iterator_component,omitempty"`
}

func (c *Component) IsStartComponent() bool {
	return c.StartComponent != nil
}
func (c *Component) IsEndComponent() bool {
	return c.EndComponent != nil
}
func (c *Component) IsConnectorComponent() bool {
	return c.ConnectorComponent != nil
}
func (c *Component) IsOperatorComponent() bool {
	return c.OperatorComponent != nil
}
func (c *Component) IsIteratorComponent() bool {
	return c.IteratorComponent != nil
}
func (c *Component) GetCondition() *string {
	if c.IsConnectorComponent() {
		return c.ConnectorComponent.Condition
	}
	if c.IsOperatorComponent() {
		return c.OperatorComponent.Condition
	}
	if c.IsIteratorComponent() {
		return c.IteratorComponent.Condition
	}
	return nil
}

type StartComponent struct {
	Fields map[string]struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		InstillFormat  string `json:"instill_format"`
		InstillUIOrder int32  `json:"instill_ui_order"`
	} `json:"fields"`
}

type EndComponent struct {
	Fields map[string]struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		Value          string `json:"value"`
		InstillUIOrder int32  `json:"instill_ui_order"`
	} `json:"fields"`
}

type ConnectorComponent struct {
	DefinitionName string           `json:"definition_name"`
	ConnectorName  string           `json:"connector_name"`
	Task           string           `json:"task"`
	Input          *structpb.Struct `json:"input"`
	Condition      *string          `json:"condition,omitempty"`
}

type OperatorComponent struct {
	DefinitionName string           `json:"definition_name"`
	Task           string           `json:"task"`
	Input          *structpb.Struct `json:"input"`
	Condition      *string          `json:"condition,omitempty"`
}

type IteratorComponent struct {
	Input          string            `json:"input"`
	OutputElements map[string]string `json:"output_elements"`
	Condition      *string           `json:"condition,omitempty"`
	Components     []*Component      `json:"components"`
}

type Sharing struct {
	Users     map[string]*SharingUser `json:"users,omitempty"`
	ShareCode *SharingCode            `json:"share_code,omitempty"`
}

// Sharing
type SharingUser struct {
	Enabled bool   `json:"enabled,omitempty"`
	Role    string `json:"role,omitempty"`
}

type SharingCode struct {
	User    string `json:"user"`
	Code    string `json:"code"`
	Enabled bool   `json:"enabled,omitempty"`
	Role    string `json:"role,omitempty"`
}

// PipelineRole is an alias type for Protobuf enum
type PipelineRole pipelinePB.Role

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

// Scan function for custom GORM type Recipe
func (p *Sharing) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal value:", value))
	}

	if err := json.Unmarshal(bytes, &p); err != nil {
		return err
	}

	return nil
}

// Value function for custom GORM type Recipe
func (p *Sharing) Value() (driver.Value, error) {
	valueString, err := json.Marshal(p)
	return string(valueString), err
}

// Scan function for custom GORM type Recipe
func (p *SharingUser) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal value:", value))
	}

	if err := json.Unmarshal(bytes, &p); err != nil {
		return err
	}

	return nil
}

// Value function for custom GORM type Recipe
func (p *SharingUser) Value() (driver.Value, error) {
	valueString, err := json.Marshal(p)
	return string(valueString), err
}

// Scan function for custom GORM type Recipe
func (p *SharingCode) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal value:", value))
	}

	if err := json.Unmarshal(bytes, &p); err != nil {
		return err
	}

	return nil
}

// Value function for custom GORM type Recipe
func (p *SharingCode) Value() (driver.Value, error) {
	valueString, err := json.Marshal(p)
	return string(valueString), err
}

// Scan function for custom GORM type ReleaseStage
func (p *PipelineRole) Scan(value interface{}) error {
	*p = PipelineRole(pipelinePB.Role_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (p PipelineRole) Value() (driver.Value, error) {
	return pipelinePB.Role(p).String(), nil
}

// Connector is the data model of the connector table
type Connector struct {
	BaseDynamic
	ID                     string
	Owner                  string
	ConnectorDefinitionUID uuid.UUID
	Description            sql.NullString
	Tombstone              bool
	Configuration          datatypes.JSON      `gorm:"type:jsonb"`
	ConnectorType          ConnectorType       `sql:"type:CONNECTOR_VALID_CONNECTOR_TYPE"`
	State                  ConnectorState      `sql:"type:CONNECTOR_VALID_CONNECTOR_STATE"`
	Visibility             ConnectorVisibility `sql:"type:CONNECTOR_VALID_CONNECTOR_VISIBILITY"`
}

func (Connector) TableName() string {
	return "connector"
}

// ConnectorType is an alias type for Protobuf enum ConnectorType
type ConnectorVisibility pipelinePB.Connector_Visibility

// ConnectorType is an alias type for Protobuf enum ConnectorType
type ConnectorType pipelinePB.ConnectorType

// ConnectorType is an alias type for Protobuf enum ConnectorType
type Task taskPB.Task

// Scan function for custom GORM type ConnectorType
func (c *ConnectorType) Scan(value interface{}) error {
	*c = ConnectorType(pipelinePB.ConnectorType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorType
func (c ConnectorType) Value() (driver.Value, error) {
	return pipelinePB.ConnectorType(c).String(), nil
}

// ConnectorState is an alias type for Protobuf enum ConnectorState
type ConnectorState pipelinePB.Connector_State

// Scan function for custom GORM type ConnectorState
func (r *ConnectorState) Scan(value interface{}) error {
	*r = ConnectorState(pipelinePB.Connector_State_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectorState
func (r ConnectorState) Value() (driver.Value, error) {
	return pipelinePB.Connector_State(r).String(), nil
}

// Scan function for custom GORM type ReleaseStage
func (r *ConnectorVisibility) Scan(value interface{}) error {
	*r = ConnectorVisibility(pipelinePB.Connector_Visibility_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (r ConnectorVisibility) Value() (driver.Value, error) {
	return pipelinePB.Connector_Visibility(r).String(), nil
}

// Scan function for custom GORM type Task
func (r *Task) Scan(value interface{}) error {
	*r = Task(taskPB.Task_value[value.(string)])
	return nil
}

// Value function for custom GORM type Task
func (r Task) Value() (driver.Value, error) {
	return taskPB.Task(r).String(), nil
}

// ComponentDefinition is the data model for the component defintion table.
type ComponentDefinition struct {
	UID           uuid.UUID `gorm:"type:uuid;primaryKey;<-:create"` // allow read and create
	ID            string
	Title         string
	ComponentType ComponentType
	Version       string

	// This is an enum in the database but it's only used for filtering, so for
	// now we don't need to implement a domain type.
	ReleaseStage string
	// IsVisible is computed from a combination of fields (e.g. tombstone,
	// public, deprecated), and is used to hide components from the list
	// endpoint.
	IsVisible bool
	// FeatureScore is used to position results in a page, i.e., to give more
	// visibility to certain components.
	FeatureScore int
}

// TableName maps the ComponentDefinition object to a SQL table.
func (ComponentDefinition) TableName() string {
	return "component_definition_index"
}

const (
	rsUnspecified        = "RELEASE_STAGE_UNSPECIFIED"
	rsContribution       = "RELEASE_STAGE_OPEN_FOR_CONTRIBUTION"
	rsComingSoon         = "RELEASE_STAGE_COMING_SOON"
	rsAlpha              = "RELEASE_STAGE_ALPHA"
	rsBeta               = "RELEASE_STAGE_BETA"
	rsGenerallyAvailable = "RELEASE_STAGE_GA"
)

func (c ComponentDefinition) computeReleaseStage() string {
	v, err := semver.Parse(c.Version)
	if err != nil {
		return rsUnspecified
	}

	switch v.GetPrerelease() {
	case "alpha":
		return rsAlpha
	case "beta":
		return rsBeta
	}

	// TODO compute Contribution / Coming soon when introduced.

	return rsGenerallyAvailable
}

type pbDefinition interface {
	GetUid() string
	GetId() string
	GetTitle() string
	GetTombstone() bool
	GetPublic() bool
	GetVersion() string
}

// ComponentDefinitionFromProto parses a ComponentDefinition from the proto
// structure.
func ComponentDefinitionFromProto(cdpb *pipelinePB.ComponentDefinition) *ComponentDefinition {
	var def pbDefinition
	switch cdpb.Type {
	case pipelinePB.ComponentType_COMPONENT_TYPE_OPERATOR:
		def = cdpb.GetOperatorDefinition()
	case pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
		pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
		pipelinePB.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION:

		def = cdpb.GetConnectorDefinition()
	default:
		return nil
	}

	cd := &ComponentDefinition{
		ComponentType: ComponentType(cdpb.Type),

		UID:       uuid.FromStringOrNil(def.GetUid()),
		ID:        def.GetId(),
		Title:     def.GetTitle(),
		Version:   def.GetVersion(),
		IsVisible: def.GetPublic() && !def.GetTombstone(),
	}

	cd.ReleaseStage = cd.computeReleaseStage()

	// TODO read FeatureScore from definition.

	return cd
}

// ComponentType is an alias type for proto enum ComponentType.
type ComponentType pipelinePB.ComponentType

// Scan function for custom GORM type ComponentType
func (c *ComponentType) Scan(value any) error {
	*c = ComponentType(pipelinePB.ComponentType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ComponentType
func (c ComponentType) Value() (driver.Value, error) {
	return pipelinePB.ComponentType(c).String(), nil
}
