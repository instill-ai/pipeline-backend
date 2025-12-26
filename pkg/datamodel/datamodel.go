package datamodel

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	taskpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

const Iterator = "iterator"

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
	if base.UID.IsNil() {
		uuid, err := uuid.NewV4()
		if err != nil {
			return err
		}
		db.Statement.SetColumn("UID", uuid)
	}
	return nil
}

// BeforeCreate will set a UUID rather than numeric ID.
func (base *BaseDynamicHardDelete) BeforeCreate(db *gorm.DB) error {
	if base.UID.IsNil() {
		uuid, err := uuid.NewV4()
		if err != nil {
			return err
		}
		db.Statement.SetColumn("UID", uuid)
	}
	return nil
}

type HubStats struct {
	NumberOfPublicPipelines   int32
	NumberOfFeaturedPipelines int32
}

// Pipeline is the data model of the pipeline table
type Pipeline struct {
	BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString

	// The Recipe field in the database is deprecated. It will only be used for
	// the structural representation of the recipe instead of as data in the
	// database. We'll use BeforeSave and AfterFind hooks to convert RecipeYAML
	// to Recipe when reading and convert Recipe to RecipeYAML when writing.
	Recipe     *Recipe `gorm:"-"`
	RecipeYAML string  `gorm:"recipe_yaml"`

	DefaultReleaseUID uuid.UUID
	Sharing           *Sharing `gorm:"type:jsonb"`
	ShareCode         string
	Metadata          datatypes.JSON `gorm:"type:jsonb"`
	Readme            string
	SourceURL         sql.NullString
	DocumentationURL  sql.NullString
	License           sql.NullString
	ProfileImage      sql.NullString
	Releases          []*PipelineRelease
	Tags              []*Tag
	NamespaceID       string `gorm:"type:namespace_id"`
	NamespaceType     string `gorm:"type:namespace_type"`

	// CreatorUID is the UID of the user who created this pipeline.
	// This is nullable because pipelines created before this field was added
	// will not have a creator_uid.
	CreatorUID *uuid.UUID `gorm:"type:uuid"`

	// Note:
	// We store the NumberOfRuns, NumberOfClones and LastRunTime in this table
	// to make it easier to sort the pipelines. We should develop an approach to
	// sync the data between InfluxDB and here.
	LastRunTime    time.Time
	NumberOfRuns   int
	NumberOfClones int
}

// IsPublic returns the visibility of the pipeline based on its sharing
// configuration.
func (p Pipeline) IsPublic() bool {
	publicSharing, hasPublicSharing := p.Sharing.Users["*/*"]
	if !hasPublicSharing {
		return false
	}

	return publicSharing.Enabled
}

// OwnerUID returns the UID of the pipeline owner.
func (p Pipeline) OwnerUID() uuid.UUID {
	return uuid.FromStringOrNil(strings.Split(p.Owner, "/")[1])
}

type Tag struct {
	PipelineUID uuid.UUID
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

	// The Recipe field in the database is deprecated. It will only be used for
	// the structural representation of the recipe instead of as data in the
	// database. We'll use BeforeSave and AfterFind hooks to convert RecipeYAML
	// to Recipe when reading and convert Recipe to RecipeYAML when writing.
	Recipe     *Recipe `gorm:"-"`
	RecipeYAML string  `gorm:"recipe_yaml"`

	Metadata datatypes.JSON `gorm:"type:jsonb"`
	Readme   string
}

type ComponentMap map[string]*Component

// Recipe is the data model of the pipeline recipe
type Recipe struct {
	Version   string               `json:"version,omitempty" yaml:"version,omitempty"`
	On        map[string]*Event    `json:"on,omitempty" yaml:"on,omitempty"`
	Component ComponentMap         `json:"component,omitempty" yaml:"component,omitempty"`
	Variable  map[string]*Variable `json:"variable,omitempty" yaml:"variable,omitempty"`
	Secret    map[string]string    `json:"secret,omitempty" yaml:"secret,omitempty"`
	Output    map[string]*Output   `json:"output,omitempty" yaml:"output,omitempty"`
}

