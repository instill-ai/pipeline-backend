package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

// Pipeline is the data model of the pipeline table
type Pipeline06 struct {
	datamodel.BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	Recipe      *Recipe06 `gorm:"type:jsonb"`
}

func (Pipeline06) TableName() string {
	return "pipeline"
}

// Recipe is the data model of the pipeline recipe
type Recipe06 struct {
	Version    string         `json:"version,omitempty"`
	Components []*Component06 `json:"components,omitempty"`
}

type Component06 struct {
	Id             string           `json:"id"`
	DefinitionName string           `json:"definition_name"`
	ResourceName   string           `json:"resource_name"`
	Configuration  *structpb.Struct `json:"configuration"`
}

// Scan function for custom GORM type Recipe
func (r *Recipe06) Scan(value interface{}) error {
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
func (r *Recipe06) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

func migratePipelineRecipeUp000006() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []Pipeline06
	result := db.Model(&Pipeline06{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item Pipeline06
		if err = db.ScanRows(rows, &item); err != nil {
			return err
		}
		items = append(items, item)

	}

	for idx := range items {
		fmt.Printf("migrate %s\n", items[idx].UID)

		updateErr := db.Transaction(func(tx *gorm.DB) error {
			for idx := range items {

				for compIdx := range items[idx].Recipe.Components {
					if _, ok := items[idx].Recipe.Components[compIdx].Configuration.Fields["input"]; ok {
						if task, ok := items[idx].Recipe.Components[compIdx].Configuration.Fields["input"].GetStructValue().Fields["task"]; ok {
							items[idx].Recipe.Components[compIdx].Configuration.Fields["task"] = structpb.NewStringValue(task.GetStringValue())
						}
					}

				}

				result := tx.Unscoped().Model(&datamodel.Pipeline{}).Where("uid = ?", items[idx].UID).Update("recipe", &items[idx].Recipe)
				if result.Error != nil {
					return result.Error
				}
			}
			return nil
		})
		if updateErr != nil {
			return updateErr
		}

	}

	return nil
}
