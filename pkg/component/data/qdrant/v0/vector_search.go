package qdrant

import (
	"fmt"
	"strconv"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	vectorSearchPath = "/collections/%s/points/search"
)

type VectorSearchInput struct {
	CollectionName string         `json:"collection-name"`
	Vector         []float64      `json:"vector"`
	Filter         map[string]any `json:"filter"`
	Limit          int            `json:"limit"`
	Payloads       []string       `json:"payloads"`
	Params         map[string]any `json:"params"`
	MinScore       float64        `json:"min-score"`
}

type VectorSearchOutput struct {
	Status string `json:"status"`
	Result Result `json:"result"`
}

type Result struct {
	Ids      []string         `json:"ids"`
	Points   []map[string]any `json:"points"`
	Vectors  [][]float64      `json:"vectors"`
	Metadata []map[string]any `json:"metadata"`
}

type VectorSearchReq struct {
	Vector     []float64      `json:"vector"`
	Limit      int            `json:"limit"`
	Payloads   any            `json:"with_payload"`
	Filter     map[string]any `json:"filter"`
	Params     map[string]any `json:"params"`
	WithVector bool           `json:"with_vector"`
	MinScore   float64        `json:"score_threshold"`
}

type VectorSearchResp struct {
	Time   float64              `json:"time"`
	Status string               `json:"status"`
	Result []VectorSearchResult `json:"result"`
}

type VectorSearchResult struct {
	ID         any            `json:"id"`
	Version    int            `json:"version"`
	Score      float64        `json:"score"`
	Payload    map[string]any `json:"payload"`
	Vector     []float64      `json:"vector"`
	ShardKey   string         `json:"shard_key"`
	OrderValue float64        `json:"order_value"`
}

func (e *execution) vectorSearch(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct VectorSearchInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := VectorSearchResp{}

	reqParams := VectorSearchReq{
		Vector:     inputStruct.Vector,
		Limit:      inputStruct.Limit,
		WithVector: true,
		Filter:     map[string]any{},
		Params:     map[string]any{},
		Payloads:   true,
	}

	if inputStruct.Payloads != nil {
		reqParams.Payloads = inputStruct.Payloads
	}
	if inputStruct.Filter != nil {
		reqParams.Filter = inputStruct.Filter
	}
	if inputStruct.Params != nil {
		reqParams.Params = inputStruct.Params
	}
	if inputStruct.MinScore != 0 {
		reqParams.MinScore = inputStruct.MinScore
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(fmt.Sprintf(vectorSearchPath, inputStruct.CollectionName))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to vector search points: %s", res.String())
	}

	var ids []string
	var points []map[string]any
	var vectors [][]float64
	var metadata []map[string]any

	for _, result := range resp.Result {
		point := make(map[string]any)
		for k, v := range result.Payload {
			point[k] = v
		}

		switch v := result.ID.(type) {
		case string:
			point["id"] = v
			ids = append(ids, v)
		case float64:
			point["id"] = strconv.Itoa(int(v))
			ids = append(ids, strconv.Itoa(int(v)))
		}

		point["score"] = result.Score
		if result.Version != 0 {
			point["version"] = result.Version
		}
		if result.ShardKey != "" {
			point["shard_key"] = result.ShardKey
		}
		if result.OrderValue != 0 {
			point["order_value"] = result.OrderValue
		}
		point["vector"] = result.Vector

		points = append(points, point)
		vectors = append(vectors, result.Vector)
		metadata = append(metadata, result.Payload)
	}

	outputStruct := VectorSearchOutput{
		Status: fmt.Sprintf("Successfully vector searched %d points", len(resp.Result)),
		Result: Result{
			Ids:      ids,
			Points:   points,
			Vectors:  vectors,
			Metadata: metadata,
		},
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
