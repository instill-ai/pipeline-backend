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
	"gorm.io/datatypes"
	"gorm.io/gorm"

	taskPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// BaseDynamicHardDelete contains common columns for all tables with static UUID as primary key
type BaseDynamicHardDelete struct {
	UID        uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime time.Time `gorm:"autoUpdateTime:nano"`
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

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamicHardDelete) BeforeCreate(db *gorm.DB) error {
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
	Releases          []*PipelineRelease
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
	Trigger    *Trigger     `json:"trigger,omitempty"`
	Components []*Component `json:"components,omitempty"`
}

type Trigger struct {
	TriggerByRequest *TriggerByRequest `json:"trigger_by_request,omitempty"`
}

type Component struct {
	ID       string         `json:"id"`
	Metadata datatypes.JSON `json:"metadata"`
	// TODO: validate oneof
	ResponseComponent  *ResponseComponent  `json:"response_component,omitempty"`
	ConnectorComponent *ConnectorComponent `json:"connector_component,omitempty"`
	OperatorComponent  *OperatorComponent  `json:"operator_component,omitempty"`
	IteratorComponent  *IteratorComponent  `json:"iterator_component,omitempty"`
}

func (c *Component) IsResponseComponent() bool {
	return c.ResponseComponent != nil
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

type TriggerByRequest struct {
	Fields map[string]struct {
		Title              string `json:"title"`
		Description        string `json:"description"`
		InstillFormat      string `json:"instill_format"`
		InstillUIOrder     int32  `json:"instill_ui_order"`
		InstillUIMultiline bool   `json:"instill_ui_multiline"`
	} `json:"fields"`
}

type ResponseComponent struct {
	Fields map[string]struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		Value          string `json:"value"`
		InstillUIOrder int32  `json:"instill_ui_order"`
	} `json:"fields"`
}

type ConnectorComponent struct {
	DefinitionName string           `json:"definition_name"`
	Task           string           `json:"task"`
	Input          *structpb.Struct `json:"input"`
	Condition      *string          `json:"condition,omitempty"`
	Connection     *structpb.Struct `json:"connection"`
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
type PipelineRole pb.Role

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

// Scan function for custom GORM type PipelineRole
func (p *PipelineRole) Scan(value interface{}) error {
	*p = PipelineRole(pb.Role_value[value.(string)])
	return nil
}

// Value function for custom GORM type PipelineRole
func (p PipelineRole) Value() (driver.Value, error) {
	return pb.Role(p).String(), nil
}

// ConnectorType is an alias type for Protobuf enum ConnectorType
type Task taskPB.Task

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
	ReleaseStage  ReleaseStage

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

type pbDefinition interface {
	GetUid() string
	GetId() string
	GetTitle() string
	GetTombstone() bool
	GetPublic() bool
	GetVersion() string
	GetReleaseStage() pb.ComponentDefinition_ReleaseStage
}

// FeatureScores holds the feature score of each component definition. If a
// component definition is not present in the map, the score will be 0.
//
// This is implemented as an inmem map because:
//   - We plan on having a more sophisticated rank in the future (e.g. likes,
//     pipeline triggers where the pipeline uses this component).
//   - We don't want to expose a detail in the component definition list endpoint
//     (mainly used for marketing) in the component definition.
//   - Having a unified place with all the scores provides a quick way to figure
//     out the resulting order.
//
// TODO when a component major version changes, a new component with a
// different ID will be introduced. We'll probably want to keep the score, so
// it would be good that the new ID is xxx.v1 and that we use everything before
// the dot as the "component family" ID. This is what we should use to index
// the score.
var FeatureScores = map[string]int{
	"instill-model": 50,
	"openai":        40,
	"pinecone":      30,
	"numbers":       30,
	"json":          20,
	"redis":         20,
}

// ComponentDefinitionFromProto parses a ComponentDefinition from the proto
// structure.
func ComponentDefinitionFromProto(cdpb *pb.ComponentDefinition) *ComponentDefinition {
	var def pbDefinition
	switch cdpb.Type {
	case pb.ComponentType_COMPONENT_TYPE_OPERATOR:
		def = cdpb.GetOperatorDefinition()
	case pb.ComponentType_COMPONENT_TYPE_CONNECTOR_AI,
		pb.ComponentType_COMPONENT_TYPE_CONNECTOR_DATA,
		pb.ComponentType_COMPONENT_TYPE_CONNECTOR_APPLICATION:

		def = cdpb.GetConnectorDefinition()
	default:
		return nil
	}

	cd := &ComponentDefinition{
		ComponentType: ComponentType(cdpb.Type),

		UID:          uuid.FromStringOrNil(def.GetUid()),
		ID:           def.GetId(),
		Title:        def.GetTitle(),
		Version:      def.GetVersion(),
		IsVisible:    def.GetPublic() && !def.GetTombstone(),
		FeatureScore: FeatureScores[def.GetId()],
		ReleaseStage: ReleaseStage(def.GetReleaseStage()),
	}

	return cd
}

// ComponentType is an alias type for proto enum ComponentType.
type ComponentType pb.ComponentType

// Scan function for custom GORM type ComponentType
func (c *ComponentType) Scan(value any) error {
	*c = ComponentType(pb.ComponentType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ComponentType
func (c ComponentType) Value() (driver.Value, error) {
	return pb.ComponentType(c).String(), nil
}

// ReleaseStage is an alias type for proto enum ComponentDefinition_ReleaseStage.
type ReleaseStage pb.ComponentDefinition_ReleaseStage

// Scan function for custom GORM type ReleaseStage
func (c *ReleaseStage) Scan(value any) error {
	*c = ReleaseStage(pb.ComponentDefinition_ReleaseStage_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (c ReleaseStage) Value() (driver.Value, error) {
	return pb.ComponentDefinition_ReleaseStage(c).String(), nil
}

type Secret struct {
	BaseDynamicHardDelete
	ID          string
	Owner       string
	Description string
	Value       *string
}
