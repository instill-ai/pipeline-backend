package convert000015

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"gorm.io/datatypes"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/v1beta"
)

// Pipeline is the data model of the pipeline table
type OldPipeline struct {
	UID    uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ID     string
	Owner  string
	Recipe *OldRecipe `gorm:"type:jsonb"`
}

func (OldPipeline) TableName() string {
	return "pipeline"
}

// PipelineRelease is the data model of the pipeline release table
type OldPipelineRelease struct {
	UID         uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ID          string
	PipelineUID uuid.UUID
	Recipe      *OldRecipe `gorm:"type:jsonb"`
}

func (OldPipelineRelease) TableName() string {
	return "pipeline_release"
}

type OldRecipe struct {
	Version    string          `json:"version,omitempty"`
	Trigger    *OldTrigger     `json:"trigger,omitempty"`
	Components []*OldComponent `json:"components,omitempty"`
}

type OldTriggerByRequestRequestFields map[string]struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	InstillFormat      string `json:"instill_format"`
	InstillUIOrder     int32  `json:"instill_ui_order"`
	InstillUIMultiline bool   `json:"instill_ui_multiline"`
}

type OldTriggerByRequestResponseFields map[string]struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Value          string `json:"value"`
	InstillUIOrder int32  `json:"instill_ui_order"`
}

type OldTriggerByRequest struct {
	RequestFields  OldTriggerByRequestRequestFields  `json:"request_fields"`
	ResponseFields OldTriggerByRequestResponseFields `json:"response_fields"`
}

type OldTrigger struct {
	TriggerByRequest *OldTriggerByRequest `json:"trigger_by_request,omitempty"`
}

type OldComponent struct {
	ID                 string                 `json:"id"`
	Metadata           map[string]any         `json:"metadata"`
	ConnectorComponent *OldConnectorComponent `json:"connector_component,omitempty"`
	OperatorComponent  *OldOperatorComponent  `json:"operator_component,omitempty"`
	IteratorComponent  *OldIteratorComponent  `json:"iterator_component,omitempty"`
}

type OldStartComponent struct {
	Fields map[string]struct {
		Title              string `json:"title"`
		Description        string `json:"description"`
		InstillFormat      string `json:"instill_format"`
		InstillUIOrder     int32  `json:"instill_ui_order"`
		InstillUIMultiline bool   `json:"instill_ui_multiline"`
	} `json:"fields"`
}

type OldEndComponent struct {
	Fields map[string]struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		Value          string `json:"value"`
		InstillUIOrder int32  `json:"instill_ui_order"`
	} `json:"fields"`
}

type OldConnectorComponent struct {
	DefinitionName string         `json:"definition_name"`
	ConnectorName  string         `json:"connector_name,omitempty"`
	Task           string         `json:"task"`
	Input          map[string]any `json:"input"`
	Condition      *string        `json:"condition,omitempty"`
	Connection     map[string]any `json:"connection"`
}

type OldOperatorComponent struct {
	DefinitionName string         `json:"definition_name"`
	Task           string         `json:"task"`
	Input          map[string]any `json:"input"`
	Condition      *string        `json:"condition,omitempty"`
}

type OldIteratorComponent struct {
	Input          string            `json:"input"`
	OutputElements map[string]string `json:"output_elements"`
	Condition      *string           `json:"condition,omitempty"`
	Components     []*OldComponent   `json:"components"`
}

// Scan function for custom GORM type Recipe
func (r *OldRecipe) Scan(value interface{}) error {
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
func (r *OldRecipe) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
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

// Recipe is the data model of the pipeline recipe
type Recipe struct {
	Version   string                `json:"version,omitempty"`
	On        *On                   `json:"on,omitempty"`
	Component map[string]IComponent `json:"component,omitempty"`
	Variable  map[string]*Variable  `json:"variable,omitempty"`
	Secret    map[string]string     `json:"secret,omitempty"`
	Output    map[string]*Output    `json:"output,omitempty"`
}

type IComponent interface{}

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
					c := ComponentConfig{}
					_ = json.Unmarshal(b, &c)
					r.Component[id] = &c
				}
			}
		}
	}

	return nil
}

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

type ComponentConfig struct {
	Type       string                          `json:"type,omitempty"`
	Task       string                          `json:"task,omitempty"`
	Input      map[string]any                  `json:"input,omitempty"`
	Condition  *string                         `json:"condition,omitempty"`
	Setup      map[string]any                  `json:"setup,omitempty"`
	Metadata   map[string]any                  `json:"metadata,omitempty"`
	Definition *pipelinepb.ComponentDefinition `json:"definition,omitempty"`
}

