package milvus

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	searchPath             = "/v2/vectordb/entities/search"
	describeCollectionPath = "/v2/vectordb/collections/describe"
	loadCollectionPath     = "/v2/vectordb/collections/load"
)

type SearchOutput struct {
	Status string `json:"status"`
	Result Result `json:"result"`
}

type Result struct {
	Ids      []string         `json:"ids"`
	Data     []map[string]any `json:"data"`
	Vectors  [][]float32      `json:"vectors"`
	Metadata []map[string]any `json:"metadata"`
}

type SearchInput struct {
	CollectionName string         `json:"collection-name"`
	PartitionName  string         `json:"partition-name"`
	Vector         []float32      `json:"vector"`
	Filter         string         `json:"filter"`
	Limit          int            `json:"limit"`
	VectorField    string         `json:"vector-field"`
	Offset         int            `json:"offset"`
	GroupingField  string         `json:"grouping-field"`
	Fields         []string       `json:"fields"`
	SearchParams   map[string]any `json:"search-params"`
}

type SearchReq struct {
	CollectionName string         `json:"collectionName"`
	PartitionName  string         `json:"partitionName"`
	Data           [][]float32    `json:"data"`
	Filter         string         `json:"filter"`
	Limit          int            `json:"limit"`
	AnnsField      string         `json:"annsField"`
	Offset         int            `json:"offset"`
	GroupingField  string         `json:"groupingField"`
	OutputFields   []string       `json:"outputFields"`
	SearchParams   map[string]any `json:"searchParams"`
}

type SearchResp struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    []map[string]any `json:"data"`
}

type DescribeCollection struct {
	CollectionName string `json:"collection-name"`
}

type DescribeCollectionReq struct {
	CollectionNameReq string `json:"collectionName"`
}

type DescribeCollectionResp struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    DataDescribe `json:"data"`
}

type LoadCollectionReq struct {
	CollectionNameReq string `json:"collectionName"`
}

type LoadCollectionResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type DataDescribe struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	Name       string `json:"name"`
	PrimaryKey bool   `json:"primaryKey"`
}

func (e *execution) search(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct SearchInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	respDescribe := DescribeCollectionResp{}

	reqParamsDescribe := DescribeCollectionReq{
		CollectionNameReq: inputStruct.CollectionName,
	}

	respLoadCollection := LoadCollectionResp{}

	reqLoadCollection := LoadCollectionReq{
		CollectionNameReq: inputStruct.CollectionName,
	}

	if e.Setup.Fields["username"].GetStringValue() == "mock-root" {
		respDescribe.Data.Fields = []Field{
			{
				Name:       "id",
				PrimaryKey: true,
			},
			{
				Name:       "name",
				PrimaryKey: false,
			},
			{
				Name:       "vector",
				PrimaryKey: false,
			},
		}
	} else {
		reqLoadCollection := e.client.R().SetBody(reqLoadCollection).SetResult(&respLoadCollection)

		resLoadCollection, err := reqLoadCollection.Post(loadCollectionPath)

		if err != nil {
			return nil, err
		}

		if resLoadCollection.StatusCode() != 200 {
			return nil, fmt.Errorf("failed to load collection: %s", resLoadCollection.String())
		}

		if respLoadCollection.Message != "" && respLoadCollection.Code != 200 {
			return nil, fmt.Errorf("failed to load collection: %s", respLoadCollection.Message)
		}

		reqDescribe := e.client.R().SetBody(reqParamsDescribe).SetResult(&respDescribe)

		resDescribe, err := reqDescribe.Post(describeCollectionPath)

		if err != nil {
			return nil, err
		}

		if resDescribe.StatusCode() != 200 {
			return nil, fmt.Errorf("failed to describe collection: %s", resDescribe.String())
		}

		if respDescribe.Message != "" && respDescribe.Code != 200 {
			return nil, fmt.Errorf("failed to describe collection: %s", respDescribe.Message)
		}
	}

	if respDescribe.Message != "" && respDescribe.Code != 200 {
		return nil, fmt.Errorf("failed to describe collection: %s", respDescribe.Message)
	}

	var primaryKeyField string
	for _, field := range respDescribe.Data.Fields {
		if field.PrimaryKey {
			primaryKeyField = field.Name
			break
		}
	}

	var fields []string
	if inputStruct.Fields == nil {
		for _, field := range respDescribe.Data.Fields {
			fields = append(fields, field.Name)
		}
	} else {
		fields = append(fields, primaryKeyField)
		fields = append(fields, inputStruct.VectorField)
		fields = append(fields, inputStruct.Fields...)
	}

	resp := SearchResp{}

	reqParams := SearchReq{
		CollectionName: inputStruct.CollectionName,
		Data:           [][]float32{inputStruct.Vector},
		Limit:          inputStruct.Limit,
		AnnsField:      inputStruct.VectorField,
		OutputFields:   fields,
	}
	if inputStruct.PartitionName != "" {
		reqParams.PartitionName = inputStruct.PartitionName
	}
	if inputStruct.Filter != "" {
		reqParams.Filter = inputStruct.Filter
	}
	if inputStruct.Offset != 0 {
		reqParams.Offset = inputStruct.Offset
	}
	if inputStruct.GroupingField != "" {
		reqParams.GroupingField = inputStruct.GroupingField
	}
	if inputStruct.SearchParams != nil {
		reqParams.SearchParams = inputStruct.SearchParams
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(searchPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to Search point: %s", res.String())
	}

	if resp.Message != "" && resp.Code != 200 {
		return nil, fmt.Errorf("failed to upsert data: %s", resp.Message)
	}

	var ids []string
	var metadata []map[string]any
	var vectors [][]float32
	data := resp.Data

	for _, d := range data {
		var vectorFloat32 []float32
		for _, v := range d[inputStruct.VectorField].([]any) {
			vectorFloat32 = append(vectorFloat32, float32(v.(float64)))
		}

		ids = append(ids, fmt.Sprintf("%v", d[primaryKeyField]))
		vectors = append(vectors, vectorFloat32)

		metadatum := make(map[string]any)
		if inputStruct.Fields != nil {
			for _, field := range inputStruct.Fields {
				if _, ok := metadatum[field]; ok {
					metadatum[field] = d[field]
				}
			}
		} else {
			for _, field := range fields {
				if field == "distance" {
					continue
				}
				metadatum[field] = d[field]
			}
		}

		metadata = append(metadata, metadatum)
	}

	outputStruct := SearchOutput{
		Status: fmt.Sprintf("Successfully searched %d data", len(resp.Data)),
		Result: Result{
			Ids:      ids,
			Data:     data,
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
