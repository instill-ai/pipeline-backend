package bigquery

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/protobuf/types/known/structpb"
)

type DataSaver struct {
	Schema  bigquery.Schema
	DataMap map[string]bigquery.Value
}

func (v DataSaver) Save() (row map[string]bigquery.Value, insertID string, err error) {
	return v.DataMap, bigquery.NoDedupeID, nil
}

func insertDataToBigQuery(projectID, datasetID, tableName string, valueSaver DataSaver, client *bigquery.Client) error {
	ctx := context.Background()
	tableRef := client.Dataset(datasetID).Table(tableName)
	inserter := tableRef.Inserter()
	if err := inserter.Put(ctx, valueSaver); err != nil {
		return fmt.Errorf("error inserting data: %v", err)
	}
	fmt.Printf("Data inserted into %s.%s.%s.\n", projectID, datasetID, tableName)
	return nil
}

func getDataSaver(input *structpb.Struct, schema bigquery.Schema) (DataSaver, error) {
	inputObj := input.GetFields()["data"].GetStructValue()
	dataMap := map[string]bigquery.Value{}
	for _, sc := range schema {
		dataMap[sc.Name] = inputObj.GetFields()[sc.Name].AsInterface()
	}
	return DataSaver{Schema: schema, DataMap: dataMap}, nil
}
