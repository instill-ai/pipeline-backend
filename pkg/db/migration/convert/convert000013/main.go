package convert000013

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PaesslerAG/jsonpath"
	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

type Connector struct {
	UID                    uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ConnectorDefinitionUID uuid.UUID
	DeleteTime             gorm.DeletedAt `sql:"index"`
	ID                     string
	Owner                  string
	Configuration          datatypes.JSON `gorm:"type:jsonb"`
}

type Secret struct {
	UID         uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	CreateTime  time.Time `gorm:"autoCreateTime:nano"`
	UpdateTime  time.Time `gorm:"autoUpdateTime:nano"`
	ID          string
	Owner       string
	Description string
	Value       *string
}

// BeforeCreate will set a UUID rather than numeric ID.
func (s *Secret) BeforeCreate(db *gorm.DB) error {
	uuid, err := uuid.NewV4()
	if err != nil {
		return err
	}
	db.Statement.SetColumn("UID", uuid)
	return nil
}

// Pipeline is the data model of the pipeline table
type Pipeline struct {
	UID    uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ID     string
	Owner  string
	Recipe *Recipe `gorm:"type:jsonb"`
}

// PipelineRelease is the data model of the pipeline release table
type PipelineRelease struct {
	UID         uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ID          string
	PipelineUID uuid.UUID
	Recipe      *Recipe `gorm:"type:jsonb"`
}

type Recipe struct {
	Version    string       `json:"version,omitempty"`
	Trigger    *Trigger     `json:"trigger,omitempty"`
	Components []*Component `json:"components,omitempty"`
}

type TriggerByRequestRequestFields map[string]struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	InstillFormat      string `json:"instill_format"`
	InstillUIOrder     int32  `json:"instill_ui_order"`
	InstillUIMultiline bool   `json:"instill_ui_multiline"`
}

type TriggerByRequestResponseFields map[string]struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Value          string `json:"value"`
	InstillUIOrder int32  `json:"instill_ui_order"`
}

type TriggerByRequest struct {
	RequestFields  TriggerByRequestRequestFields  `json:"request_fields"`
	ResponseFields TriggerByRequestResponseFields `json:"response_fields"`
}

type Trigger struct {
	TriggerByRequest *TriggerByRequest `json:"trigger_by_request,omitempty"`
}

