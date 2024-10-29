package collection

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) append(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	arr := in.Fields["array"]
	element := in.Fields["element"]
	arr.GetListValue().Values = append(arr.GetListValue().Values, element)

	out := &structpb.Struct{Fields: make(map[string]*structpb.Value)}
	out.Fields["array"] = arr
	return out, nil
}
