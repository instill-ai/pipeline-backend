package weaviate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/schema"
	"github.com/weaviate/weaviate/entities/models"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type InsertInput struct {
	ID             string         `json:"id"`
	CollectionName string         `json:"collection-name"`
	Vector         []float32      `json:"vector"`
	Metadata       map[string]any `json:"metadata"`
}

type InsertOutput struct {
	Status string `json:"status"`
}

type VectorSearchInput struct {
	CollectionName string         `json:"collection-name"`
	Vector         []float32      `json:"vector"`
	Filter         map[string]any `json:"filter"`
	Limit          int            `json:"limit"`
	Fields         []string       `json:"fields"`
	Tenant         string         `json:"tenant"`
}

type Result struct {
	IDs      []string         `json:"ids"`
	Objects  []map[string]any `json:"objects"`
	Vectors  [][]float32      `json:"vectors"`
	Metadata []map[string]any `json:"metadata"`
}

type VectorSearchOutput struct {
	Status string `json:"status"`
	Result Result `json:"result"`
}

type UpdateInput struct {
	ID             string         `json:"id"`
	CollectionName string         `json:"collection-name"`
	Metadata       map[string]any `json:"update-metadata"`
	Vector         []float32      `json:"update-vector"`
}

type UpdateOutput struct {
	Status string `json:"status"`
}

type DeleteInput struct {
	ID             string         `json:"id"`
	CollectionName string         `json:"collection-name"`
	Filter         map[string]any `json:"filter"`
}

type DeleteOutput struct {
	Status string `json:"status"`
}

type BatchInsertInput struct {
	ArrayID        []string         `json:"array-id"`
	CollectionName string           `json:"collection-name"`
	ArrayMetadata  []map[string]any `json:"array-metadata"`
	ArrayVector    [][]float32      `json:"array-vector"`
}

type BatchInsertOutput struct {
	Status string `json:"status"`
}

type DeleteCollectionInput struct {
	CollectionName string `json:"collection-name"`
}

type DeleteCollectionOutput struct {
	Status string `json:"status"`
}

func jsonToWhereBuilder(jsonWhere *map[string]any) (*filters.WhereBuilder, error) {
	where := filters.Where()

	for key, value := range *jsonWhere {
		if key == "operands" {
			values := value.([]map[string]any)
			var operands []*filters.WhereBuilder
			for _, nestedJSONWhere := range values {
				operand, err := jsonToWhereBuilder(&nestedJSONWhere)
				operands = append(operands, operand)

				if err != nil {
					return nil, err
				}
			}
			where.WithOperands(operands)
		} else {
			switch key {
			case "path":
				path := value.(string)
				where.WithPath([]string{path})
			case "operator":
				operator := value.(string)
				switch operator {
				case "And":
					where.WithOperator(filters.And)
				case "Or":
					where.WithOperator(filters.Or)
				case "Equal":
					where.WithOperator(filters.Equal)
				case "NotEqual":
					where.WithOperator(filters.NotEqual)
				case "GreaterThan":
					where.WithOperator(filters.GreaterThan)
				case "GreaterThanEqual":
					where.WithOperator(filters.GreaterThanEqual)
				case "LessThan":
					where.WithOperator(filters.LessThan)
				case "LessThanEqual":
					where.WithOperator(filters.LessThanEqual)
				case "Like":
					where.WithOperator(filters.Like)
				case "WithinGeoRange":
					where.WithOperator(filters.WithinGeoRange)
				case "IsNull":
					where.WithOperator(filters.IsNull)
				case "ContainsAny":
					where.WithOperator(filters.ContainsAny)
				case "ContainsAll":
					where.WithOperator(filters.ContainsAll)
				default:
					return nil, fmt.Errorf("unknown operator: %s", operator)
				}
			case "valueInt":
				val := int64(value.(int))
				where.WithValueInt(val)
			case "valueBoolean":
				val := value.(bool)
				where.WithValueBoolean(val)
			case "valueString":
				val := value.(string)
				where.WithValueString(val)
			case "valueText":
				val := value.(string)
				where.WithValueText(val)
			case "valueNumber":
				val := value.(float64)
				where.WithValueNumber(val)
			case "valueDate":
				val := value.(time.Time)
				where.WithValueDate(val)
			default:
				return nil, fmt.Errorf("unknown key: %s", key)
			}
		}
	}

	return where, nil
}

