package legacy

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

type pipeline06 struct {
	datamodel.BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	Recipe      *recipe06 `gorm:"type:jsonb"`
}

func (pipeline06) TableName() string {
	return "pipeline"
}

// Recipe is the data model of the pipeline recipe
type recipe06 struct {
	Version    string         `json:"version,omitempty"`
	Components []*component06 `json:"components,omitempty"`
}

type component06 struct {
	ID             string           `json:"id"`
	DefinitionName string           `json:"definition_name"`
	ResourceName   string           `json:"resource_name"`
	Configuration  *structpb.Struct `json:"configuration"`
}

// Scan function for custom GORM type Recipe
func (r *recipe06) Scan(value interface{}) error {
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
func (r *recipe06) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

func MigratePipelineRecipeUp000006() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []pipeline06
	result := db.Model(&pipeline06{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item pipeline06
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
