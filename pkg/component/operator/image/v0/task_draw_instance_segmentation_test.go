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

//go:embed testdata/inst-seg-coco-1.json
var instSegCOCO1JSON []byte

//go:embed testdata/inst-seg-coco-2.json
var instSegCOCO2JSON []byte

//go:embed testdata/inst-seg-stomata.json
var instSegStomataJSON []byte

// TestDrawInstanceSegmentation tests the drawInstanceSegmentation function
func TestDrawInstanceSegmentation(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "Instance Segmentation COCO 1",
			inputJSON: instSegCOCO1JSON,
		},
		{
			name:      "Instance Segmentation COCO 2",
			inputJSON: instSegCOCO2JSON,
		},
		{
			name:      "Instance Segmentation Stomata",
			inputJSON: instSegStomataJSON,
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
				Task:      "TASK_DRAW_INSTANCE_SEGMENTATION",
			})

			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawInstanceSegmentation create execution returned an error"))

			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawInstanceSegmentation returned an error"))

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
