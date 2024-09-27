package qdrant

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	batchUpsertPath = "/collections/%s/points?wait=true&ordering=%s"
)

type BatchUpsertInput struct {
	CollectionName string           `json:"collection-name"`
	ArrayID        []string         `json:"array-id"`
	ArrayMetadata  []map[string]any `json:"array-metadata"`
	ArrayVector    [][]float64      `json:"array-vector"`
	Ordering       string           `json:"ordering"`
}

type BatchUpsertOutput struct {
	Status string `json:"status"`
}

type BatchUpsertReq struct {
	Batch Batch `json:"batch"`
}

type Batch struct {
	IDs      []string         `json:"ids"`
	Vectors  [][]float64      `json:"vectors"`
	Payloads []map[string]any `json:"payloads"`
}

type BatchUpsertResult struct {
	Status      string `json:"status"`
	OperationID int    `json:"operation_id"`
}

type BatchUpsertResp struct {
	Time   float64           `json:"time"`
	Status string            `json:"status"`
	Result BatchUpsertResult `json:"result"`
}

func (e *execution) batchUpsert(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct BatchUpsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := BatchUpsertResp{}

	reqParams := BatchUpsertReq{
		Batch: Batch{
			IDs:      inputStruct.ArrayID,
			Vectors:  inputStruct.ArrayVector,
			Payloads: inputStruct.ArrayMetadata,
		},
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Put(fmt.Sprintf(batchUpsertPath, inputStruct.CollectionName, inputStruct.Ordering))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to batch upsert points: %s", res.String())
	}

	outputStruct := BatchUpsertOutput{
		Status: fmt.Sprintf("Successfully batch upserted %d points", len(inputStruct.ArrayVector)),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
