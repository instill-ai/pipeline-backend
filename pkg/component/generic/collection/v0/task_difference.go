package collection

import (
	"context"
	"encoding/json"

	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) difference(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	setA := in.Fields["set-a"]
	setB := in.Fields["set-b"]

	valuesA := make([]string, len(setA.GetListValue().Values))
	for idx, v := range setA.GetListValue().Values {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		valuesA[idx] = string(b)
	}

	valuesB := make([]string, len(setB.GetListValue().Values))
	for idx, v := range setB.GetListValue().Values {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		valuesB[idx] = string(b)
	}
	dif, _ := lo.Difference(valuesA, valuesB)

	set := &structpb.ListValue{Values: make([]*structpb.Value, len(dif))}

	for idx, c := range dif {
		var a any

		err := json.Unmarshal([]byte(c), &a)
		if err != nil {
			return nil, err
		}
		v, err := structpb.NewValue(a)
		if err != nil {
			return nil, err
		}
		set.Values[idx] = v
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["set"] = structpb.NewListValue(set)
	return out, nil
}
