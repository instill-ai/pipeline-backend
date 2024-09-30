package qdrant

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type UpsertOutput struct {
	Status string `json:"status"`
}

type UpsertInput struct {
	CollectionName string         `json:"collection-name"`
	ID             string         `json:"id"`
	Metadata       map[string]any `json:"metadata"`
	Vector         []float64      `json:"vector"`
	Ordering       string         `json:"ordering"`
}

func (e *execution) upsert(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct UpsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := BatchUpsertResp{}

	reqParams := BatchUpsertReq{
		Batch: Batch{
			IDs:      []string{inputStruct.ID},
			Vectors:  [][]float64{inputStruct.Vector},
			Payloads: []map[string]any{inputStruct.Metadata},
		},
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Put(fmt.Sprintf(batchUpsertPath, inputStruct.CollectionName, inputStruct.Ordering))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to upsert point: %s", res.String())
	}

	outputStruct := UpsertOutput{
		Status: "Successfully upserted 1 point",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
