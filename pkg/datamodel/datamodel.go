package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	componentbase "github.com/instill-ai/component/base"
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

type HubStats struct {
	NumberOfPublicPipelines   int32
	NumberOfFeaturedPipelines int32
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
	Tags              []*Tag

	// Note:
	// We store the NumberOfRuns and LastRunTime in this table to
	// make it easier to sort the pipelines. We should develop an approach
	// to sync the data between InfluxDB and here.
	NumberOfRuns int
	LastRunTime  time.Time
}

type Tag struct {
	PipelineUID string
	TagName     string
	CreateTime  time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime  time.Time `gorm:"autoUpdateTime:nano"`
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
	Version   string                `json:"version,omitempty"`
	On        *On                   `json:"on,omitempty"`
	Component map[string]IComponent `json:"component,omitempty"`
	Variable  map[string]*Variable  `json:"variable,omitempty"`
	Secret    map[string]string     `json:"secret,omitempty"`
	Output    map[string]*Output    `json:"output,omitempty"`
}

type IComponent interface {
	IsComponent()
	GetCondition() *string
}

func (r *Recipe) UnmarshalJSON(data []byte) error {
	// TODO: we should catch the errors here and return to the user.
	tmp := make(map[string]any)
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	if v, ok := tmp["on"]; ok && v != nil {
		b, _ := json.Marshal(tmp["on"])
		_ = json.Unmarshal(b, &r.On)
	}
	if v, ok := tmp["variable"]; ok && v != nil {
		b, _ := json.Marshal(tmp["variable"])
		_ = json.Unmarshal(b, &r.Variable)

	}
	if v, ok := tmp["secret"]; ok && v != nil {
		b, _ := json.Marshal(tmp["secret"])
		_ = json.Unmarshal(b, &r.Secret)

	}
	if v, ok := tmp["output"]; ok && v != nil {
		b, _ := json.Marshal(tmp["output"])
		_ = json.Unmarshal(b, &r.Output)
	}
	if v, ok := tmp["component"]; ok && v != nil {
		comps := v.(map[string]any)
		r.Component = make(map[string]IComponent)
		for id, comp := range comps {

			if t, ok := comp.(map[string]any)["type"]; !ok {
				return fmt.Errorf("component type error")
			} else {
				if t == "iterator" {
					b, _ := json.Marshal(comp)
					c := IteratorComponent{}
					_ = json.Unmarshal(b, &c)
					r.Component[id] = &c
				} else {
					b, _ := json.Marshal(comp)
					c := componentbase.ComponentConfig{}
					_ = json.Unmarshal(b, &c)
					r.Component[id] = &c
				}
			}
		}
	}

	return nil
}
func (i *IteratorComponent) IsComponent() {}
func (i *IteratorComponent) GetCondition() *string {
	return i.Condition
}

type Variable struct {
	Title              string `json:"title,omitempty"`
	Description        string `json:"description,omitempty"`
	InstillFormat      string `json:"instillFormat,omitempty"`
	InstillUIOrder     int32  `json:"instillUiOrder,omitempty"`
	InstillUIMultiline bool   `json:"instillUiMultiline,omitempty"`
}

type Output struct {
	Title          string `json:"title,omitempty"`
	Description    string `json:"description,omitempty"`
	Value          string `json:"value,omitempty"`
	InstillUIOrder int32  `json:"instillUiOrder,omitempty"`
}

type On struct {
}

type IteratorComponent struct {
	Type              string                `json:"type,omitempty"`
	Input             string                `json:"input,omitempty"`
	OutputElements    map[string]string     `json:"outputElements,omitempty"`
	Condition         *string               `json:"condition,omitempty"`
	Component         map[string]IComponent `json:"component,omitempty"`
	Metadata          datatypes.JSON        `json:"metadata,omitempty"`
	DataSpecification *pb.DataSpecification `json:"dataSpecification,omitempty"`
}

func (i *IteratorComponent) UnmarshalJSON(data []byte) error {
	tmp := make(map[string]any)
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	if v, ok := tmp["type"]; ok && v != nil {
		i.Type = tmp["type"].(string)
	}
	if v, ok := tmp["input"]; ok && v != nil {
		i.Input = tmp["input"].(string)
	}
	if v, ok := tmp["outputElements"]; ok && v != nil {
		b, _ := json.Marshal(v)
		c := map[string]string{}
		_ = json.Unmarshal(b, &c)
		i.OutputElements = c
	}
	if v, ok := tmp["condition"]; ok && v != nil {
		i.Condition = tmp["condition"].(*string)
	}
	if v, ok := tmp["metadata"]; ok && v != nil {
		b, _ := json.Marshal(tmp["metadata"])
		m := datatypes.JSON{}
		err = json.Unmarshal(b, &m)
		if err != nil {
			return err
		}
		i.Metadata = m
	}
	if v, ok := tmp["data_specification"]; ok && v != nil {
		i.DataSpecification = tmp["data_specification"].(*pb.DataSpecification)
	}
	if v, ok := tmp["component"]; ok && v != nil {
		comps := v.(map[string]any)
		i.Component = make(map[string]IComponent)
		for id, comp := range comps {

			if _, ok := comp.(map[string]any)["type"]; ok {

				b, _ := json.Marshal(comp)
				c := componentbase.ComponentConfig{}
				_ = json.Unmarshal(b, &c)
				i.Component[id] = &c

			}

		}
	}

	return nil
}

type Sharing struct {
	Users     map[string]*SharingUser `json:"users,omitempty"`
	ShareCode *SharingCode            `json:"shareCode,omitempty"`
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

	cd := &ComponentDefinition{
		ComponentType: ComponentType(cdpb.Type),

		UID:          uuid.FromStringOrNil(cdpb.GetUid()),
		ID:           cdpb.GetId(),
		Title:        cdpb.GetTitle(),
		Version:      cdpb.GetVersion(),
		IsVisible:    cdpb.GetPublic() && !cdpb.GetTombstone(),
		FeatureScore: FeatureScores[cdpb.GetId()],
		ReleaseStage: ReleaseStage(cdpb.GetReleaseStage()),
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
