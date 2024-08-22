package data

import (
	"google.golang.org/protobuf/types/known/structpb"
)

type Array struct {
	Values []Value
}

func NewArray(v []Value) (arr *Array) {
	if v == nil {
		v = []Value{}
	}
	return &Array{
		Values: v,
	}
}

func (Array) isValue() {}

func (a *Array) ToStructValue() (v *structpb.Value, err error) {
	arr := &structpb.ListValue{Values: make([]*structpb.Value, len(a.Values))}
	for idx, v := range a.Values {
		if v == nil {
			arr.Values[idx] = structpb.NewNullValue()
		} else {
			arr.Values[idx], err = v.ToStructValue()
			if err != nil {
				return nil, err
			}
		}
	}
	return structpb.NewListValue(arr), nil
}
