package zilliz

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	upsertPath = "/v2/vectordb/entities/upsert"
)

type UpsertOutput struct {
	Status string `json:"status"`
}

type UpsertInput struct {
	CollectionName string         `json:"collection-name"`
	PartitionName  string         `json:"partition-name"`
	Data           map[string]any `json:"data"`
}

type UpsertReq struct {
	CollectionNameReq string           `json:"collectionName"`
	PartitionNameReq  string           `json:"partitionName,omitempty"`
	DataReq           []map[string]any `json:"data"`
}

type UpsertResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    Data   `json:"data"`
}

type Data struct {
	UpsertCount int      `json:"upsertCount"`
	UpsertIDs   []string `json:"upsertIds"`
}

func (e *execution) upsert(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct UpsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := UpsertResp{}

	reqParams := UpsertReq{
		CollectionNameReq: inputStruct.CollectionName,
		DataReq:           []map[string]any{inputStruct.Data},
	}
	if inputStruct.PartitionName != "" {
		reqParams.PartitionNameReq = inputStruct.PartitionName
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
		Status: "Successfully upserted 1 data",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
