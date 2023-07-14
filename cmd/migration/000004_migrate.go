package main

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gorm"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	utils "github.com/instill-ai/pipeline-backend/pkg/utils"
	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
)

func migratePipelineRecipeUp000004() error {
	db := database.GetConnection()
	defer database.Close(db)

	var items []datamodel.Pipeline
	result := db.Model(&datamodel.Pipeline{})
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
	connectorPublicServiceClient, connectorPublicServiceClientConn := external.InitConnectorPublicServiceClient(context.Background())
	if connectorPublicServiceClientConn != nil {
		defer connectorPublicServiceClientConn.Close()
	}

	for idx := range items {
		needUpgrade := false

		for compIdx := range items[idx].Recipe.Components {

			connectorResp, err := connectorPrivateServiceClient.LookUpConnectorAdmin(context.Background(), &connectorPB.LookUpConnectorAdminRequest{
				Permalink: items[idx].Recipe.Components[compIdx].ResourceName,
			})
			if err != nil {
				panic(err)
			}
			if connectorResp.Connector.Name == "connectors/source-grpc-deprected" {
				triggerConnectorResp, err := connectorPublicServiceClient.GetConnector(utils.InjectOwnerToContextWithOwnerPermalink(context.Background(), items[idx].Owner), &connectorPB.GetConnectorRequest{
					Name: "connectors/trigger",
				})
				uid := ""
				if err != nil {
					createResp, _ := connectorPublicServiceClient.CreateConnector(utils.InjectOwnerToContextWithOwnerPermalink(context.Background(), items[idx].Owner), &connectorPB.CreateConnectorRequest{
						Connector: &connectorPB.Connector{
							Id:                      "trigger",
							ConnectorDefinitionName: "connector-definitions/trigger",
						},
					})
					uid = createResp.Connector.Uid
				} else {
					uid = triggerConnectorResp.Connector.Uid
				}
				items[idx].Recipe.Components[compIdx].ResourceName = fmt.Sprintf("connectors/%s", uid)
				needUpgrade = true
			}
			if connectorResp.Connector.Name == "connectors/destination-grpc-deprected" {
				responseConnectorResp, err := connectorPublicServiceClient.GetConnector(utils.InjectOwnerToContextWithOwnerPermalink(context.Background(), items[idx].Owner), &connectorPB.GetConnectorRequest{
					Name: "connectors/response",
				})
				uid := ""
				if err != nil {
					createResp, _ := connectorPublicServiceClient.CreateConnector(utils.InjectOwnerToContextWithOwnerPermalink(context.Background(), items[idx].Owner), &connectorPB.CreateConnectorRequest{
						Connector: &connectorPB.Connector{
							Id:                      "response",
							ConnectorDefinitionName: "connector-definitions/response",
							Configuration:           &structpb.Struct{},
						},
					})
					uid = createResp.Connector.Uid
				} else {
					uid = responseConnectorResp.Connector.Uid
				}
				items[idx].Recipe.Components[compIdx].ResourceName = fmt.Sprintf("connectors/%s", uid)
				needUpgrade = true
			}

		}

		if needUpgrade {
			fmt.Printf("migrate %s\n", items[idx].UID)
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

	}

	return nil
}
