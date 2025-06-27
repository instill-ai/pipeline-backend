package document

import (
	"bytes"
	"context"
	"fmt"

	pdfcpu "github.com/pdfcpu/pdfcpu/pkg/api"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/x/errmsg"
)

func (e *execution) splitInPages(ctx context.Context, job *base.Job) error {
	in := SplitInPagesInput{}
	if err := job.Input.ReadData(ctx, &in); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	ct := in.Document.ContentType().String()
	fe := util.TransformContentTypeToFileExtension(ct)
	if fe != "pdf" {
		// return fmt.Errorf("invalid file extension: %s", fe)
		return errmsg.AddMessage(
			fmt.Errorf("invalid file extension"),
			"Page split task takes only PDF documents.",
		)
	}

	b, err := in.Document.Binary()
	if err != nil {
		return fmt.Errorf("converting document to byte array: %w", err)
	}

	batchSize := int(in.BatchSize)
	if batchSize == 0 {
		batchSize = 1
	}

	rs := bytes.NewReader(b.ByteArray())
	rawPages, err := pdfcpu.SplitRaw(rs, batchSize, nil)
	if err != nil {
		return fmt.Errorf("splitting PDF: %w", err)
	}

	pages := make([]format.Document, len(rawPages))
	for i, rawPage := range rawPages {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rawPage.Reader); err != nil {
			return fmt.Errorf("reading bytes from split page: %w", err)
		}

		page, err := data.NewDocumentFromBytes(buf.Bytes(), in.Document.ContentType().String(), "")
		if err != nil {
			return fmt.Errorf("creating document from split page: %w", err)
		}

		pages[i] = page
	}

	out := SplitInPagesOutput{Batches: pages}
	return job.Output.WriteData(ctx, out)
}
