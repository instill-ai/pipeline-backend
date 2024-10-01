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

//go:embed testdata/cls-dog.json
var clsDogJSON []byte

// TestDrawClassification tests the drawClassification function
func TestDrawClassification(t *testing.T) {
	c := quicktest.New(t)

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
		c.Run(tc.name, func(c *quicktest.C) {
			inputData := &structpb.Struct{}
			err := json.Unmarshal(tc.inputJSON, inputData)
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("Failed to unmarshal test data"))

			bc := base.Component{}
			component := Init(bc)

			e, err := component.CreateExecution(base.ComponentExecution{
					Component: component,
					Task:      "TASK_DRAW_CLASSIFICATION",
				})

			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawClassification create execution returned an error"))

			ir, ow, eh, job := base.GenerateMockJob(c)
			ir.ReadMock.Expect(context.Background()).Return(inputData, nil)
			ow.WriteMock.Times(1).Return(nil)

			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, quicktest.IsNil, quicktest.Commentf("drawClassification returned an error"))

			ir.MinimockFinish()
			ow.MinimockFinish()
			eh.MinimockFinish()
		})
	}
}
