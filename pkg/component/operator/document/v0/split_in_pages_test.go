package document

import (
	"bytes"
	"context"
	"testing"

	pdfreader "github.com/dslipak/pdf"
	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"

	errorsx "github.com/instill-ai/x/errors"
)

func Test_SplitInPages(t *testing.T) {
	c := qt.New(t)
	c.Parallel()

	testCases := []struct {
		name          string
		batchSize     uint32
		filePath      string
		contentType   string
		wantFilePaths []string
		wantErr       string
	}{
		{
			name:        "nok - non-PDF document",
			filePath:    "testdata/test.docx",
			contentType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
			wantErr:     ".*only PDF documents.*",
		},
		{
			name:        "ok - 1-page batches",
			batchSize:   1,
			filePath:    "testdata/split-in-pages-input.pdf",
			contentType: "application/pdf",
			wantFilePaths: []string{
				"testdata/split-in-pages-output-1.pdf",
				"testdata/split-in-pages-output-2.pdf",
				"testdata/split-in-pages-output-3.pdf",
			},
		},
		{
			name:        "ok - 2-page batches",
			batchSize:   2,
			filePath:    "testdata/split-in-pages-input.pdf",
			contentType: "application/pdf",
			wantFilePaths: []string{
				"testdata/split-in-pages-output-1-2.pdf",
				"testdata/split-in-pages-output-3.pdf",
			},
		},
		{
			name:        "ok - 4-page batches",
			batchSize:   4,
			filePath:    "testdata/split-in-pages-input.pdf",
			contentType: "application/pdf",
			wantFilePaths: []string{
				"testdata/split-in-pages-input.pdf",
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			c.Parallel()

			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      taskSplitInPages,
			})
			c.Assert(err, qt.IsNil)

			// Use cached file content for better performance
			fileContent, err := getTestFileContent(tc.filePath)
			c.Assert(err, qt.IsNil)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Times(1).Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *SplitInPagesInput:
					*input = SplitInPagesInput{
						Document: func() format.Document {
							doc, err := data.NewDocumentFromBytes(fileContent, tc.contentType, "")
							if err != nil {
								return nil
							}
							return doc
						}(),
						BatchSize: tc.batchSize,
					}
				}
				return nil
			})

			// output capture
			var capturedOutput SplitInPagesOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(SplitInPagesOutput)
				return nil
			})

			// error capture
			var capturedErr error
			eh.ErrorMock.Set(func(ctx context.Context, err error) {
				capturedErr = err
			})

			if tc.wantErr == "" {
				eh.ErrorMock.Optional()
			} else {
				ow.WriteDataMock.Optional()
			}

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)

			if tc.wantErr != "" {
				c.Assert(capturedErr, qt.IsNotNil)
				c.Check(errorsx.Message(capturedErr), qt.Matches, tc.wantErr)
				return
			}

			c.Assert(err, qt.IsNil)

			c.Check(capturedOutput.Batches, qt.HasLen, len(tc.wantFilePaths))
			for i, batch := range capturedOutput.Batches {
				c.Check(batch.ContentType().String(), qt.Equals, "application/pdf")

				// If we just compare the binary data in the the output and the
				// expected files, we might see differences due to various
				// factors in PDF structure and generation (e.g., compression
				// levels, encoding, metadata). Therefore, we'll compare the
				// _text_ in the PDFs.

				// Read output text
				gotBytes := func() []byte {
					b, err := batch.Binary()
					c.Assert(err, qt.IsNil)
					return b.ByteArray()
				}()

				gotReader, err := pdfreader.NewReader(bytes.NewReader(gotBytes), int64(len(gotBytes)))
				c.Assert(err, qt.IsNil)

				gotText, err := textFromReader(gotReader)
				c.Assert(err, qt.IsNil)

				// Read expected text
				wantReader, err := pdfreader.Open(tc.wantFilePaths[i])
				c.Assert(err, qt.IsNil)

				wantText, err := textFromReader(wantReader)
				c.Assert(err, qt.IsNil)

				// Compare
				c.Check(gotText, qt.Equals, wantText, qt.Commentf("comparing page %d", i))
			}
		})
	}
}

func textFromReader(r *pdfreader.Reader) (string, error) {
	var buf bytes.Buffer
	tr, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	if _, err := buf.ReadFrom(tr); err != nil {
		return "", err
	}
	return buf.String(), nil
}
