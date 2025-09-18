package document

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// TestFastConversions tests only conversions that don't require external dependencies
// This ensures CI/CD pipelines can run basic functionality tests quickly
func TestFastConversions(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	tests := []struct {
		name        string
		filepath    string
		contentType string
		expected    ConvertDocumentToMarkdownOutput
	}{
		{
			name:        "Convert HTML file",
			filepath:    "testdata/test.html",
			contentType: "text/html",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "This is test file",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
		{
			name:        "Convert CSV file",
			filepath:    "testdata/test.csv",
			contentType: "text/csv",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "| test | test | tse |\n| --- | --- | --- |\n| 1 | 23 | 2 |\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
		{
			name:        "Convert XLSX file",
			filepath:    "testdata/test.xlsx",
			contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# Sheet1\n| test | test | tse |\n| --- | --- | --- |\n| 1 | 23 | 2 |\n\n\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
		{
			name:        "Convert XLS file",
			filepath:    "testdata/test.xls",
			contentType: "application/vnd.ms-excel",
			expected: ConvertDocumentToMarkdownOutput{
				Body:          "# Sheet1\n| Name | Age |  |\n| --- | --- | --- |\n| ChunHao | 27 |  |\n| Benny | 27 |  |\n| Kevin | 27 |  |\n\n\n# Sheet2\n| Name | Age |  |\n| --- | --- | --- |\n| ChunHao | 28 |  |\n| Benny | 28 |  |\n| Kevin | 28 |  |\n\n\n",
				Images:        []format.Image{},
				AllPageImages: []format.Image{},
			},
		},
	}

	for _, test := range tests {
		c.Run(test.name, func(c *qt.C) {
			c.Parallel()

			ctx := context.Background()
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CONVERT_TO_MARKDOWN",
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)

			// Use cached file content for better performance
			fileContent, err := getTestFileContent(test.filepath)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *ConvertDocumentToMarkdownInput:
					*input = ConvertDocumentToMarkdownInput{
						Document: func() format.Document {
							doc, err := data.NewDocumentFromBytes(fileContent, test.contentType, "")
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

// TestComponentInitialization tests basic component setup without external dependencies
func TestComponentInitialization(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	component := Init(base.Component{})
	c.Assert(component, qt.IsNotNil)

	// Test all task types can be created
	tasks := []string{
		"TASK_CONVERT_TO_MARKDOWN",
		"TASK_CONVERT_TO_TEXT",
		"TASK_CONVERT_TO_IMAGES",
		"TASK_SPLIT_IN_PAGES",
	}

	for _, task := range tasks {
		c.Run(task, func(c *qt.C) {
			c.Parallel()

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      task,
			})
			c.Assert(err, qt.IsNil)
			c.Assert(execution, qt.IsNotNil)
		})
	}
}
