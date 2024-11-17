package document

import (
	"context"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestConvertToText(t *testing.T) {
	c := qt.New(t)
	tests := []struct {
		name     string
		filepath string
		expected ConvertToTextOutput
	}{
		{
			name:     "Convert pdf file",
			filepath: "testdata/test.pdf",
			expected: ConvertToTextOutput{
				Body: "This is test file for markdown",
				Meta: map[string]string{
					"Encrypted":      "no",
					"File size":      "15489 bytes",
					"Form":           "none",
					"JavaScript":     "no",
					"Optimized":      "no",
					"PDF version":    "1.4",
					"Page rot":       "0",
					"Page size":      "596 x 842 pts (A4)",
					"Pages":          "1",
					"Producer":       "Skia/PDF m128 Google Docs Renderer",
					"Suspects":       "no",
					"Tagged":         "no",
					"Title":          "Untitled document",
					"UserProperties": "no",
				},
				MSecs: 3,
			},
		},
		{
			name:     "Convert docx file",
			filepath: "testdata/test.docx",
			expected: ConvertToTextOutput{
				Body: "This is test file for markdown",
				Meta: map[string]string{},
			},
		},
	}

	bc := base.Component{}
	for _, test := range tests {
		c.Run(test.name, func(c *qt.C) {
			component := Init(bc)
			ctx := context.Background()

			fileContent, err := os.ReadFile(test.filepath)
			c.Assert(err, qt.IsNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CONVERT_TO_TEXT",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *ConvertToTextInput:
					*input = ConvertToTextInput{
						Document: func() format.Document {
							doc, err := data.NewDocumentFromBytes(fileContent, mimeTypeByExtension(test.filepath), "")
							if err != nil {
								return nil
							}
							return doc
						}(),
					}
				}
				return nil
			})

			var capturedOutput any
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output
				return nil
			})
			eh.ErrorMock.Optional()

			err = execution.Execute(ctx, []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.(ConvertToTextOutput).Body, qt.DeepEquals, test.expected.Body)
			c.Assert(capturedOutput.(ConvertToTextOutput).Meta, qt.DeepEquals, test.expected.Meta)
			c.Assert(capturedOutput.(ConvertToTextOutput).Error, qt.DeepEquals, test.expected.Error)
			c.Assert(capturedOutput.(ConvertToTextOutput).Filename, qt.DeepEquals, test.expected.Filename)
		})
	}
}
