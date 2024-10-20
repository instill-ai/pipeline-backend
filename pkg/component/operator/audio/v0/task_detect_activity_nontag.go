//go:build !onnx
// +build !onnx

package audio

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func detectActivity(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	return nil, fmt.Errorf("the Audio operator wasn't built with onnxruntime")
}
