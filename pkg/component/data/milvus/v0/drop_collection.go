package milvus

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	dropCollectionPath    = "/v2/vectordb/collections/drop"
	releaseCollectionPath = "/v2/vectordb/collections/release"
)

type DropCollectionOutput struct {
	Status string `json:"status"`
}

type DropCollectionInput struct {
	CollectionName string `json:"collection-name"`
}

type DropCollectionReq struct {
	CollectionNameReq string `json:"collectionName"`
}

type DropCollectionResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (e *execution) dropCollection(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := DropCollectionResp{}

	reqParams := DropCollectionReq{
		CollectionNameReq: inputStruct.CollectionName,
	}

	if e.Setup.Fields["username"].GetStringValue() != "mock-root" {
		err := releaseCollection(e.client, inputStruct.CollectionName)
		if err != nil {
			return nil, err
		}
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(dropCollectionPath)

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to drop collection: %s", res.String())
	}

	if err != nil {
		return nil, err
	}

	outputStruct := DropCollectionOutput{
		Status: "Successfully dropped 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
