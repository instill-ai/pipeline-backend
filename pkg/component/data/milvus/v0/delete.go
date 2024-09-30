package milvus

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	deletePath = "/v2/vectordb/entities/delete"
)

type DeleteOutput struct {
	Status string `json:"status"`
}

type DeleteInput struct {
	CollectionName string `json:"collection-name"`
	PartitionName  string `json:"partition-name"`
	Filter         string `json:"filter"`
}

type DeleteReq struct {
	CollectionNameReq string `json:"collectionName"`
	PartitionNameReq  string `json:"partitionName"`
	FilterReq         string `json:"filter"`
}

type DeleteResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (e *execution) delete(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := DeleteResp{}

	reqParams := DeleteReq{
		CollectionNameReq: inputStruct.CollectionName,
		PartitionNameReq:  inputStruct.PartitionName,
		FilterReq:         inputStruct.Filter,
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(deletePath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to delete point: %s", res.String())
	}

	if resp.Message != "" && resp.Code != 0 {
		return nil, fmt.Errorf("failed to delete data: %s", resp.Message)
	}

	outputStruct := DeleteOutput{
		Status: "Successfully deleted data",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
