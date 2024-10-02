//go:build ocr
// +build ocr

package document

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	"code.sajari.com/docconv"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

// TestConvertToText tests the convert to text task
func TestConvertToText(t *testing.T) {
	c := qt.New(t)
	tests := []struct {
		name     string
		filepath string
	}{
		{
			name:     "Convert pdf file",
			filepath: "testdata/test.pdf",
		},
		{
			name:     "Convert docx file",
			filepath: "testdata/test.docx",
		},
		{
			name:     "Convert html file",
			filepath: "testdata/test.html",
		},
		{
			name:     "Convert odt file",
			filepath: "testdata/test.odt",
		},
		{
			name:     "Convert rtf file",
			filepath: "testdata/test.rtf",
		},
		{
			name:     "Convert png file",
			filepath: "testdata/test.png",
		},
		{
			name:     "Convert jpg file",
			filepath: "testdata/test.jpg",
		},
		{
			name:     "Convert tiff file",
			filepath: "testdata/test.tif",
		},
		{
			name:     "Convert txt file",
			filepath: "testdata/test.txt",
		},
		{
			name:     "Convert md file",
			filepath: "testdata/test.md",
		},
		{
			name:     "Convert csv file",
			filepath: "testdata/test.csv",
		},
		{
			name:     "Convert xlsx file",
			filepath: "testdata/test.xlsx",
		},
	}

	bc := base.Component{}
	for _, test := range tests {
		c.Run(test.name, func(c *qt.C) {
			component := Init(bc)
			// Read the fileContent content
			fileContent, err := os.ReadFile(test.filepath)
			c.Assert(err, qt.IsNil)

			base64DataURI := fmt.Sprintf("data:%s;base64,%s", docconv.MimeTypeByExtension(test.filepath), base64.StdEncoding.EncodeToString(fileContent))

			input := &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"document": {Kind: &structpb.Value_StringValue{StringValue: base64DataURI}},
				},
			}

			e, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CONVERT_TO_TEXT",
			})
			c.Assert(err, qt.IsNil)

			if test.name == "Convert xlsx file" {

				ir, ow, eh, job := mock.GenerateMockJob(c)
				ir.ReadMock.Return(input, nil)
				ow.WriteMock.Optional().Return(nil)
				eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
					c.Assert(err, qt.ErrorMatches, "unsupported content type")
				})
				err = e.Execute(context.Background(), []*base.Job{job})
				c.Assert(err, qt.IsNil)

				return
			}

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(input, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				c.Assert(output.Fields["body"].GetStringValue(), qt.Not(qt.Equals), "")
				c.Assert(output.Fields["meta"].GetStructValue(), qt.IsNotNil)
				return nil
			})
			eh.ErrorMock.Optional()

			err = e.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)

		})
	}

}
