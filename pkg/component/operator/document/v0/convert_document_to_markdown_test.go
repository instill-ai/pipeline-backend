package document

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestConvertDocumentToMarkdown(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name     string
		filepath string
		expected ConvertDocumentToMarkdownOutput
	}{
		{
			name:     "Convert PDF file",
			filepath: "testdata/test.pdf",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# This is test file for markdown\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
				Markdowns:     []string{"# This is test file for markdown\n"},
			},
		},
		{
			name:     "Convert DOCX file",
			filepath: "testdata/test.docx",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# This is test file for markdown\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
				Markdowns:     []string{"# This is test file for markdown\n"},
			},
		},
		{
			name:     "Convert HTML file",
			filepath: "testdata/test.html",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "This is test file",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
		{
			name:     "Convert PPTX file",
			filepath: "testdata/test.pptx",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# This           is     test          file       for markdown\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
				Markdowns:     []string{"# This           is     test          file       for markdown\n"},
			},
		},
		{
			name:     "Convert XLSX file",
			filepath: "testdata/test.xlsx",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# Sheet1\n| test | test | tse |\n| --- | --- | --- |\n| 1 | 23 | 2 |\n\n\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
		{
			name:     "Convert XLS file",
			filepath: "testdata/test.xls",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# Sheet1\n| Name | Age |  |\n| --- | --- | --- |\n| ChunHao | 27 |  |\n| Benny | 27 |  |\n| Kevin | 27 |  |\n\n\n# Sheet2\n| Name | Age |  |\n| --- | --- | --- |\n| ChunHao | 28 |  |\n| Benny | 28 |  |\n| Kevin | 28 |  |\n\n\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
		{
			name:     "Convert CSV file",
			filepath: "testdata/test.csv",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "| test | test | tse |\n| --- | --- | --- |\n| 1 | 23 | 2 |\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
	}

	bc := base.Component{}
	ctx := context.Background()

	for _, test := range tests {
		c.Run(test.name, func(c *qt.C) {
			component := Init(bc)
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CONVERT_TO_MARKDOWN",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			fileContent, err := os.ReadFile(test.filepath)
			c.Assert(err, qt.IsNil)

			base64DataURI := fmt.Sprintf("data:%s;base64,%s", mimeTypeByExtension(test.filepath), base64.StdEncoding.EncodeToString(fileContent))

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *ConvertDocumentToMarkdownInput:
					*input = ConvertDocumentToMarkdownInput{
						Document: func() format.Document {
							doc, err := data.NewDocumentFromURL(base64DataURI)
							if err != nil {
								return nil
							}
							return doc
						}(),
						DisplayImageTag: false,
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
			c.Assert(capturedOutput, qt.DeepEquals, test.expected)
		})
	}
}

func mimeTypeByExtension(filepath string) string {
	switch filepath {
	case "testdata/test.pdf":
		return "application/pdf"
	case "testdata/test.docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "testdata/test.html":
		return "text/html"
	case "testdata/test.pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case "testdata/test.xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case "testdata/test.xls":
		return "application/vnd.ms-excel"
	case "testdata/test.csv":
		return "text/csv"
	default:
		return ""
	}
}
