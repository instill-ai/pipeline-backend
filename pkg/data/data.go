package data

import (
	"fmt"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"google.golang.org/protobuf/types/known/structpb"
)

type InstillFormat string

const (
	FormatNull      InstillFormat = "null"
	FormatBoolean   InstillFormat = "boolean"
	FormatString    InstillFormat = "string"
	FormatNumber    InstillFormat = "number"
	FormatByteArray InstillFormat = "byte-array"
	FormatFile      InstillFormat = "file"
	FormatDocument  InstillFormat = "document"
	FormatImage     InstillFormat = "image"
	FormatVideo     InstillFormat = "video"
	FormatAudio     InstillFormat = "audio"
)

func NewValue(in any) (val format.Value, err error) {

	switch in := in.(type) {
	case nil:
		return NewNull(), nil
	case time.Time:
		// Now, we don't provide time format in pipeline recipe.
		// However, in YAML, it read the time format and convert it to time.Time.
		// So, we need to convert it back to string.
		// In the future, we may need to provide time format in pipeline recipe.
		return NewString(in.Format(time.DateOnly)), nil
	case bool:
		return NewBoolean(in), nil
	case float64:
		return NewNumberFromFloat(in), nil
	case int:
		return NewNumberFromInteger(in), nil
	case string:
		return NewString(in), nil
	case []any:
		arr := make(Array, len(in))
		for i, item := range in {
			arr[i], err = NewValue(item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	case map[string]any:
		mp := make(Map)
		for k, v := range in {
			mp[k], err = NewValue(v)
			if err != nil {
				return nil, err
			}
		}
		return mp, nil
	}

	return nil, fmt.Errorf("NewValue error")
}

func NewJSONValue(in any) (val format.Value, err error) {

	switch in := in.(type) {
	case bool:
		return NewBoolean(in), nil
	case float64:
		return NewNumberFromFloat(in), nil
	case int:
		return NewNumberFromInteger(in), nil
	case string:
		return NewString(in), nil
	case []any:
		arr := make(Array, len(in))
		for i, item := range in {
			arr[i], err = NewJSONValue(item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	case map[string]any:
		mp := make(Map)
		for k, v := range in {
			mp[k], err = NewJSONValue(v)
			if err != nil {
				return nil, err
			}
		}
		return mp, nil
	case nil:
		return NewNull(), nil
	}

	return nil, fmt.Errorf("NewJSONValue error")
}

func NewValueFromStruct(in *structpb.Value) (val format.Value, err error) {

	if in == nil {
		return NewNull(), nil
	}

	switch in := in.Kind.(type) {
	case *structpb.Value_NullValue:
		return NewNull(), nil
	case *structpb.Value_BoolValue:
		return NewBoolean(in.BoolValue), nil
	case *structpb.Value_NumberValue:
		return NewNumberFromFloat(in.NumberValue), nil
	case *structpb.Value_StringValue:
		return NewString(in.StringValue), nil
	case *structpb.Value_ListValue:
		arr := make(Array, len(in.ListValue.Values))
		for i, item := range in.ListValue.GetValues() {
			arr[i], err = NewValueFromStruct(item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	case *structpb.Value_StructValue:
		mp := make(Map)
		for k, v := range in.StructValue.GetFields() {

			mp[k], err = NewValueFromStruct(v)
			if err != nil {
				return nil, err
			}
		}
		return mp, nil
	}

	return nil, fmt.Errorf("NewValueFromStruct error")
}
