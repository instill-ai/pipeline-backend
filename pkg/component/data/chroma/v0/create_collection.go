package chroma

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	createCollectionPath = "/api/v1/collections"
)

type CreateCollectionOutput struct {
	Status string `json:"status"`
}

type CreateCollectionInput struct {
	CollectionName string         `json:"collection-name"`
	Configuration  map[string]any `json:"configuration"`
	Metadata       map[string]any `json:"metadata"`
	GetOrCreate    bool           `json:"get-or-create"`
}

type CreateCollectionReq struct {
	Name          string         `json:"name"`
	Configuration map[string]any `json:"configuration"`
	Metadata      map[string]any `json:"metadata"`
	GetOrCreate   bool           `json:"get_or_create"`
}

type CreateCollectionResp struct {
	Detail []map[string]any `json:"detail"`
}

func (e *execution) createCollection(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := CreateCollectionResp{}

	reqParams := CreateCollectionReq{
		Name:        inputStruct.CollectionName,
		GetOrCreate: inputStruct.GetOrCreate,
	}
	if inputStruct.Metadata != nil {
		reqParams.Metadata = inputStruct.Metadata
	}
	if inputStruct.Configuration != nil {
		reqParams.Configuration = inputStruct.Configuration
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(createCollectionPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create collection: %s", res.String())
	}

	if resp.Detail != nil {
		return nil, fmt.Errorf("failed to create collection: %s", resp.Detail[0]["msg"])
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
