package milvus

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	createIndexPath = "/v2/vectordb/indexes/create"
)

type CreateIndexOutput struct {
	Status string `json:"status"`
}

type CreateIndexInput struct {
	CollectionName string         `json:"collection-name"`
	IndexParams    map[string]any `json:"index-params"`
}

type CreateIndexReq struct {
	CollectionName string           `json:"collectionName"`
	IndexParams    []map[string]any `json:"indexParams"`
}

type CreateIndexResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *execution) createIndex(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateIndexInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := CreateIndexResp{}

	reqParams := CreateIndexReq{
		CollectionName: inputStruct.CollectionName,
		IndexParams:    []map[string]any{inputStruct.IndexParams},
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(createIndexPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create index: %s", res.String())
	}

	if resp.Message != "" && resp.Code != 0 {
		return nil, fmt.Errorf("failed to create index: %s", resp.Message)
	}

	outputStruct := CreateIndexOutput{
		Status: "Successfully created 1 index",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil

}
