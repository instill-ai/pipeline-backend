package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type Array []format.Value

func (Array) IsValue() {}

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
