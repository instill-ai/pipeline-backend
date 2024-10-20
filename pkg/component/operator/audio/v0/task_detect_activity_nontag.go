//go:build !onnx
// +build !onnx

package audio

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func detectActivity(ctx context.Context, job *base.Job) error {
	return fmt.Errorf("the Audio operator wasn't built with onnxruntime")
}
