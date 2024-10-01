package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	"github.com/frankban/quicktest"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

//go:embed testdata/det-coco-1.json
var detCOCO1JSON []byte

//go:embed testdata/det-coco-2.json
var detCOCO2JSON []byte

// TestDrawDetection tests the drawDetection function
func TestDrawDetection(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "Detection COCO 1",
			inputJSON: detCOCO1JSON,
		},
		{
			name:      "Detection COCO 2",
			inputJSON: detCOCO2JSON,
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
				Task:      "TASK_DRAW_DETECTION",
			})

			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawDetection create execution returned an error"))

			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawDetection returned an error"))

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
