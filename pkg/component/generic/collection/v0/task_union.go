package collection

import (
	"context"
	"encoding/json"

	"github.com/samber/lo"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) union(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	sets := in.Fields["sets"].GetListValue().Values
	cache := [][]string{}

	for _, s := range sets {
		c := []string{}
		for _, v := range s.GetListValue().Values {
			b, err := protojson.Marshal(v)
			if err != nil {
				return nil, err
			}
			c = append(c, string(b))
		}
		cache = append(cache, c)
	}

	set := &structpb.ListValue{Values: []*structpb.Value{}}
	un := lo.Union(cache...)
	for _, u := range un {
		var a any
		err := json.Unmarshal([]byte(u), &a)
		if err != nil {
			return nil, err
		}
		v, err := structpb.NewValue(a)
		if err != nil {
			return nil, err
		}
		set.Values = append(set.Values, v)
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["set"] = structpb.NewListValue(set)
	return out, nil
}
