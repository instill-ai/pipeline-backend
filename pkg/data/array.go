package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type Array []value.Value

func (Array) IsValue() {}

func (a Array) Get(path string) (v value.Value, err error) {
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
	if index >= len(a) {
		return nil, fmt.Errorf("path not found: %s", path)
	}

	return a[index].Get(remainingPath)
}
func (a Array) ToStructValue() (v *structpb.Value, err error) {
	arr := &structpb.ListValue{Values: make([]*structpb.Value, len(a))}
	for idx, v := range a {
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
