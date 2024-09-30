package bigquery

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
)

type ReadInput struct {
	ProjectID string
	DatasetID string
	TableName string
	Client    *bigquery.Client
	Filtering string
}

type ReadOutput struct {
	Data []map[string]any `json:"data"`
}

func queryBuilder(input ReadInput) string {
	if input.Filtering == "" {
		return fmt.Sprintf("SELECT * FROM `%s.%s.%s`", input.ProjectID, input.DatasetID, input.TableName)
	}
	return fmt.Sprintf("SELECT * FROM `%s.%s.%s` %s", input.ProjectID, input.DatasetID, input.TableName, input.Filtering)
}

func readDataFromBigQuery(input ReadInput) (ReadOutput, error) {

	ctx := context.Background()
	client := input.Client

	sql := queryBuilder(input)
	q := client.Query(sql)
	it, err := q.Read(ctx)
	if err != nil {
		return ReadOutput{}, err
	}
	result := []map[string]any{}
	for {
		var values []bigquery.Value
		err := it.Next(&values)

		if err == iterator.Done {
			break
		}
		data := map[string]any{}

		for i, schema := range it.Schema {
			data[schema.Name] = values[i]
		}

		result = append(result, data)
	}

	return ReadOutput{Data: result}, nil
}
