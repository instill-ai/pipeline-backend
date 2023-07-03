package main

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func migratePipelineRecipeUp000003() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []datamodel.Pipeline
	result := db.Unscoped().Model(&datamodel.Pipeline{})
	if result.Error != nil {
		return result.Error
	}

	rows, err := result.Rows()
	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var item datamodel.Pipeline
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
		var source *datamodel.Component
		var model *datamodel.Component
		var destination *datamodel.Component
		for compIdx := range items[idx].Recipe.Components {
			connector, err := connectorPrivateServiceClient.LookUpConnectorAdmin(context.Background(), &connectorPB.LookUpConnectorAdminRequest{
				Permalink: items[idx].Recipe.Components[compIdx].ResourceName,
			})
			if err != nil {
				panic(err)
			}

			if connector.Connector.ConnectorType == connectorPB.ConnectorType_CONNECTOR_TYPE_SOURCE {
				source = items[idx].Recipe.Components[compIdx]
			}
			if connector.Connector.ConnectorType == connectorPB.ConnectorType_CONNECTOR_TYPE_AI {
				model = items[idx].Recipe.Components[compIdx]
			}
			if connector.Connector.ConnectorType == connectorPB.ConnectorType_CONNECTOR_TYPE_DESTINATION {
				destination = items[idx].Recipe.Components[compIdx]
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
