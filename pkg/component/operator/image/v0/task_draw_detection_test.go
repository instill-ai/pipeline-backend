package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

//go:embed testdata/det-coco-1.json
var detCOCO1JSON []byte

//go:embed testdata/det-coco-2.json
var detCOCO2JSON []byte

// TestDrawDetection tests the drawDetection function
func TestDrawDetection(t *testing.T) {
	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "COCO Detection 1",
			inputJSON: detCOCO1JSON,
		},
		{
			name:      "COCO Detection 2",
			inputJSON: detCOCO2JSON,
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
				Task:      "TASK_DRAW_DETECTION",
			})

			if err != nil {
				t.Fatalf("drawDetection create execution returned an error: %v", err)
			}

			ir, ow, eh, job := base.GenerateMockJob(t)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			if err := e.Execute(context.Background(), []*base.Job{job}); err != nil {
				t.Fatalf("drawDetection returned an error: %v", err)
			}

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
