package data

import (
	"encoding/gob"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

type Data map[string]Value
type Value interface {
	isValue()
	ToStructValue() (v *structpb.Value, err error)
}

func NewValue(in any) (val Value, err error) {

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
		arr := NewArray(make([]Value, len(in)))
		for i, item := range in {
			arr.Values[i], err = NewValue(item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	case map[string]any:
		mp := NewMap(nil)
		for k, v := range in {
			mp.Fields[k], err = NewValue(v)
			if err != nil {
				return nil, err
			}
		}
		return mp, nil
	}

	return nil, fmt.Errorf("NewValue error")
}

func NewJSONValue(in any) (val Value, err error) {

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
		arr := NewArray(make([]Value, len(in)))
		for i, item := range in {
			arr.Values[i], err = NewJSONValue(item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	case map[string]any:
		mp := NewMap(nil)
		for k, v := range in {
			mp.Fields[k], err = NewJSONValue(v)
			if err != nil {
				return nil, err
			}
		}
		return mp, nil
	}

	return nil, fmt.Errorf("NewJSONValue error")
}

func NewValueFromStruct(in *structpb.Value) (val Value, err error) {

	switch in := in.Kind.(type) {
	case *structpb.Value_BoolValue:
		return NewBoolean(in.BoolValue), nil
	case *structpb.Value_NumberValue:
		return NewNumberFromFloat(in.NumberValue), nil
	case *structpb.Value_StringValue:
		return NewString(in.StringValue), nil
	case *structpb.Value_ListValue:
		arr := NewArray(make([]Value, len(in.ListValue.Values)))
		for i, item := range in.ListValue.Values {
			arr.Values[i], err = NewValueFromStruct(item)
			if err != nil {
				return nil, err
			}
		}
		return arr, nil
	case *structpb.Value_StructValue:
		mp := NewMap(nil)
		for k, v := range in.StructValue.Fields {

			mp.Fields[k], err = NewValueFromStruct(v)
			if err != nil {
				return nil, err
			}
		}
		return mp, nil
	}

	return nil, fmt.Errorf("NewValueFromStruct error")
}

func init() {
	gob.Register(&Map{})
	gob.Register(&Array{})
	gob.Register(&Boolean{})
	gob.Register(&Number{})
	gob.Register(&String{})
	gob.Register(&ByteArray{})
	gob.Register(&File{})
	gob.Register(&Image{})
	gob.Register(&Video{})
	gob.Register(&Audio{})
	gob.Register(&Document{})
}
