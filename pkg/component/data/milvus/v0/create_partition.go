package milvus

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	createPartitionPath = "/v2/vectordb/partitions/create"
)

type CreatePartitionOutput struct {
	Status string `json:"status"`
}

type CreatePartitionInput struct {
	CollectionName string `json:"collection-name"`
	PartitionName  string `json:"partition-name"`
}

type CreatePartitionReq struct {
	CollectionNameReq string `json:"collectionName"`
	PartitionNameReq  string `json:"partitionName"`
}

type CreatePartitionResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (e *execution) createPartition(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreatePartitionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := CreatePartitionResp{}

	reqParams := CreatePartitionReq{
		CollectionNameReq: inputStruct.CollectionName,
		PartitionNameReq:  inputStruct.PartitionName,
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(createPartitionPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create partition: %s", res.String())
	}

	outputStruct := CreatePartitionOutput{
		Status: "Successfully created 1 partition",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
