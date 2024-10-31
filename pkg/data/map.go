package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type Map map[string]format.Value

func (Map) IsValue() {}

func (m Map) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return m, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}

	if firstSeg.SegmentType == path.KeySegment {
		if v, ok := m[firstSeg.Key]; !ok {
			return nil, fmt.Errorf("path not found: %s", p)
		} else {
			return v.Get(remainingPath)
		}
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (m Map) ToStructValue() (v *structpb.Value, err error) {
	mp := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	for k, v := range m {
		if v != nil {
			switch v := v.(type) {
			case *nullData:
			default:
				mp.Fields[k], err = v.ToStructValue()
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return structpb.NewStructValue(mp), nil
}

func (m Map) Equal(other format.Value) bool {
	if other, ok := other.(Map); ok {
		if len(m) != len(other) {
			return false
		}
		for k, v := range m {
			if !v.Equal(other[k]) {
				return false
			}
		}
		return true
	}
	return false
}
