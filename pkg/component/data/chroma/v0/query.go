package chroma

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	queryPath = "/api/v1/collections/%s/query"
)

type QueryOutput struct {
	Status string `json:"status"`
	Result Result `json:"result"`
}

type Result struct {
	Ids      []string         `json:"ids"`
	Items    []map[string]any `json:"items"`
	Vectors  [][]float64      `json:"vectors"`
	Metadata []map[string]any `json:"metadata"`
}

type QueryInput struct {
	CollectionName string         `json:"collection-name"`
	Vector         []float64      `json:"vector"`
	Filter         map[string]any `json:"filter"`
	FilterDocument map[string]any `json:"filter-document"`
	NResults       int            `json:"n-results"`
	Fields         []string       `json:"fields"`
}

type QueryReq struct {
	QueryEmbeddings [][]float64    `json:"query_embeddings"`
	Where           map[string]any `json:"where"`
	WhereDocument   map[string]any `json:"where_document"`
	NResults        int            `json:"n_results"`
	Include         []string       `json:"include"`
}

type QueryResp struct {
	IDs        [][]string         `json:"ids"`
	Distances  [][]float64        `json:"distances"`
	Metadatas  [][]map[string]any `json:"metadatas"`
	Embeddings [][][]float64      `json:"embeddings"`
	Documents  [][]string         `json:"documents"`
	Uris       [][]string         `json:"uris"`

	Detail []map[string]any `json:"detail"`
}

func (e *execution) query(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct QueryInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := QueryResp{}

	reqParams := QueryReq{
		QueryEmbeddings: [][]float64{inputStruct.Vector},
		NResults:        inputStruct.NResults,
		Include:         []string{"embeddings", "metadatas", "distances", "documents"},
	}
	if inputStruct.Filter != nil {
		reqParams.Where = inputStruct.Filter
	}
	if inputStruct.FilterDocument != nil {
		reqParams.WhereDocument = inputStruct.FilterDocument
	}

	var collID string

	collID, err = getCollectionID(inputStruct.CollectionName, e.client)
	if err != nil {
		return nil, err
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(fmt.Sprintf(queryPath, collID))

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to query item: %s", res.String())
	}

	if resp.Detail != nil {
		return nil, fmt.Errorf("failed to query item: %s", resp.Detail[0]["msg"])
	}

	if inputStruct.Fields != nil {
		var notFields []string
		for k := range resp.Metadatas[0][0] {
			var found bool
			for _, field := range inputStruct.Fields {
				if k == field {
					found = true
					break
				}
			}
			if !found {
				notFields = append(notFields, k)
			}
		}

		for _, metadata := range resp.Metadatas[0] {
			for _, notField := range notFields {
				delete(metadata, notField)
			}

		}
	}

	ids := resp.IDs[0]
	metadatas := resp.Metadatas[0]
	vectors := resp.Embeddings[0]
	var uris []string
	if len(resp.Uris) > 0 {
		uris = resp.Uris[0]
	}
	var documents []string
	if len(resp.Documents) > 0 {
		documents = resp.Documents[0]
	}
	var items []map[string]any

	for i, metadata := range metadatas {
		item := make(map[string]any)
		for k, v := range metadata {
			if k != "id" {
				item[k] = v
			}
		}
		item["distance"] = resp.Distances[0][i]
		item["id"] = ids[i]
		item["vector"] = vectors[i]
		if len(uris) > 0 {
			item["uri"] = uris[i]
		}
		if len(documents) > 0 {
			item["document"] = documents[i]
		}
		items = append(items, item)
	}

	outputStruct := QueryOutput{
		Status: fmt.Sprintf("Successfully queryed %d items", len(resp.IDs[0])),
		Result: Result{
			Ids:      ids,
			Items:    items,
			Vectors:  vectors,
			Metadata: metadatas,
		},
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
