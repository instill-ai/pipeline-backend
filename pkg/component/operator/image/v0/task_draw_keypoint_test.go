package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

//go:embed testdata/kp-coco-1.json
var kpCOCO1JSON []byte

//go:embed testdata/kp-coco-2.json
var kpCOCO2JSON []byte

// TestDrawKeypoint tests the drawKeypoint function
func TestDrawKeypoint(t *testing.T) {
	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "COCO Keypoint 1",
			inputJSON: kpCOCO1JSON,
		},
		{
			name:      "COCO Keypoint 2",
			inputJSON: kpCOCO2JSON,
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
				Task:      "TASK_DRAW_KEYPOINT",
			})

			if err != nil {
				t.Fatalf("drawKeypoint create execution returned an error: %v", err)
			}

			ir, ow, eh, job := base.GenerateMockJob(t)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			if err := e.Execute(context.Background(), []*base.Job{job}); err != nil {
				t.Fatalf("drawKeypoint returned an error: %v", err)
			}

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