type IteratorComponent struct {
	Type           string                `json:"type,omitempty"`
	Input          string                `json:"input,omitempty"`
	OutputElements map[string]string     `json:"outputElements,omitempty"`
	Condition      *string               `json:"condition,omitempty"`
	Component      map[string]IComponent `json:"component,omitempty"`
	Metadata       datatypes.JSON        `json:"metadata,omitempty"`
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

	if v, ok := tmp["component"]; ok && v != nil {
		comps := v.(map[string]any)
		i.Component = make(map[string]IComponent)
		for id, comp := range comps {

			if _, ok := comp.(map[string]any)["type"]; ok {

				b, _ := json.Marshal(comp)
				c := ComponentConfig{}
				_ = json.Unmarshal(b, &c)
				i.Component[id] = &c

			}

		}
	}

	return nil
}

func migratePipeline() error {
	db := database.GetConnection()
	defer database.Close(db)

	var pipelines []OldPipeline
	result := db.Model(&OldPipeline{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var pipeline OldPipeline
		if err = db.ScanRows(rows, &pipeline); err != nil {
			return err
		}
		pipelines = append(pipelines, pipeline)
	}
	for _, p := range pipelines {

		variable := map[string]*Variable{}
		output := map[string]*Output{}
		if p.Recipe.Trigger != nil {
			if p.Recipe.Trigger.TriggerByRequest != nil {
				for k, v := range p.Recipe.Trigger.TriggerByRequest.RequestFields {
					variable[k] = (*Variable)(&v)
				}
			}
			if p.Recipe.Trigger.TriggerByRequest != nil {
				for k, v := range p.Recipe.Trigger.TriggerByRequest.ResponseFields {
					output[k] = (*Output)(&v)
				}
			}
		}

		comp := map[string]IComponent{}
		for _, v := range p.Recipe.Components {
			if v.IteratorComponent != nil {
				nestedComp := map[string]IComponent{}
				for _, nv := range v.IteratorComponent.Components {
					if nv.ConnectorComponent != nil && nv.ConnectorComponent.DefinitionName != "" {
						nestedComp[nv.ID] = ComponentConfig{
							Type:      strings.Split(nv.ConnectorComponent.DefinitionName, "/")[1],
							Task:      nv.ConnectorComponent.Task,
							Input:     nv.ConnectorComponent.Input,
							Condition: nv.ConnectorComponent.Condition,
							Setup:     nv.ConnectorComponent.Connection,
							Metadata:  nv.Metadata,
						}
					} else if nv.OperatorComponent != nil && nv.OperatorComponent.DefinitionName != "" {
						nestedComp[nv.ID] = ComponentConfig{
							Type:      strings.Split(nv.OperatorComponent.DefinitionName, "/")[1],
							Task:      nv.OperatorComponent.Task,
							Input:     nv.OperatorComponent.Input,
							Condition: nv.OperatorComponent.Condition,
							Metadata:  nv.Metadata,
						}
					}

				}
				comp[v.ID] = IteratorComponent{
					Type:           "iterator",
					Input:          v.IteratorComponent.Input,
					OutputElements: v.IteratorComponent.OutputElements,
					Condition:      v.IteratorComponent.Condition,
					Component:      nestedComp,
				}
			} else if v.ConnectorComponent != nil && v.ConnectorComponent.DefinitionName != "" {
				comp[v.ID] = ComponentConfig{
					Type:      strings.Split(v.ConnectorComponent.DefinitionName, "/")[1],
					Task:      v.ConnectorComponent.Task,
					Input:     v.ConnectorComponent.Input,
					Condition: v.ConnectorComponent.Condition,
					Setup:     v.ConnectorComponent.Connection,
					Metadata:  v.Metadata,
				}
			} else if v.OperatorComponent != nil && v.OperatorComponent.DefinitionName != "" {
				comp[v.ID] = ComponentConfig{
					Type:      strings.Split(v.OperatorComponent.DefinitionName, "/")[1],
					Task:      v.OperatorComponent.Task,
					Input:     v.OperatorComponent.Input,
					Condition: v.OperatorComponent.Condition,
					Metadata:  v.Metadata,
				}
			}

		}
		newRecipe := Recipe{
			Version:   p.Recipe.Version,
			Variable:  variable,
			Output:    output,
			Component: comp,
		}

		recipeJSON, _ := json.Marshal(newRecipe)
		recipeJSONStr := string(recipeJSON)
		recipeJSONStr = strings.ReplaceAll(recipeJSONStr, "${trigger.", "${variable.")
		recipeJSONStr = strings.ReplaceAll(recipeJSONStr, "${secrets.", "${secret.")

		fmt.Println(string(recipeJSONStr))

		result := db.Model(&Pipeline{}).Where("uid = ?", p.UID).Update("recipe", recipeJSONStr)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func migratePipelineRelease() error {
	db := database.GetConnection()
	defer database.Close(db)

	var releases []OldPipelineRelease
	result := db.Model(&OldPipelineRelease{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var release OldPipelineRelease
		if err = db.ScanRows(rows, &release); err != nil {
			return err
		}
		releases = append(releases, release)
	}
	for _, r := range releases {

		variable := map[string]*Variable{}
		output := map[string]*Output{}
		if r.Recipe.Trigger != nil {
			if r.Recipe.Trigger.TriggerByRequest != nil {
				for k, v := range r.Recipe.Trigger.TriggerByRequest.RequestFields {
					variable[k] = (*Variable)(&v)
				}
			}
			if r.Recipe.Trigger.TriggerByRequest != nil {
				for k, v := range r.Recipe.Trigger.TriggerByRequest.ResponseFields {
					output[k] = (*Output)(&v)
				}
			}
		}

		comp := map[string]IComponent{}
		for _, v := range r.Recipe.Components {
			if v.IteratorComponent != nil {
				nestedComp := map[string]IComponent{}
				for _, nv := range v.IteratorComponent.Components {
					if nv.ConnectorComponent != nil && nv.ConnectorComponent.DefinitionName != "" {
						nestedComp[nv.ID] = ComponentConfig{
							Type:      strings.Split(nv.ConnectorComponent.DefinitionName, "/")[1],
							Task:      nv.ConnectorComponent.Task,
							Input:     nv.ConnectorComponent.Input,
							Condition: nv.ConnectorComponent.Condition,
							Setup:     nv.ConnectorComponent.Connection,
							Metadata:  nv.Metadata,
						}
					} else if nv.OperatorComponent != nil && nv.OperatorComponent.DefinitionName != "" {
						nestedComp[nv.ID] = ComponentConfig{
							Type:      strings.Split(nv.OperatorComponent.DefinitionName, "/")[1],
							Task:      nv.OperatorComponent.Task,
							Input:     nv.OperatorComponent.Input,
							Condition: nv.OperatorComponent.Condition,
							Metadata:  nv.Metadata,
						}
					}

				}
				comp[v.ID] = IteratorComponent{
					Type:           "iterator",
					Input:          v.IteratorComponent.Input,
					OutputElements: v.IteratorComponent.OutputElements,
					Condition:      v.IteratorComponent.Condition,
					Component:      nestedComp,
				}
			} else if v.ConnectorComponent != nil && v.ConnectorComponent.DefinitionName != "" {
				comp[v.ID] = ComponentConfig{
					Type:      strings.Split(v.ConnectorComponent.DefinitionName, "/")[1],
					Task:      v.ConnectorComponent.Task,
					Input:     v.ConnectorComponent.Input,
					Condition: v.ConnectorComponent.Condition,
					Setup:     v.ConnectorComponent.Connection,
					Metadata:  v.Metadata,
				}
			} else if v.OperatorComponent != nil && v.OperatorComponent.DefinitionName != "" {
				comp[v.ID] = ComponentConfig{
					Type:      strings.Split(v.OperatorComponent.DefinitionName, "/")[1],
					Task:      v.OperatorComponent.Task,
					Input:     v.OperatorComponent.Input,
					Condition: v.OperatorComponent.Condition,
					Metadata:  v.Metadata,
				}
			}

		}

		newRecipe := Recipe{
			Version:   r.Recipe.Version,
			Variable:  variable,
			Output:    output,
			Component: comp,
		}

		recipeJSON, _ := json.Marshal(newRecipe)
		recipeJSONStr := string(recipeJSON)
		recipeJSONStr = strings.ReplaceAll(recipeJSONStr, "${trigger.", "${variable.")
		recipeJSONStr = strings.ReplaceAll(recipeJSONStr, "${secrets.", "${secret.")

		fmt.Println(string(recipeJSONStr))

		result := db.Model(&PipelineRelease{}).Where("uid = ?", r.UID).Update("recipe", recipeJSONStr)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil

}

// Migrate runs the 15th revision migration.
func (m *Migration) Migrate() error {
	if err := migratePipeline(); err != nil {
		return err
	}

	return migratePipelineRelease()
}

// Migration executes code along with the 15th database schema revision.
type Migration struct{}
