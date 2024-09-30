package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

//go:embed testdata/cls-dog.json
var clsDogJSON []byte

// TestDrawClassification tests the drawClassification function
func TestDrawClassification(t *testing.T) {
	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "Classification Dog",
			inputJSON: clsDogJSON,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputData := &structpb.Struct{}
			if err := json.Unmarshal(tc.inputJSON, inputData); err != nil {
				t.Fatalf("Failed to unmarshal test data: %v", err)
			}

			bc := base.Component{}
			component := Init(bc)

			e, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_CLASSIFICATION",
			})

			if err != nil {
				t.Fatalf("drawClassification create execution returned an error: %v", err)
			}

			ir, ow, eh, job := base.GenerateMockJob(t)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			if err := e.Execute(context.Background(), []*base.Job{job}); err != nil {
				t.Fatalf("drawClassification returned an error: %v", err)
			}

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
