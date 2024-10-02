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

//go:embed testdata/ocr-mm.json
var ocrMMJSON []byte

// TestDrawOCR tests the drawOCR function
func TestDrawOCR(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name      string
		inputJSON []byte
	}{
		{
			name:      "OCR MM",
			inputJSON: ocrMMJSON,
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
				Task:      "TASK_DRAW_OCR",
			})

			if err != nil {
				c.Fatal(err)
			}

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			if err := e.Execute(context.Background(), []*base.Job{job}); err != nil {
				c.Fatal(err)
			}

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
