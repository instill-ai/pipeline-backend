package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type Array []format.Value

func (Array) IsValue() {}

var arrayGetters = map[string]func(Array) (format.Value, error){
	"length": func(a Array) (format.Value, error) { return a.Length(), nil },
}

func (a Array) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return a, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}
	if firstSeg.SegmentType == path.IndexSegment {
		index := firstSeg.Index
		if index >= len(a) {
			return nil, fmt.Errorf("path not found: %s", p)
		}
		return a[index].Get(remainingPath)
	} else if firstSeg.SegmentType == path.AttributeSegment {
		getter, exists := arrayGetters[firstSeg.Attribute]
		if !exists {
			return nil, fmt.Errorf("path not found: %s", p)
		}
		return getter(a)
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
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

func (a Array) Equal(other format.Value) bool {
	if other, ok := other.(Array); ok {
		if len(a) != len(other) {
			return false
		}
		for i, v := range a {
			if !v.Equal(other[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (a Array) Length() format.Number {
	return NewNumberFromInteger(len(a))
}
