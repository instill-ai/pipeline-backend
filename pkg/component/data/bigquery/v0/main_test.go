package bigquery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

func mockBigQueryServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		response := map[string]interface{}{
			"kind": "bigquery#queryResponse",
			"schema": map[string]interface{}{
				"fields": []map[string]interface{}{
					{"name": "field1", "type": "STRING"},
				},
			},
			"jobComplete": true,
			"rows": []map[string]interface{}{
				{"f": []map[string]interface{}{{"v": "row_value"}}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

func TestExecuteQueryWithMockServer(t *testing.T) {

	server := mockBigQueryServer()
	defer server.Close()

	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, "test-project", option.WithEndpoint(server.URL), option.WithoutAuthentication())
	assert.NoError(t, err)
	assert.NotNil(t, client)

	query := client.Query("SELECT * FROM test_table")
	it, err := query.Read(ctx)
	assert.NoError(t, err)

	var values []bigquery.Value
	err = it.Next(&values)
	assert.NoError(t, err)
	assert.Equal(t, []bigquery.Value{"row_value"}, values)
}
