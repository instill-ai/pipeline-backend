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

func Test_ConvertDocumentToImages(t *testing.T) {
	c := qt.New(t)

	test := struct {
		name        string
		filepath    string
		expectedLen int
	}{
		name:        "Convert PDF to Images",
		filepath:    "testdata/test.pdf",
		expectedLen: 1,
	}

	bc := base.Component{}
	ctx := context.Background()

	component := Init(bc)
	c.Assert(component, qt.IsNotNil)

	execution, err := component.CreateExecution(base.ComponentExecution{
		Component: component,
		Task:      taskConvertToImages,
	})
	c.Assert(err, qt.IsNil)
	c.Assert(execution, qt.IsNotNil)

	fileContent, err := os.ReadFile(test.filepath)
	c.Assert(err, qt.IsNil)

	ir, ow, eh, job := mock.GenerateMockJob(c)

	ir.ReadDataMock.Times(1).Set(func(ctx context.Context, input any) error {
		switch input := input.(type) {
		case *ConvertDocumentToImagesInput:
			*input = ConvertDocumentToImagesInput{
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

	ow.WriteDataMock.Times(1).Set(func(ctx context.Context, output any) error {
		switch output := output.(type) {
		case *ConvertDocumentToImagesOutput:
			mock.Equal(len(output.Images), test.expectedLen)
		}
		return nil
	})

	eh.ErrorMock.Optional()

	err = execution.Execute(ctx, []*base.Job{job})
	c.Assert(err, qt.IsNil)
}
