package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

//go:embed testdata/inst-seg-coco-1.json
var instSegCOCO1JSON []byte

//go:embed testdata/inst-seg-coco-2.json
var instSegCOCO2JSON []byte

//go:embed testdata/inst-seg-stomata.json
var instSegStomataJSON []byte

// TestDrawInstanceSegmentation tests the drawInstanceSegmentation function
func TestDrawInstanceSegmentation(t *testing.T) {
	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "COCO Instance Segmentation 1",
			inputJSON: instSegCOCO1JSON,
		},
		{
			name:      "COCO Instance Segmentation 2",
			inputJSON: instSegCOCO2JSON,
		},
		{
			name:      "Stomata Instance Segmentation",
			inputJSON: instSegStomataJSON,
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
				Task:      "TASK_DRAW_INSTANCE_SEGMENTATION",
			})

			if err != nil {
				t.Fatalf("drawInstanceSegmentation create execution returned an error: %v", err)
			}

			ir, ow, eh, job := base.GenerateMockJob(t)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			if err := e.Execute(context.Background(), []*base.Job{job}); err != nil {
				t.Fatalf("drawInstanceSegmentation returned an error: %v", err)
			}

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
