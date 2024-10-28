package collection

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) split(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	arr := in.Fields["array"].GetListValue().Values
	size := int(in.Fields["group-size"].GetNumberValue())
	groups := make([][]*structpb.Value, 0)

	for i := 0; i < len(arr); i += size {
		end := i + size
		if end > len(arr) {
			end = len(arr)
		}
		groups = append(groups, arr[i:end])
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["arrays"] = structpb.NewListValue(&structpb.ListValue{Values: make([]*structpb.Value, len(groups))})

	for idx, g := range groups {
		out.Fields["arrays"].GetListValue().Values[idx] = structpb.NewListValue(&structpb.ListValue{Values: g})
	}

	return out, nil
}
