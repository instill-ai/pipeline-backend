package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	"gorm.io/gorm"
)

type Recipe struct {
	Source      string   `json:"source,omitempty"`
	Destination string   `json:"destination,omitempty"`
	Models      []string `json:"models,omitempty"`
	Logics      []string `json:"logics,omitempty"`
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

type Pipeline struct {
	datamodel.BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	State       datamodel.PipelineState
	Recipe      *Recipe `gorm:"type:jsonb"`
}

func migrateRecipeUp(oldRecipe *Recipe) (*datamodel.Recipe, error) {

	if oldRecipe.Source == "" || oldRecipe.Destination == "" || len(oldRecipe.Models) == 0 {
		return nil, fmt.Errorf("upgrade failed: recipe error")
	}
	newRecipe := &datamodel.Recipe{}
	newRecipe.Version = "v1alpha"
	newRecipe.Components = append(
		newRecipe.Components,
		&datamodel.Component{
			Id:           "source",
			ResourceName: oldRecipe.Source,
		},
	)
	newRecipe.Components = append(
		newRecipe.Components,
		&datamodel.Component{
			Id:           "destination",
			ResourceName: oldRecipe.Destination,
		},
	)
	for idx, model := range oldRecipe.Models {
		newRecipe.Components = append(
			newRecipe.Components,
			&datamodel.Component{
				Id:           fmt.Sprintf("model_%d", idx),
				ResourceName: model,
			},
		)
	}
	return newRecipe, nil
}
func migratePipelineRecipeUp000002() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []Pipeline

	result := db.Unscoped().Model(&Pipeline{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item Pipeline
		if err = db.ScanRows(rows, &item); err != nil {
			return err
		}
		items = append(items, item)
	}
	updateErr := db.Transaction(func(tx *gorm.DB) error {
		for idx := range items {
			newRecipe, err := migrateRecipeUp(items[idx].Recipe)
			if err != nil {
				return err
			}

			result := tx.Unscoped().Model(&datamodel.Pipeline{}).Where("uid = ?", items[idx].UID).Update("recipe", &newRecipe)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
	if updateErr != nil {
		return updateErr
	}

	return nil
}
