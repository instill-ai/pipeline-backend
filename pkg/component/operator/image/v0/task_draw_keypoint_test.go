package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"github.com/frankban/quicktest"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

//go:embed testdata/kp-coco-1.json
var kpCOCO1JSON []byte

//go:embed testdata/kp-coco-2.json
var kpCOCO2JSON []byte

// TestDrawKeypoint tests the drawKeypoint function
func TestDrawKeypoint(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "Keypoint COCO 1",
			inputJSON: kpCOCO1JSON,
		},
		{
			name:      "Keypoint COCO 2",
			inputJSON: kpCOCO2JSON,
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *quicktest.C) {
			inputData := &structpb.Struct{}
			err := json.Unmarshal(tc.inputJSON, inputData)
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("Failed to unmarshal test data"))

			bc := base.Component{}
			component := Init(bc)

			e, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_KEYPOINT",
			})

			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawKeypoint create execution returned an error"))

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawKeypoint returned an error"))

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