func convertRecipeYAMLToRecipe(recipeYAML string) (*Recipe, error) {

	recipe := &Recipe{}
	err := yaml.Unmarshal([]byte(recipeYAML), recipe)
	if err != nil {
		return nil, err
	}
	return recipe, nil
}

func (c *ComponentMap) MarshalJSON() ([]byte, error) {
	if *c == nil {
		c = &ComponentMap{}
	}
	return json.Marshal(*c)
}

func (c *ComponentMap) UnmarshalYAML(node *yaml.Node) error {

	// In this function, we decode the YAML of ComponentMap and parse the YAML
	// comments into `description` and `note`. To achieve this, we need to
	// decode the original `ComponentMap` once and then add the `description`
	// and `note` into the decoded struct. To prevent infinite recursion, we
	// define an alias of `ComponentMap` called `LocalComponentMap`. We decode
	// into `LocalComponentMap`, add the corresponding comments, and then
	// convert it back to `ComponentMap`.
	type LocalComponentMap ComponentMap
	var result LocalComponentMap
	if err := node.Decode(&result); err != nil {
		return err
	}

	for _, content := range node.Content {
		if _, ok := result[content.Value]; ok {
			if result[content.Value] == nil {
				result[content.Value] = &Component{}
			}
			headCommentLines := strings.Split(content.HeadComment, "\n")
			for i := range headCommentLines {
				headCommentLines[i] = strings.TrimLeft(headCommentLines[i], "# ")
			}
			result[content.Value].Description = strings.Join(headCommentLines, "\n")
		}

	}
	*c = ComponentMap(result)
	return nil
}

func convertRecipeToRecipeYAML(recipe *Recipe) (string, error) {
	if recipe == nil {
		return "", nil
	}
	recipeYAML, err := yaml.Marshal(recipe)
	if err != nil {
		return "", err
	}
	return string(recipeYAML), nil
}

