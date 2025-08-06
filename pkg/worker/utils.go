package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	rangeStart             = "start"
	rangeStop              = "stop"
	rangeStep              = "step"
	defaultRangeIdentifier = "i"
)

// setIteratorIndex converts the iterator index identifier into a numeric
// index. For example, it converts `${variable.array[i]}` into
// `${variable.array[0]}`.
func setIteratorIndex(v format.Value, identifier string, index int) format.Value {
	if identifier == "" {
		identifier = defaultRangeIdentifier
	}
	switch v := v.(type) {
	case format.ReferenceString:
		s := v.String()
		val := ""
		for {
			startIdx := strings.Index(s, "${")
			if startIdx == -1 {
				val += s
				break
			}
			val += s[:startIdx]
			s = s[startIdx:]
			endIdx := strings.Index(s, "}")
			if endIdx == -1 {
				val += s
				break
			}

			ref := strings.TrimSpace(s[2:endIdx])
			ref = strings.ReplaceAll(ref, fmt.Sprintf("[%s]", identifier), fmt.Sprintf("[%d]", index))

			val += fmt.Sprintf("${%s}", ref)
			s = s[endIdx+1:]
		}
		return data.NewString(val)
	case data.Array:
		m := make(data.Array, len(v))
		for idx, item := range v {
			m[idx] = setIteratorIndex(item, identifier, index)
		}
		return m
	case data.Map:
		m := data.Map{}
		for k, v := range v {
			m[k] = setIteratorIndex(v, identifier, index)
		}
		return m
	default:
		return v
	}
}

// isUnstructuredData checks if a string contains unstructured data (data URI format)
func isUnstructuredData(data string) bool {
	return strings.HasPrefix(data, "data:") && strings.Contains(data, ";base64,")
}

// processStructUnstructuredData processes unstructured data in a struct
func processStructUnstructuredData(ctx context.Context, dataStruct *structpb.Struct, uploadFn uploadFunc, param *ComponentActivityParam) (*structpb.Struct, error) {
	for key, value := range dataStruct.GetFields() {
		updatedValue, err := processValueUnstructuredData(ctx, value, uploadFn, param)
		if err == nil {
			dataStruct.GetFields()[key] = updatedValue
		}
	}
	return dataStruct, nil
}

// uploadFunc is a function type for uploading unstructured data
type uploadFunc func(ctx context.Context, data string, param *ComponentActivityParam) (string, error)

// processValueUnstructuredData processes unstructured data in a value
func processValueUnstructuredData(ctx context.Context, value *structpb.Value, uploadFn uploadFunc, param *ComponentActivityParam) (*structpb.Value, error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_StringValue:
		if isUnstructuredData(v.StringValue) {
			downloadURL, err := uploadFn(ctx, v.StringValue, param)
			if err != nil {
				return nil, err
			}
			return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: downloadURL}}, nil
		}
	case *structpb.Value_ListValue:
		listValue := v.ListValue
		updatedListValue, err := processListUnstructuredData(ctx, listValue, uploadFn, param)
		if err == nil {
			return &structpb.Value{Kind: &structpb.Value_ListValue{ListValue: updatedListValue}}, nil
		}
	case *structpb.Value_StructValue:
		for _, item := range v.StructValue.GetFields() {
			structData := item.GetStructValue()
			updatedStructData, err := processStructUnstructuredData(ctx, structData, uploadFn, param)
			// Note: we don't want to fail the whole process if one of the data structs fails to upload.
			if err == nil {
				return &structpb.Value{Kind: &structpb.Value_StructValue{StructValue: updatedStructData}}, nil
			}
		}
	}
	return value, nil
}

// processListUnstructuredData processes unstructured data in a list
func processListUnstructuredData(ctx context.Context, list *structpb.ListValue, uploadFn uploadFunc, param *ComponentActivityParam) (*structpb.ListValue, error) {
	for i, item := range list.Values {
		updatedItem, err := processValueUnstructuredData(ctx, item, uploadFn, param)
		if err == nil {
			list.Values[i] = updatedItem
		}
	}
	return list, nil
}
