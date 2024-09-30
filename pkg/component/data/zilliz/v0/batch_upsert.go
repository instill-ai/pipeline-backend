package zilliz

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type BatchUpsertOutput struct {
	Status string `json:"status"`
}

type BatchUpsertInput struct {
	CollectionName string           `json:"collection-name"`
	PartitionName  string           `json:"partition-name"`
	ArrayData      []map[string]any `json:"array-data"`
}

func (e *execution) batchUpsert(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct BatchUpsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := UpsertResp{}

	reqParams := UpsertReq{
		CollectionNameReq: inputStruct.CollectionName,
		PartitionNameReq:  inputStruct.PartitionName,
		DataReq:           inputStruct.ArrayData,
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(upsertPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to upsert data: %s", res.String())
	}

	if resp.Message != "" && resp.Code != 0 {
		return nil, fmt.Errorf("failed to upsert data: %s", resp.Message)
	}

	outputStruct := UpsertOutput{
		Status: fmt.Sprintf("Successfully batch upserted %d data", resp.Data.UpsertCount),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