func getAllFields(ctx context.Context, client *schema.ClassGetter, collectionName string) ([]string, error) {
	res, err := client.WithClassName(collectionName).Do(ctx)
	if err != nil {
		return nil, err
	}

	mapRes := make(map[string]any)
	byteRes, err := res.MarshalBinary()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(byteRes, &mapRes)
	if err != nil {
		return nil, err
	}

	properties, ok := mapRes["properties"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for properties")
	}

	var fields []string
	for _, property := range properties {
		if propertyMap, ok := property.(map[string]any); ok {
			fieldName, ok := propertyMap["name"].(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for field name")
			}
			fields = append(fields, fieldName)
		} else {
			return nil, fmt.Errorf("unexpected property format")
		}
	}

	return fields, nil
}

// vector is optional, nil will return all objects
// fields is optional, nil will return all objects
// limit is optional, 0 will return all objects
// tenant is optional, required for multi-tenancy
// filter is optional, nil will have no filter
func VectorSearch(ctx context.Context, client weaviate.Client, inputStruct VectorSearchInput) ([]map[string]any, error) {
	collectionName := inputStruct.CollectionName
	filter := inputStruct.Filter
	limit := inputStruct.Limit
	rawFields := inputStruct.Fields
	vector := inputStruct.Vector
	tenant := inputStruct.Tenant

	withBuilder := client.GraphQL().Get().
		WithClassName(collectionName)

	if vector != nil {
		nearVector := client.GraphQL().NearVectorArgBuilder().
			WithVector(vector)

		withBuilder.WithNearVector(nearVector)
	}
	if filter != nil {
		where, err := jsonToWhereBuilder(&filter)
		if err != nil {
			return nil, err
		}
		withBuilder.WithWhere(where)
	}
	if limit > 0 {
		withBuilder.WithLimit(limit)
	}
	fields := []graphql.Field{{Name: "_additional", Fields: []graphql.Field{
		{Name: "id"},
		{Name: "distance"},
		{Name: "vector"},
	}}}
	if len(rawFields) == 0 || rawFields == nil {
		allFields, err := getAllFields(ctx, client.Schema().ClassGetter(), collectionName)
		if err != nil {
			return nil, err
		}
		rawFields = allFields
	}
	for _, field := range rawFields {
		fields = append(fields, graphql.Field{Name: field})
	}
	withBuilder.WithFields(fields...)
	if tenant != "" {
		withBuilder.WithTenant(tenant)
	}

	res, err := withBuilder.Do(ctx)
	if err != nil || res.Errors != nil {
		return nil, err
	}

	mapRes := make(map[string]any)
	byteRes, err := res.MarshalBinary()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(byteRes, &mapRes)

	if err != nil {
		return nil, err
	}

	data, ok := mapRes["data"].(map[string]any)["Get"].(map[string]any)[collectionName].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected type for data")
	}

	var results []map[string]any
	for _, item := range data {
		if itemMap, ok := item.(map[string]any); ok {
			results = append(results, itemMap)
		} else {
			return nil, fmt.Errorf("unexpected item format")
		}
	}

	return results, nil
}

