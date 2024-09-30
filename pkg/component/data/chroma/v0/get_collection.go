package chroma

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	getCollectionPath = "/api/v1/collections/%s"
)

type GetCollectionResp struct {
	ID string `json:"id"`

	Detail []map[string]any `json:"detail"`
}

func getCollectionID(collectionName string, client *httpclient.Client) (string, error) {
	respGetColl := GetCollectionResp{}

	reqGetColl := client.R().SetResult(&respGetColl)

	resGetColl, err := reqGetColl.Get(fmt.Sprintf(getCollectionPath, collectionName))

	if err != nil {
		return "", err
	}

	if resGetColl.StatusCode() != 200 {
		return "", fmt.Errorf("failed to get collection: %s", resGetColl.String())
	}

	if respGetColl.Detail != nil {
		return "", fmt.Errorf("failed to get collection: %s", respGetColl.Detail[0]["msg"])
	}

	return respGetColl.ID, nil
}
