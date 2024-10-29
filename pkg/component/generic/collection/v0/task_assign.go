package collection

import (
	"context"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) assign(in *structpb.Struct, _ *base.Job, _ context.Context) (*structpb.Struct, error) {
	out := in
	return out, nil
}
