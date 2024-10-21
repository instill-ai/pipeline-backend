package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type Map map[string]value.Value

func (Map) IsValue() {}

func (m Map) Get(path string) (v value.Value, err error) {

	if path == "" {
		return m, nil
	}
	path, err = StandardizePath(path)
	if err != nil {
		return nil, err
	}
	key, remainingPath, err := trimFirstKeyFromPath(path)
	if err != nil {
		return nil, err
	}

	if v, ok := m[key]; !ok {
		return nil, fmt.Errorf("path not found: %s", path)
	} else {
		return v.Get(remainingPath)
	}
}

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
