package main

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

type pipeline12 struct {
	datamodel.BaseDynamic
	ID     string
	Recipe *recipe12 `gorm:"type:jsonb"`
}

func (pipeline12) TableName() string {
	return "pipeline"
}

type pipelineRelease07 struct {
	datamodel.BaseDynamic
	ID     string
	Recipe *recipe07 `gorm:"type:jsonb"`
}

func (pipelineRelease07) TableName() string {
	return "pipeline_release"
}

type pipelineRelease12 struct {
	datamodel.BaseDynamic
	ID     string
	Recipe *recipe12 `gorm:"type:jsonb"`
}

func (pipelineRelease12) TableName() string {
	return "pipeline_release"
}

// Recipe is the data model of the pipeline recipe
type recipe12 struct {
	Version    string        `json:"version,omitempty"`
	Components []component12 `json:"components,omitempty"`
}

type component12 struct {
	ID       string         `json:"id"`
	Metadata datatypes.JSON `json:"metadata"`
	// TODO: validate oneof
	StartComponent     *startComponent     `json:"start_component,omitempty"`
	EndComponent       *endComponent       `json:"end_component,omitempty"`
	ConnectorComponent *connectorComponent `json:"connector_component,omitempty"`
	OperatorComponent  *operatorComponent  `json:"operator_component,omitempty"`
	IteratorComponent  *iteratorComponent  `json:"iterator_component,omitempty"`
}

type startComponentField struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	InstillFormat  string `json:"instill_format"`
	InstillUIOrder int32  `json:"instill_ui_order"`
}
type startComponent struct {
	Fields map[string]startComponentField `json:"fields"`
}

type endComponentField struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Value          string `json:"value"`
	InstillUIOrder int32  `json:"instill_ui_order"`
}

type endComponent struct {
	Fields map[string]endComponentField `json:"fields"`
}

type connectorComponent struct {
	DefinitionName string           `json:"definition_name"`
	ConnectorName  string           `json:"connector_name"`
	Task           string           `json:"task"`
	Input          *structpb.Struct `json:"input"`
	Condition      *string          `json:"condition,omitempty"`
}

type operatorComponent struct {
	DefinitionName string           `json:"definition_name"`
	Task           string           `json:"task"`
	Input          *structpb.Struct `json:"input"`
	Condition      *string          `json:"condition,omitempty"`
}

type iteratorComponent struct {
	Input          string            `json:"input"`
	OutputElements map[string]string `json:"output_elements"`
	Condition      *string           `json:"condition,omitempty"`
	Components     []*component12    `json:"components"`
}

