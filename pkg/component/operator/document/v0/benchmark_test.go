package document

import (
	"context"
	"os"
	"testing"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// BenchmarkFileReading compares cached vs uncached file reading
func BenchmarkFileReading(b *testing.B) {
	testFile := "testdata/test.pdf"

	b.Run("Uncached", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := readFileDirectly(testFile)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Cached", func(b *testing.B) {
		// Clear cache before benchmark
		clearTestFileCache()

		for i := 0; i < b.N; i++ {
			_, err := getTestFileContent(testFile)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkHTMLConversion benchmarks the fastest conversion (HTML)
func BenchmarkHTMLConversion(b *testing.B) {
	if !checkExternalDependency("python3") && !checkExternalDependency("python") {
		b.Skip("Python not found, skipping benchmark")
		return
	}

	fileContent, err := getTestFileContent("testdata/test.html")
	if err != nil {
		b.Fatal(err)
	}

	component := Init(base.Component{})
	execution, err := component.CreateExecution(base.ComponentExecution{
		Component: component,
		Task:      "TASK_CONVERT_TO_MARKDOWN",
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()

		ir, ow, eh, job := mock.GenerateMockJob(nil)
		ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
			switch input := input.(type) {
			case *ConvertDocumentToMarkdownInput:
				*input = ConvertDocumentToMarkdownInput{
					Document: func() format.Document {
						doc, err := data.NewDocumentFromBytes(fileContent, "text/html", "")
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

		ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
			return nil
		})
		eh.ErrorMock.Optional()

		err := execution.Execute(ctx, []*base.Job{job})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// readFileDirectly reads file without caching (for benchmark comparison)
func readFileDirectly(filepath string) ([]byte, error) {
	return os.ReadFile(filepath)
}
