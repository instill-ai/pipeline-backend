package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

type pipeline07 struct {
	datamodel.BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	Recipe      *recipe07 `gorm:"type:jsonb"`
}

func (pipeline07) TableName() string {
	return "pipeline"
}

// Recipe is the data model of the pipeline recipe
type recipe07 struct {
	Version    string         `json:"version,omitempty"`
	Components []*component07 `json:"components,omitempty"`
}

type component07 struct {
	ID             string           `json:"id"`
	DefinitionName string           `json:"definition_name"`
	ResourceName   string           `json:"resource_name"`
	Configuration  *structpb.Struct `json:"configuration"`
}

// Scan function for custom GORM type Recipe
func (r *recipe07) Scan(value interface{}) error {
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
func (r *recipe07) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

func migratePipelineRecipeUp000007() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []pipeline07
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
		var item pipeline07
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
					if items[idx].Recipe.Components[compIdx].ID == "start" {
						if metadata, ok := items[idx].Recipe.Components[compIdx].Configuration.Fields["metadata"]; ok {
							for k := range metadata.GetStructValue().Fields {
								vType := metadata.GetStructValue().Fields[k].GetStructValue().Fields["type"].GetStringValue()
								instillFormat := ""
								switch vType {
								case "number", "integer", "boolean", "string":
									instillFormat = vType
								case "text":
									instillFormat = "string"
									vType = "string"
								case "audio", "image", "video":
									instillFormat = fmt.Sprintf("%s/*", vType)
									vType = "string"

								case "text_array":
									instillFormat = "array:string"
									items := &structpb.Struct{Fields: map[string]*structpb.Value{}}
									items.Fields["type"] = structpb.NewStringValue("string")
									metadata.GetStructValue().Fields[k].GetStructValue().Fields["items"] = structpb.NewStructValue(items)
									vType = "array"

								case "audio_array", "image_array", "video_array":
									instillFormat = fmt.Sprintf("array:%s/*", strings.Split(vType, "_")[0])
									items := &structpb.Struct{Fields: map[string]*structpb.Value{}}
									items.Fields["type"] = structpb.NewStringValue("string")
									metadata.GetStructValue().Fields[k].GetStructValue().Fields["items"] = structpb.NewStructValue(items)
									vType = "array"

								case "number_array", "integer_array", "boolean_array":
									instillFormat = fmt.Sprintf("array:%s", strings.Split(vType, "_")[0])
									items := &structpb.Struct{Fields: map[string]*structpb.Value{}}
									items.Fields["type"] = structpb.NewStringValue(strings.Split(vType, "_")[0])
									metadata.GetStructValue().Fields[k].GetStructValue().Fields["items"] = structpb.NewStructValue(items)
									vType = "array"

								}
								metadata.GetStructValue().Fields[k].GetStructValue().Fields["type"] = structpb.NewStringValue(vType)
								metadata.GetStructValue().Fields[k].GetStructValue().Fields["instillFormat"], _ = structpb.NewValue(instillFormat)

							}
							items[idx].Recipe.Components[compIdx].Configuration.Fields["metadata"] = metadata
							continue
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
