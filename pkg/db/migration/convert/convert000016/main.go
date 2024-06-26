package convert000016

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v3"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

const Iterator = "iterator"

// Pipeline is the data model of the pipeline table
type Pipeline struct {
	UID    uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ID     string
	Owner  string
	Recipe *Recipe `gorm:"type:jsonb"`
}

func (Pipeline) TableName() string {
	return "pipeline"
}

// PipelineRelease is the data model of the pipeline release table
type PipelineRelease struct {
	UID         uuid.UUID `gorm:"type:uuid;primary_key;<-:create"` // allow read and create
	ID          string
	PipelineUID uuid.UUID
	Recipe      *Recipe `gorm:"type:jsonb"`
}

func (PipelineRelease) TableName() string {
	return "pipeline_release"
}

// Recipe is the data model of the pipeline recipe
type Recipe struct {
	Version   string                `json:"version,omitempty" yaml:"version,omitempty"`
	RunOn     *RunOn                `json:"runOn,omitempty" yaml:"run-on,omitempty"`
	Component map[string]*Component `json:"component,omitempty" yaml:"component,omitempty"`
	Variable  map[string]*Variable  `json:"variable,omitempty" yaml:"variable,omitempty"`
	Secret    map[string]string     `json:"secret,omitempty" yaml:"secret,omitempty"`
	Output    map[string]*Output    `json:"output,omitempty" yaml:"output,omitempty"`
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

type Variable struct {
	Title              string `json:"title,omitempty" yaml:"title,omitempty"`
	Description        string `json:"description,omitempty" yaml:"description,omitempty"`
	InstillFormat      string `json:"instillFormat,omitempty" yaml:"instill-format,omitempty"`
	InstillUIOrder     int32  `json:"instillUiOrder,omitempty" yaml:"instill-ui-order,omitempty"`
	InstillUIMultiline bool   `json:"instillUiMultiline,omitempty" yaml:"instill-ui-multiline,omitempty"`
}

type Output struct {
	Title          string `json:"title,omitempty" yaml:"title,omitempty"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	Value          string `json:"value,omitempty" yaml:"value,omitempty"`
	InstillUIOrder int32  `json:"instillUiOrder,omitempty" yaml:"instill-ui-order,omitempty"`
}

type RunOn struct {
}

type Component struct {
	// Shared fields
	Type      string         `json:"type,omitempty" yaml:"type,omitempty"`
	Task      string         `json:"task,omitempty" yaml:"task,omitempty"`
	Input     any            `json:"input,omitempty" yaml:"input,omitempty"`
	Condition string         `json:"condition,omitempty" yaml:"condition,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"  yaml:"metadata,omitempty"`

	// Fields for regular components
	Setup map[string]any `json:"setup,omitempty" yaml:"setup,omitempty"`

	// Fields for iterators
	Component      map[string]*Component `json:"component,omitempty" yaml:"component,omitempty"`
	OutputElements map[string]string     `json:"outputElements,omitempty" yaml:"output-elements,omitempty"`
}

func updateReferenceToKebabCase(in string, compIDmap map[string]string) string {
	// Note: Update the string inside the ${} block
	// 1. Update the component ID based on the component ID mapping
	// 2. Update all the reference fields to kebab-case

	val := ""
	for {
		startIdx := strings.Index(in, "${")
		if startIdx == -1 {
			val += in
			break
		}
		val += in[:startIdx+2]
		in = in[startIdx:]
		endIdx := strings.Index(in, "}")
		if endIdx == -1 {
			val += in
			break
		}

		ref := strings.TrimSpace(in[2:endIdx])

		// We don't update the naming in variables and secrets to maintain
		// compatibility with the API request.
		if strings.HasPrefix(ref, "variable") || strings.HasPrefix(ref, "secret") {
			val += ref
		} else {
			compID := strings.Split(ref, ".")[0]
			if newCompID, ok := compIDmap[compID]; ok {
				compID = newCompID
			}
			val += compID + "." + updateStringToKebabCase(strings.Join(strings.Split(ref, ".")[1:], "."))
		}

		in = in[endIdx:]
	}
	return val
}

func updateStringToKebabCase(in string) string {
	return strings.ReplaceAll(in, "_", "-")
}

func updateToKebabCase(in any, compIDmap map[string]string) any {
	switch in := in.(type) {
	case string:
		return updateReferenceToKebabCase(in, compIDmap)

	case map[string]any:
		out := map[string]any{}
		for k, v := range in {
			newK := updateStringToKebabCase(k)
			switch v := v.(type) {
			case map[string]any:
				out[newK] = updateToKebabCase(v, compIDmap)
			case string:
				out[newK] = updateToKebabCase(v, compIDmap)
			default:
				out[newK] = v
			}
		}
		return out
	}
	return in
}
func convertToYAML(recipe *Recipe) (string, error) {
	compIDmap := map[string]string{}
	newComponentMap := map[string]*Component{}
	for id := range recipe.Component {
		newID := strings.ReplaceAll(id, "_", "-")
		if _, exist := compIDmap[newID]; exist {
			// If ID duplicated after migrate, we add a postfix
			newID = newID + "0"
		}
		compIDmap[id] = newID
		newComponentMap[newID] = recipe.Component[id]

		if recipe.Component[id].Type == Iterator {

			newIterComponentMap := map[string]*Component{}
			for nestedID := range recipe.Component[id].Component {
				newID := strings.ReplaceAll(nestedID, "_", "-")
				if _, exist := compIDmap[newID]; exist {
					// If ID duplicated after migrate, we'll add a postfix
					newID = newID + "0"
				}
				compIDmap[id] = newID
				newIterComponentMap[newID] = recipe.Component[id].Component[nestedID]
			}
			recipe.Component[id].Component = newIterComponentMap
			newComponentMap[newID] = recipe.Component[id]
		}

	}
	recipe.Component = newComponentMap

	for id := range recipe.Component {

		switch recipe.Component[id].Type {
		default:
			recipe.Component[id].Input = updateToKebabCase(recipe.Component[id].Input, compIDmap)
			recipe.Component[id].Condition = updateReferenceToKebabCase(recipe.Component[id].Condition, compIDmap)
			if recipe.Component[id].Setup != nil {
				recipe.Component[id].Setup = updateToKebabCase(recipe.Component[id].Setup, compIDmap).(map[string]any)
			}

		case Iterator:
			if recipe.Component[id].Input != nil {
				recipe.Component[id].Input = updateReferenceToKebabCase(recipe.Component[id].Input.(string), compIDmap)
			}

			for k := range recipe.Component[id].OutputElements {
				recipe.Component[id].OutputElements[updateStringToKebabCase(k)] = updateReferenceToKebabCase(recipe.Component[id].OutputElements[k], compIDmap)
			}
			for nestedID := range recipe.Component[id].Component {
				if recipe.Component[id].Component[nestedID].Type != Iterator {
					recipe.Component[id].Component[nestedID].Input = updateToKebabCase(recipe.Component[id].Component[nestedID].Input, compIDmap)
					recipe.Component[id].Component[nestedID].Condition = updateReferenceToKebabCase(recipe.Component[id].Component[nestedID].Condition, compIDmap)
				}
			}
			recipe.Component[id].Condition = updateReferenceToKebabCase(recipe.Component[id].Condition, compIDmap)
		}
	}
	for k := range recipe.Output {
		recipe.Output[k].Value = updateReferenceToKebabCase(recipe.Output[k].Value, compIDmap)
	}

	yamlStr, err := yaml.Marshal(recipe)
	if err != nil {
		return "", err
	}
	return string(yamlStr), nil
}
func migratePipeline() error {
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

		recipe := p.Recipe
		if recipe == nil {
			continue
		}

		yamlStr, err := convertToYAML(p.Recipe)
		if err != nil {
			return err
		}
		result := db.Model(&Pipeline{}).Where("uid = ?", p.UID).Update("recipe_yaml", string(yamlStr))
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func migratePipelineRelease() error {
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

		recipe := r.Recipe
		if recipe == nil {
			continue
		}
		yamlStr, err := convertToYAML(r.Recipe)
		if err != nil {
			return err
		}
		result := db.Model(&PipelineRelease{}).Where("uid = ?", r.UID).Update("recipe_yaml", string(yamlStr))
		if result.Error != nil {
			return result.Error
		}
	}
	return nil

}

func Migrate() error {

	var err error

	if err = migratePipeline(); err != nil {
		return err
	}
	if err = migratePipelineRelease(); err != nil {
		return err
	}
	return nil
}
