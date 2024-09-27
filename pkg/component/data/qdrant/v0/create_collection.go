package qdrant

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	createCollectionPath = "/collections/%s"
)

type CreateCollectionInput struct {
	CollectionName string         `json:"collection-name"`
	Config         map[string]any `json:"config"`
}

type CreateCollectionOutput struct {
	Status string `json:"status"`
}

type CreateCollectionResp struct {
	Time   float64 `json:"time"`
	Status string  `json:"status"`
	Result bool    `json:"result"`
}

func (e *execution) createCollection(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := CreateCollectionResp{}

	reqParams := inputStruct.Config

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Put(fmt.Sprintf(createCollectionPath, inputStruct.CollectionName))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create collection: %s", res.String())
	}

	outputStruct := CreateCollectionOutput{
		Status: "Successfully created 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
