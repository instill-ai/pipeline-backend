package chroma

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	deleteCollectionPath = "/api/v1/collections/%s"
)

type DeleteCollectionOutput struct {
	Status string `json:"status"`
}

type DeleteCollectionInput struct {
	CollectionName string `json:"collection-name"`
}

type DeleteCollectionResp struct {
	Detail []map[string]any `json:"detail"`
}

func (e *execution) deleteCollection(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := DeleteCollectionResp{}

	req := e.client.R().SetResult(&resp)

	res, err := req.Delete(fmt.Sprintf(deleteCollectionPath, inputStruct.CollectionName))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to delete collection: %s", res.String())
	}

	if resp.Detail != nil {
		return nil, fmt.Errorf("failed to delete collection: %s", resp.Detail[0]["msg"])
	}

	outputStruct := CreateCollectionOutput{
		Status: "Successfully deleted 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
