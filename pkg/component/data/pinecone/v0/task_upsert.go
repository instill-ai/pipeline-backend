package pinecone

import (
	"context"
	"fmt"

	"github.com/pinecone-io/go-pinecone/pinecone"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

type upsertReq struct {
	Vectors   []*pinecone.Vector `json:"vectors"`
	Namespace string             `json:"namespace,omitempty"`
}

type upsertResp struct {
	RecordsUpserted int64 `json:"upsertedCount"`
}

func (e *execution) upsert(ctx context.Context, job *base.Job) error {
	input := taskUpsertInput{}
	if err := job.Input.ReadData(ctx, &input); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	resp := upsertResp{}
	body, err := input.asRequest()
	if err != nil {
		job.Error.Error(ctx, err)
		return err
	}
	_, err = newIndexClient(e.Setup, e.GetLogger()).
		R().
		SetResult(&resp).
		SetBody(body).
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
