package qdrant

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	deleteCollectionPath = "/collections/%s"
)

type DeleteCollectionInput struct {
	CollectionName string `json:"collection-name"`
}

type DeleteCollectionOutput struct {
	Status string `json:"status"`
}

type DeleteCollectionResp struct {
	Time   float64 `json:"time"`
	Status string  `json:"status"`
	Result bool    `json:"result"`
}

func (e *execution) deleteCollection(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := DeleteCollectionResp{}

	reqParams := make(map[string]any)

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Delete(fmt.Sprintf(deleteCollectionPath, inputStruct.CollectionName))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to delete collection: %s", res.String())
	}

	outputStruct := DeleteCollectionOutput{
		Status: "Successfully deleted 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
