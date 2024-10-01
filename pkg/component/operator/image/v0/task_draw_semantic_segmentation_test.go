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

//go:embed testdata/sem-seg-cityscape.json
var semSegCityscapeJSON []byte

// TestDrawSemanticSegmentation tests the drawSemanticSegmentation function
func TestDrawSemanticSegmentation(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "Semantic Segmentation Cityscape",
			inputJSON: semSegCityscapeJSON,
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
				Task:      "TASK_DRAW_SEMANTIC_SEGMENTATION",
			})

			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawSemanticSegmentation create execution returned an error"))

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawSemanticSegmentation returned an error"))

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