// Scan function for custom GORM type Recipe
func (r *recipe12) Scan(value interface{}) error {
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
func (r *recipe12) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

func migratePipelineRecipeUp000012() error {
	db := database.GetConnection()
	defer database.Close(db)

	{
		var pipelines []pipeline07
		result := db.Model(&pipeline07{})
		if result.Error != nil {
			return result.Error
		}

		rows, err := result.Rows()
		if err != nil {
			return err
		}

		defer rows.Close()

		for rows.Next() {
			var pipeline pipeline07
			if err = db.ScanRows(rows, &pipeline); err != nil {
				return err
			}
			pipelines = append(pipelines, pipeline)

		}

		for idx := range pipelines {
			fmt.Printf("migrate %s\n", pipelines[idx].UID)

			updateErr := db.Transaction(func(tx *gorm.DB) error {

				newRecipe := &recipe12{Version: pipelines[idx].Recipe.Version, Components: []component12{}}
				for compIdx := range pipelines[idx].Recipe.Components {
					oldComp := pipelines[idx].Recipe.Components[compIdx]
					newComp := component12{ID: pipelines[idx].Recipe.Components[compIdx].ID}

					switch {
					// start
					case oldComp.DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3":

						newComp.StartComponent = &startComponent{Fields: make(map[string]startComponentField)}

						for k, v := range oldComp.Configuration.Fields["metadata"].GetStructValue().Fields {
							newComp.StartComponent.Fields[k] = startComponentField{
								Title:          v.GetStructValue().Fields["title"].GetStringValue(),
								Description:    v.GetStructValue().Fields["description"].GetStringValue(),
								InstillFormat:  v.GetStructValue().Fields["instillFormat"].GetStringValue(),
								InstillUIOrder: int32(v.GetStructValue().Fields["instillUiOrder"].GetNumberValue()),
							}
						}

					// end
					case oldComp.DefinitionName == "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516":

						newComp.EndComponent = &endComponent{Fields: make(map[string]endComponentField)}

						for k, v := range oldComp.Configuration.Fields["metadata"].GetStructValue().Fields {
							newComp.EndComponent.Fields[k] = endComponentField{
								Title:          v.GetStructValue().Fields["title"].GetStringValue(),
								Description:    v.GetStructValue().Fields["description"].GetStringValue(),
								Value:          oldComp.Configuration.Fields["input"].GetStructValue().Fields[k].GetStringValue(),
								InstillUIOrder: int32(v.GetStructValue().Fields["instillUiOrder"].GetNumberValue()),
							}
						}

					case strings.HasPrefix(oldComp.DefinitionName, "connector"):
						c := oldComp.Configuration.Fields["condition"].GetStringValue()
						newComp.ConnectorComponent = &connectorComponent{
							DefinitionName: oldComp.DefinitionName,
							ConnectorName:  oldComp.ResourceName,
							Input:          oldComp.Configuration.Fields["input"].GetStructValue(),
							Task:           oldComp.Configuration.Fields["task"].GetStringValue(),
							Condition:      &c,
						}
					case strings.HasPrefix(oldComp.DefinitionName, "operator"):
						c := oldComp.Configuration.Fields["condition"].GetStringValue()
						newComp.OperatorComponent = &operatorComponent{
							DefinitionName: oldComp.DefinitionName,
							Input:          oldComp.Configuration.Fields["input"].GetStructValue(),
							Task:           oldComp.Configuration.Fields["task"].GetStringValue(),
							Condition:      &c,
						}
					}
					newRecipe.Components = append(newRecipe.Components, newComp)
				}

				result := tx.Unscoped().Model(&pipeline12{}).Where("uid = ?", pipelines[idx].UID).Update("recipe", newRecipe)
				if result.Error != nil {
					return result.Error
				}

				return nil
			})
			if updateErr != nil {
				return updateErr
			}

		}
	}

	{
		var releases []pipelineRelease07
		result := db.Model(&pipelineRelease07{})
		if result.Error != nil {
			return result.Error
		}

		rows, err := result.Rows()
		if err != nil {
			return err
		}

		defer rows.Close()

		for rows.Next() {
			var release pipelineRelease07
			if err = db.ScanRows(rows, &release); err != nil {
				return err
			}
			releases = append(releases, release)

		}

		for idx := range releases {
			fmt.Printf("migrate %s\n", releases[idx].UID)

			updateErr := db.Transaction(func(tx *gorm.DB) error {

				newRecipe := &recipe12{Version: releases[idx].Recipe.Version, Components: []component12{}}
				for compIdx := range releases[idx].Recipe.Components {
					oldComp := releases[idx].Recipe.Components[compIdx]
					newComp := component12{ID: releases[idx].Recipe.Components[compIdx].ID}

					switch {
					// start
					case oldComp.DefinitionName == "operator-definitions/2ac8be70-0f7a-4b61-a33d-098b8acfa6f3":

						newComp.StartComponent = &startComponent{Fields: make(map[string]startComponentField)}

						for k, v := range oldComp.Configuration.Fields["metadata"].GetStructValue().Fields {
							newComp.StartComponent.Fields[k] = startComponentField{
								Title:          v.GetStructValue().Fields["title"].GetStringValue(),
								Description:    v.GetStructValue().Fields["description"].GetStringValue(),
								InstillFormat:  v.GetStructValue().Fields["instillFormat"].GetStringValue(),
								InstillUIOrder: int32(v.GetStructValue().Fields["instillUiOrder"].GetNumberValue()),
							}
						}

					// end
					case oldComp.DefinitionName == "operator-definitions/4f39c8bc-8617-495d-80de-80d0f5397516":

						newComp.EndComponent = &endComponent{Fields: make(map[string]endComponentField)}

						for k, v := range oldComp.Configuration.Fields["metadata"].GetStructValue().Fields {
							newComp.EndComponent.Fields[k] = endComponentField{
								Title:          v.GetStructValue().Fields["title"].GetStringValue(),
								Description:    v.GetStructValue().Fields["description"].GetStringValue(),
								Value:          oldComp.Configuration.Fields["input"].GetStructValue().Fields[k].GetStringValue(),
								InstillUIOrder: int32(v.GetStructValue().Fields["instillUiOrder"].GetNumberValue()),
							}
						}

					case strings.HasPrefix(oldComp.DefinitionName, "connector"):
						c := oldComp.Configuration.Fields["condition"].GetStringValue()
						newComp.ConnectorComponent = &connectorComponent{
							DefinitionName: oldComp.DefinitionName,
							ConnectorName:  oldComp.ResourceName,
							Input:          oldComp.Configuration.Fields["input"].GetStructValue(),
							Task:           oldComp.Configuration.Fields["task"].GetStringValue(),
							Condition:      &c,
						}
					case strings.HasPrefix(oldComp.DefinitionName, "operator"):
						c := oldComp.Configuration.Fields["condition"].GetStringValue()
						newComp.OperatorComponent = &operatorComponent{
							DefinitionName: oldComp.DefinitionName,
							Input:          oldComp.Configuration.Fields["input"].GetStructValue(),
							Task:           oldComp.Configuration.Fields["task"].GetStringValue(),
							Condition:      &c,
						}
					}
					newRecipe.Components = append(newRecipe.Components, newComp)
				}

				result := tx.Unscoped().Model(&pipelineRelease12{}).Where("uid = ?", releases[idx].UID).Update("recipe", newRecipe)
				if result.Error != nil {
					return result.Error
				}

				return nil
			})
			if updateErr != nil {
				return updateErr
			}

		}
	}

	return nil
}
