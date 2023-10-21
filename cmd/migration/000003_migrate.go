package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

type pipeline03 struct {
	datamodel.BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	Recipe      *recipe03 `gorm:"type:jsonb"`
}

func (pipeline03) TableName() string {
	return "pipeline"
}

type recipe03 struct {
	Version    string       `json:"version,omitempty"`
	Components []*component `json:"components,omitempty"`
}

type component struct {
	ID             string            `json:"id,omitempty"`
	ResourceName   string            `json:"resource_name,omitempty"`
	ResourceDetail *structpb.Struct  `json:"resource_detail,omitempty"`
	Metadata       *structpb.Struct  `json:"metadata,omitempty"`
	Dependencies   map[string]string `json:"dependencies,omitempty"`
	Type           string            `json:"type,omitempty"`
}

// Scan function for custom GORM type Recipe
func (r *recipe03) Scan(value interface{}) error {
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
func (r *recipe03) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

func migratePipelineRecipeUp000003() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []pipeline03
	result := db.Model(&pipeline03{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item pipeline03
		if err = db.ScanRows(rows, &item); err != nil {
			return err
		}
		items = append(items, item)

	}

	connectorPrivateServiceClient, connectorPrivateServiceClientConn := external.InitConnectorPrivateServiceClient(context.Background())
	if connectorPrivateServiceClientConn != nil {
		defer connectorPrivateServiceClientConn.Close()
	}
	for idx := range items {
		fmt.Printf("migrate %s\n", items[idx].UID)
		var source *component
		var model *component
		var destination *component
		for compIdx := range items[idx].Recipe.Components {
			connector, err := connectorPrivateServiceClient.LookUpConnectorResourceAdmin(context.Background(), &connectorPB.LookUpConnectorResourceAdminRequest{
				Permalink: items[idx].Recipe.Components[compIdx].ResourceName,
			})
			if err != nil {
				panic(err)
			}

			if connector.ConnectorResource.Type == connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE {
				source = items[idx].Recipe.Components[compIdx]
			}
			if connector.ConnectorResource.Type == connectorPB.ConnectorType_CONNECTOR_TYPE_AI {
				model = items[idx].Recipe.Components[compIdx]
			}
			if connector.ConnectorResource.Type == connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION {
				destination = items[idx].Recipe.Components[compIdx]
			}

		}
		source.ResourceName = "connectors/" + source.ResourceName[strings.LastIndex(source.ResourceName, "/")+1:]
		model.ResourceName = "connectors/" + model.ResourceName[strings.LastIndex(model.ResourceName, "/")+1:]
		destination.ResourceName = "connectors/" + destination.ResourceName[strings.LastIndex(destination.ResourceName, "/")+1:]

		if source.Dependencies == nil {
			source.Dependencies = map[string]string{
				"texts":           "[]",
				"images":          "[]",
				"structured_data": "{}",
				"metadata":        "{}",
			}
		}
		if model.Dependencies == nil {
			model.Dependencies = map[string]string{
				"texts":           fmt.Sprintf("[*%s.texts]", source.ID),
				"images":          fmt.Sprintf("[*%s.images]", source.ID),
				"structured_data": fmt.Sprintf("{**%s.structured_data}", source.ID),
				"metadata":        fmt.Sprintf("{**%s.metadata}", source.ID),
			}
		}
		if destination.Dependencies == nil {
			destination.Dependencies = map[string]string{
				"texts":           fmt.Sprintf("[*%s.texts]", model.ID),
				"images":          fmt.Sprintf("[*%s.images]", model.ID),
				"structured_data": fmt.Sprintf("{**%s.structured_data}", model.ID),
				"metadata":        fmt.Sprintf("{**%s.metadata}", model.ID),
			}
		}
		updateErr := db.Transaction(func(tx *gorm.DB) error {
			for idx := range items {

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
