package zilliz

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	createCollectionPath = "/v2/vectordb/collections/create"
)

type CreateCollectionOutput struct {
	Status string `json:"status"`
}

type CreateCollectionInput struct {
	CollectionName   string         `json:"collection-name"`
	Dimension        int            `json:"dimension"`
	MetricType       string         `json:"metric-type"`
	IDType           string         `json:"id-type"`
	AutoID           bool           `json:"auto-id"`
	PrimaryFieldName string         `json:"primary-field-name"`
	VectorFieldName  string         `json:"vector-field-name"`
	Schema           map[string]any `json:"schema"`
	IndexParams      map[string]any `json:"index-params"`
	Params           map[string]any `json:"params"`
}

type CreateCollectionReq struct {
	CollectionNameReq   string         `json:"collectionName"`
	DimensionReq        int            `json:"dimension"`
	MetricTypeReq       string         `json:"metricType"`
	IDTypeReq           string         `json:"idType"`
	AutoIDReq           bool           `json:"autoID"`
	PrimaryFieldNameReq string         `json:"primaryFieldName"`
	VectorFieldNameReq  string         `json:"vectorFieldName"`
	SchemaReq           map[string]any `json:"schema"`
	IndexParamsReq      map[string]any `json:"indexParams"`
	ParamsReq           map[string]any `json:"params"`
}

type CreateCollectionResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func (e *execution) createCollection(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	resp := CreateCollectionResp{}

	reqParams := CreateCollectionReq{
		CollectionNameReq: inputStruct.CollectionName,
		MetricTypeReq:     inputStruct.MetricType,
		AutoIDReq:         inputStruct.AutoID,
		DimensionReq:      inputStruct.Dimension,
	}
	if inputStruct.IDType != "" {
		reqParams.IDTypeReq = inputStruct.IDType
	}
	if inputStruct.PrimaryFieldName != "" {
		reqParams.PrimaryFieldNameReq = inputStruct.PrimaryFieldName
	}
	if inputStruct.VectorFieldName != "" {
		reqParams.VectorFieldNameReq = inputStruct.VectorFieldName
	}
	if inputStruct.Schema != nil {
		reqParams.SchemaReq = inputStruct.Schema
	}
	if inputStruct.IndexParams != nil {
		reqParams.IndexParamsReq = inputStruct.IndexParams
	}
	if inputStruct.Params != nil {
		reqParams.ParamsReq = inputStruct.Params
	}

	req := e.client.R().SetBody(reqParams).SetResult(&resp)

	res, err := req.Post(createCollectionPath)

	if err != nil {
		return nil, err
	}

	if res.StatusCode() != 200 {
		return nil, fmt.Errorf("failed to create collection: %s", res.String())
	}

	if resp.Message != "" && resp.Code != 0 {
		return nil, fmt.Errorf("failed to create collection: %s", resp.Message)
	}

	outputStruct := CreateCollectionOutput{
		Status: "Successfully created 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
