package collection

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) concat(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	arrays := in.Fields["arrays"].GetListValue().Values
	concat := &structpb.ListValue{Values: []*structpb.Value{}}

	for _, a := range arrays {
		concat.Values = append(concat.Values, a.GetListValue().Values...)
	}

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["array"] = structpb.NewListValue(concat)
	return out, nil
}
