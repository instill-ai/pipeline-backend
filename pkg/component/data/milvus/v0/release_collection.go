package milvus

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

type ReleaseCollectionReq struct {
	CollectionNameReq string `json:"collectionName"`
}

type ReleaseCollectionResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func releaseCollection(client *httpclient.Client, collectionName string) error {
	resp := ReleaseCollectionResp{}

	req := ReleaseCollectionReq{
		CollectionNameReq: collectionName,
	}

	reqReleaseCollection := client.R().SetBody(req).SetResult(&resp)

	resReleaseCollection, err := reqReleaseCollection.Post(releaseCollectionPath)

	if err != nil {
		return nil
	}

	if resReleaseCollection.StatusCode() != 200 {
		return fmt.Errorf("failed to load collection: %s", resReleaseCollection.String())
	}

	if resp.Message != "" && resp.Code != 200 {
		return fmt.Errorf("failed to load collection: %s", resp.Message)
	}

	return nil
}
