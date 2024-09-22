package data

import (
	"fmt"

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

func (a *Array) Get(path string) (v Value, err error) {
	if path == "" {
		return a, nil
	}
	path, err = StandardizePath(path)
	if err != nil {
		return nil, err
	}
	index, remainingPath, err := trimFirstIndexFromPath(path)
	if err != nil {
		return nil, err
	}
	if index >= len(a.Values) {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	return a.Values[index].Get(remainingPath)
}
func (a Array) ToStructValue() (v *structpb.Value, err error) {
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
