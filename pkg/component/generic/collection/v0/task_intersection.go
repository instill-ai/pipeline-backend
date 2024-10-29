package collection

import (
	"context"
	"encoding/json"

	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) intersection(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	sets := in.Fields["sets"].GetListValue().Values

	if len(sets) == 1 {
		out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
		out.Fields["set"] = structpb.NewListValue(sets[0].GetListValue())
		return out, nil
	}

	curr := make([]string, len(sets[0].GetListValue().Values))
	for idx, v := range sets[0].GetListValue().Values {
		b, err := protojson.Marshal(v)
		if err != nil {
			return nil, err
		}
		curr[idx] = string(b)
	}

	for _, s := range sets[1:] {
		next := make([]string, len(s.GetListValue().Values))
		for idx, v := range s.GetListValue().Values {
			b, err := protojson.Marshal(v)
			if err != nil {
				return nil, err
			}
			next[idx] = string(b)
		}

		i := lo.Intersect(curr, next)
		curr = i

	}

	set := &structpb.ListValue{Values: make([]*structpb.Value, len(curr))}

	for idx, c := range curr {
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
