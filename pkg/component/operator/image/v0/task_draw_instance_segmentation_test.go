package image

import (
	"context"
	"encoding/json"
	"testing"

	_ "embed"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

//go:embed testdata/inst-seg-coco-1.json
var instSegCOCO1JSON []byte

//go:embed testdata/inst-seg-coco-2.json
var instSegCOCO2JSON []byte

//go:embed testdata/inst-seg-stomata.json
var instSegStomataJSON []byte

//go:embed testdata/inst-seg-coco-1.jpeg
var instSegCOCO1JPEG []byte

//go:embed testdata/inst-seg-coco-2.jpeg
var instSegCOCO2JPEG []byte

//go:embed testdata/inst-seg-stomata.jpeg
var instSegStomataJPEG []byte

// TestDrawInstanceSegmentation tests the drawInstanceSegmentation function
func TestDrawInstanceSegmentation(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name      string
		inputJPEG []byte
		inputJSON []byte

		expectedError  string
		expectedOutput bool
	}{
		{
			name:           "Instance Segmentation COCO 1",
			inputJPEG:      instSegCOCO1JPEG,
			inputJSON:      instSegCOCO1JSON,
			expectedOutput: true,
		},
		{
			name:           "Instance Segmentation COCO 2",
			inputJPEG:      instSegCOCO2JPEG,
			inputJSON:      instSegCOCO2JSON,
			expectedOutput: true,
		},
		{
			name:           "Instance Segmentation Stomata",
			inputJPEG:      instSegStomataJPEG,
			inputJSON:      instSegStomataJSON,
			expectedOutput: true,
		},
		{
			name:          "Invalid Image",
			inputJPEG:     []byte("invalid image data"),
			inputJSON:     instSegCOCO1JSON,
			expectedError: "convert image: failed to decode source image: invalid JPEG format: missing SOI marker",
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DRAW_INSTANCE_SEGMENTATION",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *drawInstanceSegmentationInput:
					img, err := data.NewImageFromBytes(tc.inputJPEG, "image/jpeg", "test")
					if err != nil {
						return err
					}
					var segmentationResult struct {
						Objects []*instanceSegmentationObject `json:"objects"`
					}
					err = json.Unmarshal(tc.inputJSON, &segmentationResult)
					if err != nil {
						return err
					}
					*input = drawInstanceSegmentationInput{
						Image:   img,
						Objects: segmentationResult.Objects,
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				return nil
			})
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.ErrorMatches, tc.expectedError)
			})
			if tc.expectedError != "" {
				ow.WriteDataMock.Optional()
			} else {
				eh.ErrorMock.Optional()
			}

			err = execution.Execute(context.Background(), []*base.Job{job})

			if tc.expectedError == "" {
				c.Assert(err, qt.IsNil)
				output, ok := capturedOutput.(drawInstanceSegmentationOutput)
				c.Assert(ok, qt.IsTrue)
				c.Assert(output.Image, qt.Not(qt.IsNil))
			}
		})
	}
}
