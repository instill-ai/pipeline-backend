package zilliz

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	dropPartitionPath = "/v2/vectordb/partitions/drop"
)

type DropPartitionOutput struct {
	Status string `json:"status"`
}

type DropPartitionInput struct {
	CollectionName string `json:"collection-name"`
	PartitionName  string `json:"partition-name"`
}

type DropPartitionReq struct {
	CollectionNameReq string `json:"collectionName"`
	PartitionNameReq  string `json:"partitionName"`
}

type DropPartitionResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (e *execution) dropPartition(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropPartitionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := DropPartitionResp{}

	reqParams := DropPartitionReq{
		CollectionNameReq: inputStruct.CollectionName,
		PartitionNameReq:  inputStruct.PartitionName,
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(dropPartitionPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to drop partition: %s", res.String())
	}

	outputStruct := DropPartitionOutput{
		Status: "Successfully dropped 1 partition",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
