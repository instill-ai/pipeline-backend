package milvus

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	dropIndexPath = "/v2/vectordb/indexes/drop"
)

type DropIndexOutput struct {
	Status string `json:"status"`
}

type DropIndexInput struct {
	CollectionName string `json:"collection-name"`
	IndexName      string `json:"index-name"`
}

type DropIndexReq struct {
	CollectionNameReq string `json:"collectionName"`
	IndexNameReq      string `json:"indexName"`
}

type DropIndexResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *execution) dropIndex(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropIndexInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := DropIndexResp{}

	reqParams := DropIndexReq{
		CollectionNameReq: inputStruct.CollectionName,
		IndexNameReq:      inputStruct.IndexName,
	}

	if e.Setup.Fields["username"].GetStringValue() != "mock-root" {
		err := releaseCollection(e.client, inputStruct.CollectionName)
		if err != nil {
			return nil, err
		}
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(dropIndexPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to drop index: %s", res.String())
	}

	if resp.Message != "" && resp.Code != 0 {
		return nil, fmt.Errorf("failed to drop index: %s", resp.Message)
	}

	outputStruct := DropIndexOutput{
		Status: "Successfully dropped 1 index",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil

}
