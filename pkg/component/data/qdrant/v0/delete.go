package qdrant

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	deletePath = "/collections/%s/points/delete?wait=true&ordering=%s"
)

type DeleteInput struct {
	ID             string         `json:"id"`
	CollectionName string         `json:"collection-name"`
	Filter         map[string]any `json:"filter"`
	Ordering       string         `json:"ordering"`
}

type DeleteOutput struct {
	Status string `json:"status"`
}

type DeleteReq struct {
	Points []string       `json:"points"`
	Filter map[string]any `json:"filter"`
}

type DeleteResp struct {
	Time   float64           `json:"time"`
	Status string            `json:"status"`
	Result BatchUpsertResult `json:"result"`
}

func (e *execution) delete(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	if inputStruct.ID == "" && inputStruct.Filter == nil {
		return nil, fmt.Errorf("either ID or Filter is required")
	}

	resp := DeleteResp{}

	reqParams := DeleteReq{}
	if inputStruct.ID != "" {
		reqParams.Points = []string{inputStruct.ID}
	}
	if inputStruct.Filter != nil {
		reqParams.Filter = inputStruct.Filter
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(fmt.Sprintf(deletePath, inputStruct.CollectionName, inputStruct.Ordering))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to delete points: %s", res.String())
	}

	outputStruct := DeleteOutput{
		Status: "Successfully deleted points",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
