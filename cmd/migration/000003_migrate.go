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

// Pipeline is the data model of the pipeline table
type Pipeline03 struct {
	datamodel.BaseDynamic
	ID          string
	Owner       string
	Description sql.NullString
	State       datamodel.PipelineState
	Recipe      *Recipe03 `gorm:"type:jsonb"`
}

func (Pipeline03) TableName() string {
	return "pipeline"
}

// Recipe is the data model of the pipeline recipe
type Recipe03 struct {
	Version    string       `json:"version,omitempty"`
	Components []*Component `json:"components,omitempty"`
}

type Component struct {
	Id             string            `json:"id,omitempty"`
	ResourceName   string            `json:"resource_name,omitempty"`
	ResourceDetail *structpb.Struct  `json:"resource_detail,omitempty"`
	Metadata       *structpb.Struct  `json:"metadata,omitempty"`
	Dependencies   map[string]string `json:"dependencies,omitempty"`
	Type           string            `json:"type,omitempty"`
}

// Scan function for custom GORM type Recipe
func (r *Recipe03) Scan(value interface{}) error {
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
func (r *Recipe03) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}

func migratePipelineRecipeUp000003() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []Pipeline03
	result := db.Model(&Pipeline03{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item Pipeline03
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
		var source *Component
		var model *Component
		var destination *Component
		for compIdx := range items[idx].Recipe.Components {
			connector, err := connectorPrivateServiceClient.LookUpConnectorResourceAdmin(context.Background(), &connectorPB.LookUpConnectorResourceAdminRequest{
				Permalink: items[idx].Recipe.Components[compIdx].ResourceName,
			})
			if err != nil {
				panic(err)
			}

			if connector.ConnectorResource.ConnectorType == connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE {
				source = items[idx].Recipe.Components[compIdx]
			}
			if connector.ConnectorResource.ConnectorType == connectorPB.ConnectorType_CONNECTOR_TYPE_AI {
				model = items[idx].Recipe.Components[compIdx]
			}
			if connector.ConnectorResource.ConnectorType == connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION {
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
				"texts":           fmt.Sprintf("[*%s.texts]", source.Id),
				"images":          fmt.Sprintf("[*%s.images]", source.Id),
				"structured_data": fmt.Sprintf("{**%s.structured_data}", source.Id),
				"metadata":        fmt.Sprintf("{**%s.metadata}", source.Id),
			}
		}
		if destination.Dependencies == nil {
			destination.Dependencies = map[string]string{
				"texts":           fmt.Sprintf("[*%s.texts]", model.Id),
				"images":          fmt.Sprintf("[*%s.images]", model.Id),
				"structured_data": fmt.Sprintf("{**%s.structured_data}", model.Id),
				"metadata":        fmt.Sprintf("{**%s.metadata}", model.Id),
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