type Component struct {
	ID                 string              `json:"id"`
	Metadata           datatypes.JSON      `json:"metadata"`
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
func (c *Component) IsIteratorComponent() bool {
	return c.IteratorComponent != nil
}

type StartComponent struct {
	Fields map[string]struct {
		Title              string `json:"title"`
		Description        string `json:"description"`
		InstillFormat      string `json:"instill_format"`
		InstillUIOrder     int32  `json:"instill_ui_order"`
		InstillUIMultiline bool   `json:"instill_ui_multiline"`
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
	ConnectorName  string           `json:"connector_name,omitempty"`
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

// Note: We list the `secretFields` that need to be migrated manually.
var secretFields = map[string]string{
	"e414a1f8-5fdf-4292-b050-9f9176254a4b": "api_key",
	"e2ffe076-ab2c-4e5e-9587-a613a6b1c146": "json_key",
	"205cbeff-6f45-4abe-b0a8-cec1a310137f": "json_key",
	"2b1da686-878a-462c-b2c6-a9690199939c": "api_key",
	"0255ef87-33ce-4f88-b9db-8897f8c17233": "api_key",
	"ddcf42c3-4c30-4c65-9585-25f1c89b2b48": "api_token",
	"70d8664a-d512-4517-a5e8-5d4da81756a7": "capture_token",
	"9fb6a2cb-bff5-4c69-bc6d-4538dd8e3362": "api_key",
	"4b1dcf82-e134-4ba7-992f-f9a02536ec2b": "api_key",
	"fd0ad325-f2f7-41f3-b247-6c71d571b1b8": "password,ssl_mode.ca_cert,ssl_mode.client_cert,ssl_mode.client_key",
	"5ee55a5c-6e30-4c7a-80e8-90165a729e0a": "authentication.password,authentication.value,authentication.token",
	"c86a95cc-7d32-4e22-a290-8c699f6705a4": "api_key",
}

var defMap = map[string]string{
	"e414a1f8-5fdf-4292-b050-9f9176254a4b": "archetype-ai",
	"e2ffe076-ab2c-4e5e-9587-a613a6b1c146": "bigquery",
	"205cbeff-6f45-4abe-b0a8-cec1a310137f": "gcs",
	"2b1da686-878a-462c-b2c6-a9690199939c": "google-search",
	"0255ef87-33ce-4f88-b9db-8897f8c17233": "hugging-face",
	"ddcf42c3-4c30-4c65-9585-25f1c89b2b48": "instill-model",
	"70d8664a-d512-4517-a5e8-5d4da81756a7": "numbers",
	"9fb6a2cb-bff5-4c69-bc6d-4538dd8e3362": "openai",
	"4b1dcf82-e134-4ba7-992f-f9a02536ec2b": "pinecone",
	"fd0ad325-f2f7-41f3-b247-6c71d571b1b8": "redis",
	"5ee55a5c-6e30-4c7a-80e8-90165a729e0a": "restapi",
	"c86a95cc-7d32-4e22-a290-8c699f6705a4": "stability-ai",
}

func migrateSecret() (map[uuid.UUID]Connector, error) {
	db := database.GetConnection()
	defer database.Close(db)

	var connectors []Connector
	connectorMap := map[uuid.UUID]Connector{}
	result := db.Model(&Connector{})
	if result.Error != nil {
		return nil, result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var connector Connector
		if err = db.ScanRows(rows, &connector); err != nil {
			return nil, err
		}
		connectors = append(connectors, connector)
		connectorMap[connector.UID] = connector
	}
	for _, con := range connectors {
		defUID := con.ConnectorDefinitionUID.String()
		if keys, ok := secretFields[defUID]; ok {
			for _, key := range strings.Split(keys, ",") {
				cfg := map[string]any{}
				err := json.Unmarshal(con.Configuration, &cfg)
				if err != nil {
					return nil, err
				}

				v, err := jsonpath.Get(fmt.Sprintf("$.%s", key), cfg)
				if err == nil {
					conID := strings.ReplaceAll(con.ID, "_", "-")
					value := v.(string)

					s := Secret{
						ID:    fmt.Sprintf("%s-%s-%s", defMap[defUID], conID, strings.ReplaceAll(key, "_", "-")),
						Value: &value,
						Owner: con.Owner,
					}

					result := db.Model(&Secret{}).Create(&s)
					if result.Error != nil {
						return nil, result.Error
					}
				}
			}

		}
	}
	return connectorMap, nil
}

func migrateConnectorComponent(connectorMap map[uuid.UUID]Connector, c *Component, owner string) {
	newComp := Component{
		ID:       c.ID,
		Metadata: c.Metadata,
		ConnectorComponent: &ConnectorComponent{
			DefinitionName: c.ConnectorComponent.DefinitionName,
			Task:           c.ConnectorComponent.Task,
			Input:          c.ConnectorComponent.Input,
			Condition:      c.ConnectorComponent.Condition,
			Connection:     &structpb.Struct{},
		},
	}

	if len(strings.Split(c.ConnectorComponent.ConnectorName, "/")) == 2 {
		defUID := strings.Split(c.ConnectorComponent.DefinitionName, "/")[1]
		conUID := uuid.FromStringOrNil(strings.Split(c.ConnectorComponent.ConnectorName, "/")[1])
		if connector, ok := connectorMap[conUID]; ok {
			if connector.Owner != owner {
				return
			}

			b, _ := json.Marshal(connector.Configuration)
			_ = protojson.Unmarshal(b, newComp.ConnectorComponent.Connection)
			keys := secretFields[defUID]
			for _, key := range strings.Split(keys, ",") {
				if splits := strings.Split(key, "."); len(splits) == 1 {
					conID := connectorMap[conUID].ID
					conID = strings.ReplaceAll(conID, "_", "-")
					if _, ok := newComp.ConnectorComponent.Connection.GetFields()[key]; ok {
						newComp.ConnectorComponent.Connection.GetFields()[key] = structpb.NewStringValue(fmt.Sprintf("${secrets.%s-%s-%s}", defMap[defUID], conID, strings.ReplaceAll(key, "_", "-")))
					}

				} else {
					conID := connectorMap[conUID].ID
					conID = strings.ReplaceAll(conID, "_", "-")
					if _, ok := newComp.ConnectorComponent.Connection.GetFields()[splits[0]]; ok {
						if _, ok := newComp.ConnectorComponent.Connection.GetFields()[splits[0]].GetStructValue().GetFields()[splits[1]]; ok {
							newComp.ConnectorComponent.Connection.GetFields()[splits[0]].GetStructValue().GetFields()[splits[1]] = structpb.NewStringValue(fmt.Sprintf("${secrets.%s-%s-%s}", defMap[defUID], conID, strings.ReplaceAll(key, "_", "-")))
						}
					}
				}
			}
		}
		*c = newComp
	}

}

func migratePipeline(connectorMap map[uuid.UUID]Connector) error {
	db := database.GetConnection()
	defer database.Close(db)

	var pipelines []Pipeline
	result := db.Model(&Pipeline{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var pipeline Pipeline
		if err = db.ScanRows(rows, &pipeline); err != nil {
			return err
		}
		pipelines = append(pipelines, pipeline)
	}
	for _, p := range pipelines {
		comps := []*Component{}
		reqFields := TriggerByRequestRequestFields{}
		resFields := TriggerByRequestResponseFields{}
		for compIdx := range p.Recipe.Components {
			if p.Recipe.Components[compIdx].IsStartComponent() {
				for k, v := range p.Recipe.Components[compIdx].StartComponent.Fields {
					reqFields[k] = v
				}
			} else if p.Recipe.Components[compIdx].IsEndComponent() {
				for k, v := range p.Recipe.Components[compIdx].EndComponent.Fields {
					resFields[k] = v
				}
			} else {
				c := p.Recipe.Components[compIdx]
				if c.IsConnectorComponent() {
					migrateConnectorComponent(connectorMap, c, p.Owner)
				}
				if c.IsIteratorComponent() {
					for nestedCompIdx := range p.Recipe.Components[compIdx].IteratorComponent.Components {
						nc := p.Recipe.Components[compIdx].IteratorComponent.Components[nestedCompIdx]
						if nc.IsConnectorComponent() {
							migrateConnectorComponent(connectorMap, nc, p.Owner)
						}
					}
				}
				comps = append(comps, c)
			}
		}

		if p.Recipe.Trigger == nil {
			p.Recipe.Trigger = &Trigger{
				TriggerByRequest: &TriggerByRequest{
					RequestFields:  reqFields,
					ResponseFields: resFields,
				},
			}
		}

		p.Recipe.Components = comps

		recipeJSON, _ := json.Marshal(p.Recipe)
		recipeJSONStr := string(recipeJSON)
		recipeJSONStr = strings.ReplaceAll(recipeJSONStr, "${start.", "${trigger.")
		newRecipe := &Recipe{}

		err = json.Unmarshal([]byte(recipeJSONStr), newRecipe)
		if err != nil {
			return err
		}

		p.Recipe = newRecipe

		result := db.Model(&Pipeline{}).Where("uid = ?", p.UID).Update("recipe", p.Recipe)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func migratePipelineRelease(connectorMap map[uuid.UUID]Connector) error {
	db := database.GetConnection()
	defer database.Close(db)

	var releases []PipelineRelease
	result := db.Model(&PipelineRelease{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var release PipelineRelease
		if err = db.ScanRows(rows, &release); err != nil {
			return err
		}
		releases = append(releases, release)
	}
	for _, r := range releases {
		comps := []*Component{}
		reqFields := TriggerByRequestRequestFields{}
		resFields := TriggerByRequestResponseFields{}

		p := Pipeline{}
		result := db.Model(&Pipeline{}).Where("uid = ?", r.PipelineUID).First(&p)
		if result.Error != nil {
			return result.Error
		}
		for compIdx := range r.Recipe.Components {
			if r.Recipe.Components[compIdx].IsStartComponent() {
				for k, v := range r.Recipe.Components[compIdx].StartComponent.Fields {
					reqFields[k] = v
				}
			} else if r.Recipe.Components[compIdx].IsEndComponent() {
				for k, v := range r.Recipe.Components[compIdx].EndComponent.Fields {
					resFields[k] = v
				}
			} else {
				c := r.Recipe.Components[compIdx]
				if c.IsConnectorComponent() {
					migrateConnectorComponent(connectorMap, c, p.Owner)
				}
				if c.IsIteratorComponent() {
					for nestedCompIdx := range r.Recipe.Components[compIdx].IteratorComponent.Components {
						nc := r.Recipe.Components[compIdx].IteratorComponent.Components[nestedCompIdx]
						if nc.IsConnectorComponent() {
							migrateConnectorComponent(connectorMap, nc, p.Owner)
						}
					}
				}
				comps = append(comps, c)
			}
		}
		if r.Recipe.Trigger == nil {
			r.Recipe.Trigger = &Trigger{
				TriggerByRequest: &TriggerByRequest{
					RequestFields:  reqFields,
					ResponseFields: resFields,
				},
			}
		}

		r.Recipe.Components = comps

		recipeJSON, _ := json.Marshal(r.Recipe)
		recipeJSONStr := string(recipeJSON)
		recipeJSONStr = strings.ReplaceAll(recipeJSONStr, "${start.", "${trigger.")
		newRecipe := &Recipe{}

		err = json.Unmarshal([]byte(recipeJSONStr), newRecipe)
		if err != nil {
			return err
		}

		r.Recipe = newRecipe

		result = db.Model(&PipelineRelease{}).Where("uid = ?", r.UID).Update("recipe", r.Recipe)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

// Migrate runs the 13th revision migration.
func (m *Migration) Migrate() error {
	var connectorMap map[uuid.UUID]Connector
	var err error
	if connectorMap, err = migrateSecret(); err != nil {
		return err
	}

	if err := migratePipeline(connectorMap); err != nil {
		return err
	}
	return migratePipelineRelease(connectorMap)
}

// Migration executes code along with the 13th database schema revision.
// NOTE: for new migrations, when possible, it is best to define a <version>.go
// in the `migration` package, next to the <version>_init.up.sql script. In
// that case, the migration struct can be unexported and have a descriptive
// type name.
type Migration struct{}