func (p *Pipeline) BeforeSave(db *gorm.DB) (err error) {

	// In the future, we'll make YAML the only input data type for pipeline
	// recipes. Until then, if the YAML recipe is empty, we'll use the JSON
	// recipe as the input data. Once the JSON recipe becomes output-only, this
	// condition will no longer be necessary.
	if p.RecipeYAML == "" {
		p.RecipeYAML, err = convertRecipeToRecipeYAML(p.Recipe)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *PipelineRelease) BeforeSave(db *gorm.DB) (err error) {

	// In the future, we'll make YAML the only input data type for pipeline
	// recipes. Until then, if the YAML recipe is empty, we'll use the JSON
	// recipe as the input data. Once the JSON recipe becomes output-only, this
	// condition will no longer be necessary.
	if p.RecipeYAML == "" {
		p.RecipeYAML, err = convertRecipeToRecipeYAML(p.Recipe)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pipeline) AfterFind(tx *gorm.DB) (err error) {
	if p.RecipeYAML == "" {
		p.Recipe = nil
		return
	}
	// For an invalid YAML recipe, we ignore the error and return a `nil`
	// structured recipe.
	p.Recipe, _ = convertRecipeYAMLToRecipe(p.RecipeYAML)
	return
}

func (p *PipelineRelease) AfterFind(tx *gorm.DB) (err error) {
	if p.RecipeYAML == "" {
		p.Recipe = nil
		return
	}

	// For an invalid YAML recipe, we ignore the error and return a `nil`
	// structured recipe.
	p.Recipe, _ = convertRecipeYAMLToRecipe(p.RecipeYAML)
	return
}

type Variable struct {
	Title       string `json:"title,omitempty" yaml:"title,omitempty"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// The `instillFormat` field has been renamed to `type`. The old field name is kept
	// in the JSON tag to preserve backward compatibility for Console.
	Type           string   `json:"instillFormat,omitempty" yaml:"type,omitempty"`
	Listen         []string `json:"listen,omitempty" yaml:"listen,omitempty"`
	Default        any      `json:"default,omitempty" yaml:"default,omitempty"`
	InstillUIOrder int32    `json:"instillUiOrder,omitempty" yaml:"instill-ui-order,omitempty"`
	Required       bool     `json:"required,omitempty" yaml:"required,omitempty"`
}

type Output struct {
	Title          string `json:"title,omitempty" yaml:"title,omitempty"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	Value          string `json:"value,omitempty" yaml:"value,omitempty"`
	InstillUIOrder int32  `json:"instillUiOrder,omitempty" yaml:"instill-ui-order,omitempty"`
}

type Event struct {
	Type   string         `json:"type,omitempty" yaml:"type,omitempty"`
	Event  string         `json:"event,omitempty" yaml:"event,omitempty"`
	Config map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
	Setup  any            `json:"setup,omitempty" yaml:"setup,omitempty"`
}

type Schedule struct {
	Cron string `json:"cron,omitempty" yaml:"cron,omitempty"`
}

type Component struct {
	// Common fields
	Type      string         `json:"type,omitempty" yaml:"type,omitempty"`
	Task      string         `json:"task,omitempty" yaml:"task,omitempty"`
	Input     any            `json:"input,omitempty" yaml:"input,omitempty"`
	Condition string         `json:"condition,omitempty" yaml:"condition,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"  yaml:"metadata,omitempty"`

	// The YAML header comment will be parsed into the `Description` field.
	Description string `json:"description,omitempty"  yaml:"-"`

	// Fields for regular components
	Setup      any         `json:"setup,omitempty" yaml:"setup,omitempty"`
	Definition *Definition `json:"definition,omitempty" yaml:"-"`

	// Fields for iterators
	Range             any                           `json:"range,omitempty" yaml:"range,omitempty"`
	Index             string                        `json:"index,omitempty" yaml:"index,omitempty"`
	Component         ComponentMap                  `json:"component,omitempty" yaml:"component,omitempty"`
	OutputElements    map[string]string             `json:"outputElements,omitempty" yaml:"output-elements,omitempty"`
	DataSpecification *pipelinepb.DataSpecification `json:"dataSpecification,omitempty" yaml:"-"`
}

type Definition struct {
	*pipelinepb.ComponentDefinition
}
type DataSpecification struct {
	*pipelinepb.DataSpecification
}

func (c *Definition) MarshalJSON() ([]byte, error) {
	defBytes, err := protojson.Marshal(c.ComponentDefinition)
	if err != nil {
		return nil, err
	}
	return defBytes, nil
}

func (c *DataSpecification) MarshalJSON() ([]byte, error) {
	defBytes, err := protojson.Marshal(c.DataSpecification)
	if err != nil {
		return nil, err
	}
	return defBytes, nil
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
type PipelineRole pipelinepb.Role

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

func (p *Pipeline) TagNames() []string {
	tags := make([]string, len(p.Tags))
	for i, t := range p.Tags {
		tags[i] = t.TagName
	}
	return tags
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
	*p = PipelineRole(pipelinepb.Role_value[value.(string)])
	return nil
}

// Value function for custom GORM type PipelineRole
func (p PipelineRole) Value() (driver.Value, error) {
	return pipelinepb.Role(p).String(), nil
}

// ConnectorType is an alias type for Protobuf enum ConnectorType
type Task taskpb.Task

// Scan function for custom GORM type Task
func (r *Task) Scan(value interface{}) error {
	*r = Task(taskpb.Task_value[value.(string)])
	return nil
}

// Value function for custom GORM type Task
func (r Task) Value() (driver.Value, error) {
	return taskpb.Task(r).String(), nil
}

// ComponentDefinition is the data model for the component defintion table.
type ComponentDefinition struct {
	UID           uuid.UUID `gorm:"type:uuid;primaryKey;<-:create"` // allow read and create
	ID            string
	Title         string
	Vendor        string
	ComponentType ComponentType
	Version       string
	ReleaseStage  ReleaseStage

	// IsVisible is computed from a combination of fields (e.g. tombstone,
	// public, deprecated), and is used to hide components from the list
	// endpoint.
	IsVisible bool
	// HasIntegration indicates that integrations can be created for a
	// component definition. It is determined by the presence of a `setup`
	// object in the specification.
	HasIntegration bool
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
func ComponentDefinitionFromProto(cdpb *pipelinepb.ComponentDefinition) *ComponentDefinition {
	props, hasIntegration := cdpb.GetSpec().GetComponentSpecification().GetFields()["properties"]
	if hasIntegration {
		_, hasIntegration = props.GetStructValue().GetFields()["setup"]
	}

	cd := &ComponentDefinition{
		ComponentType: ComponentType(cdpb.Type),

		UID:            uuid.FromStringOrNil(cdpb.GetUid()),
		ID:             cdpb.GetId(),
		Title:          cdpb.GetTitle(),
		Vendor:         cdpb.GetVendor(),
		Version:        cdpb.GetVersion(),
		IsVisible:      cdpb.GetPublic() && !cdpb.GetTombstone(),
		HasIntegration: hasIntegration,
		FeatureScore:   FeatureScores[cdpb.GetId()],
		ReleaseStage:   ReleaseStage(cdpb.GetReleaseStage()),
	}

	return cd
}

// ComponentType is an alias type for proto enum ComponentType.
type ComponentType pipelinepb.ComponentType

// Scan function for custom GORM type ComponentType
func (c *ComponentType) Scan(value any) error {
	*c = ComponentType(pipelinepb.ComponentType_value[value.(string)])
	return nil
}

// Value function for custom GORM type ComponentType
func (c ComponentType) Value() (driver.Value, error) {
	return pipelinepb.ComponentType(c).String(), nil
}

// ReleaseStage is an alias type for proto enum ComponentDefinition_ReleaseStage.
type ReleaseStage pipelinepb.ComponentDefinition_ReleaseStage

// Scan function for custom GORM type ReleaseStage
func (c *ReleaseStage) Scan(value any) error {
	*c = ReleaseStage(pipelinepb.ComponentDefinition_ReleaseStage_value[value.(string)])
	return nil
}

// Value function for custom GORM type ReleaseStage
func (c ReleaseStage) Value() (driver.Value, error) {
	return pipelinepb.ComponentDefinition_ReleaseStage(c).String(), nil
}

type Secret struct {
	BaseDynamicHardDelete
	ID            string
	Owner         string
	Description   string
	Value         *string
	NamespaceID   string `gorm:"type:namespace_id"`
	NamespaceType string `gorm:"type:namespace_type"`
}

// ConnectionMethod is an alias type for the proto enum that allows us to use its string value in the database.
type ConnectionMethod pipelinepb.Role

// Scan function for custom GORM type ConnectionMethod
func (m *ConnectionMethod) Scan(value interface{}) error {
	*m = ConnectionMethod(pipelinepb.Connection_Method_value[value.(string)])
	return nil
}

// Value function for custom GORM type ConnectionMethod
func (m ConnectionMethod) Value() (driver.Value, error) {
	return pipelinepb.Connection_Method(m).String(), nil
}

// Connection is the data model for the `integration` table
type Connection struct {
	BaseDynamic
	ID                 string
	NamespaceUID       uuid.UUID
	IntegrationUID     uuid.UUID
	Method             ConnectionMethod
	Identity           sql.NullString
	Setup              datatypes.JSON      `gorm:"type:jsonb"`
	Scopes             pq.StringArray      `gorm:"type:text[]"`
	OAuthAccessDetails datatypes.JSON      `gorm:"type:jsonb"`
	Integration        ComponentDefinition `gorm:"foreignKey:IntegrationUID;references:UID"`
}

// PipelineRunOn is the data model for the `pipeline_run_on` table
type PipelineRunOn struct {
	BaseDynamicHardDelete
	PipelineUID uuid.UUID
	ReleaseUID  uuid.UUID
	EventID     string
	RunOnType   string
	Identifier  datatypes.JSON `gorm:"type:jsonb"`
	Config      datatypes.JSON `gorm:"type:jsonb"`
	Setup       datatypes.JSON `gorm:"type:jsonb"`
}
