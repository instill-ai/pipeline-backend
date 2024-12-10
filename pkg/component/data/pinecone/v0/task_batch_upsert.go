package pinecone

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func (e *execution) batchUpsert(ctx context.Context, job *base.Job) error {
	input := taskBatchUpsertInput{}
	if err := job.Input.ReadData(ctx, &input); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	resp := upsertResp{}
	_, err := newIndexClient(e.Setup, e.GetLogger()).
		R().
		SetResult(&resp).
		SetBody(upsertReq{
			Vectors:   input.Vectors,
			Namespace: input.Namespace,
		}).
		Post(upsertPath)
	if err != nil {
		err = httpclient.WrapURLError(fmt.Errorf("upserting vectors: %w", err))
		job.Error.Error(ctx, err)
		return err
	}

	if err := job.Output.WriteData(ctx, &taskUpsertOutput{
		UpsertedCount: resp.RecordsUpserted,
	}); err != nil {

		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}
