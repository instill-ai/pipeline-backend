package elasticsearch

import (
	"github.com/elastic/go-elasticsearch/v8"
	"google.golang.org/protobuf/types/known/structpb"
)

func newClient(setup *structpb.Struct) *ESClient {
	cfg := elasticsearch.Config{
		CloudID: getCloudID(setup),
		APIKey:  getAPIKey(setup),
	}

	es, _ := elasticsearch.NewClient(cfg)

	return &ESClient{
		indexClient:        es.Index,
		searchClient:       es.Search,
		updateClient:       es.UpdateByQuery,
		deleteClient:       es.DeleteByQuery,
		createIndexClient:  es.Indices.Create,
		deleteIndexClient:  es.Indices.Delete,
		sqlTranslateClient: es.SQL.Translate,
		bulkClient:         es.Bulk,
	}
}

// Need to confirm where the map is
func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}

func getCloudID(setup *structpb.Struct) string {
	return setup.GetFields()["cloud-id"].GetStringValue()
}
