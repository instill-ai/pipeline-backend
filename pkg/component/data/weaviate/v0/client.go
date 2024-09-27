package weaviate

import (
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"google.golang.org/protobuf/types/known/structpb"
)

func newClient(setup *structpb.Struct) *weaviate.Client {
	cfg := weaviate.Config{
		Host:       getURL(setup),
		Scheme:     "https",
		AuthConfig: auth.ApiKey{Value: getAPIKey(setup)},
		Headers:    nil,
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil
	}

	return client
}

func getURL(setup *structpb.Struct) string {
	return setup.GetFields()["url"].GetStringValue()
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()["api-key"].GetStringValue()
}