func (e *execution) insert(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct InsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	if e.mockClient == nil {
		creator := e.client.Data().Creator().
			WithClassName(inputStruct.CollectionName).
			WithVector(inputStruct.Vector).
			WithProperties(inputStruct.Metadata)

		if inputStruct.ID != "" {
			creator.WithID(inputStruct.ID)
		}

		_, err = creator.Do(ctx)
		if err != nil {
			return nil, err
		}
	}

	outputStruct := InsertOutput{
		Status: "Successfully inserted 1 object",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) vectorSearch(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct VectorSearchInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	var result Result
	var successful int
	if e.mockClient == nil {
		res, err := VectorSearch(ctx, *e.client, inputStruct)
		if err != nil {
			return nil, err
		}

		var ids []string
		var objects []map[string]any
		var vectors [][]float32
		var metadata []map[string]any

		for _, item := range res {
			vector, ok := item["_additional"].(map[string]any)["vector"].([]any)
			if !ok {
				return nil, fmt.Errorf("unexpected type for vector")
			}
			vectorFloat := make([]float32, len(vector))
			for i, v := range vector {
				vectorFloat[i] = float32(v.(float64))
			}

			id, ok := item["_additional"].(map[string]any)["id"].(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for id")
			}

			vectors = append(vectors, vectorFloat)
			tempProperties := make(map[string]any)

			for key, value := range item {
				if key != "_additional" {
					tempProperties[key] = value
				}
			}
			metadata = append(metadata, tempProperties)

			objects = append(objects, item)
			ids = append(ids, id)
		}

		result = Result{
			IDs:      ids,
			Objects:  objects,
			Vectors:  vectors,
			Metadata: metadata,
		}
		successful = len(objects)
	} else {
		result = e.mockClient.VectorSearch
		successful = e.mockClient.Successful
	}

	outputStruct := VectorSearchOutput{
		Status: fmt.Sprintf("Successfully found %d objects", successful),
		Result: result,
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) update(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct UpdateInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	if inputStruct.Vector == nil && inputStruct.Metadata == nil {
		return nil, fmt.Errorf("either vector or metadata must be provided")
	}

	if e.mockClient == nil {
		updater := e.client.Data().Updater().
			WithMerge().
			WithClassName(inputStruct.CollectionName).
			WithID(inputStruct.ID).
			WithProperties(inputStruct.Metadata).
			WithVector(inputStruct.Vector)

		err = updater.Do(ctx)
		if err != nil {
			return nil, err
		}
	}

	outputStruct := UpdateOutput{
		Status: "Successfully updated 1 object",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) delete(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	filter := inputStruct.Filter
	collectionName := inputStruct.CollectionName
	id := inputStruct.ID

	if id == "" && filter == nil {
		return nil, fmt.Errorf("either id or filter must be provided")
	}

	where, err := jsonToWhereBuilder(&filter)
	if err != nil {
		return nil, err
	}

	var res *models.BatchDeleteResponse
	var successful int
	if e.mockClient == nil {
		if inputStruct.ID != "" {
			err = e.client.Data().Deleter().WithClassName(collectionName).WithID(id).Do(ctx)
			if err != nil {
				return nil, err
			}

			successful = int(1)
		} else {
			res, err = e.client.Batch().ObjectsBatchDeleter().
				WithClassName(collectionName).
				WithWhere(where).
				Do(ctx)
			if err != nil {
				return nil, err
			}

			successful = int(res.Results.Successful)
		}
	} else {
		successful = e.mockClient.Successful
	}

	outputStruct := DeleteOutput{
		Status: fmt.Sprintf("Successfully deleted %d objects", successful),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) batchInsert(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct BatchInsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	collectionName := inputStruct.CollectionName
	arrayMetadata := inputStruct.ArrayMetadata
	arrayVector := inputStruct.ArrayVector
	arrayID := inputStruct.ArrayID

	var successful int
	if e.mockClient == nil {
		batcher := e.client.Batch().ObjectsBatcher()
		for i, properties := range arrayMetadata {
			modelsObject := &models.Object{
				Class:      collectionName,
				Properties: properties,
				Vector:     arrayVector[i],
			}
			if len(arrayID) == len(arrayMetadata) {
				modelsObject.ID = strfmt.UUID(arrayID[i])
			} else if arrayID != nil {
				return nil, fmt.Errorf("array-id and array-metadata must have the same length")
			}

			batcher = batcher.WithObjects(modelsObject)
		}
		_, err = batcher.Do(ctx)

		if err != nil {
			return nil, err
		}

		successful = len(arrayMetadata)
	} else {
		successful = e.mockClient.Successful
	}

	outputStruct := BatchInsertOutput{
		Status: fmt.Sprintf("Successfully batch inserted %d objects", successful),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) deleteCollection(ctx context.Context, in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteCollectionInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}

	collectionName := inputStruct.CollectionName

	if e.mockClient == nil {
		err = e.client.Schema().ClassDeleter().
			WithClassName(collectionName).
			Do(ctx)

		if err != nil {
			return nil, err
		}
	}

	outputStruct := DeleteCollectionOutput{
		Status: "Successfully deleted 1 collection",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}

	return output, nil
}
