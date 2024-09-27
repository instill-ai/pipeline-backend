package chroma

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	upsertPath = "/api/v1/collections/%s/upsert"
)

type UpsertOutput struct {
	Status string `json:"status"`
}

type UpsertInput struct {
	CollectionName string         `json:"collection-name"`
	ID             string         `json:"id"`
	Vector         []float64      `json:"vector"`
	Metadata       map[string]any `json:"metadata"`
	Document       string         `json:"document"`
	URI            string         `json:"uri"`
}

type UpsertReq struct {
	Embeddings [][]float64      `json:"embeddings"`
	Metadatas  []map[string]any `json:"metadatas"`
	Documents  []string         `json:"documents"`
	IDs        []string         `json:"ids"`
	Uris       []string         `json:"uris"`
}

type UpsertResp struct {
	Error   string `json:"error"`
	Message string `json:"message"`
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
		Embeddings: [][]float64{inputStruct.Vector},
		Metadatas:  []map[string]any{inputStruct.Metadata},
		IDs:        []string{inputStruct.ID},
	}
	if inputStruct.Document != "" {
		reqParams.Documents = []string{inputStruct.Document}
	}
	if inputStruct.URI != "" {
		reqParams.Uris = []string{inputStruct.URI}
	}

	var collID string

	collID, err = getCollectionID(inputStruct.CollectionName, e.client)
	if err != nil {
		return nil, err
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(fmt.Sprintf(upsertPath, collID))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to upsert item: %s", res.String())
	}

	if resp.Error != "" && resp.Message != "" {
		return nil, fmt.Errorf("failed to upsert item: %s", resp.Message)
	}

	outputStruct := UpsertOutput{
		Status: "Successfully upserted 1 item",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
