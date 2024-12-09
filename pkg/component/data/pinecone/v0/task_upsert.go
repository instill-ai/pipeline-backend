package pinecone

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) upsert(ctx context.Context, job *base.Job) error {

	input := taskUpsertInput{}
	if err := job.Input.ReadData(ctx, &input); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	req := newIndexClient(e.Setup, e.GetLogger()).R()

	upsertReq := convertInput(input)

	resp := upsertResp{}

	req.SetResult(&resp).SetBody(upsertReq)
	if _, err := req.Post(upsertPath); err != nil {
		err = fmt.Errorf("upserting vectors: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	output := convertOutput(resp)

	if err := job.Output.WriteData(ctx, output); err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func convertInput(input taskUpsertInput) upsertReq {

	upsertReq := upsertReq{
		Vectors:   []vector{},
		Namespace: input.Namespace,
	}

	vector := vector{
		ID:     input.ID,
		Values: input.Values,
	}

	if input.Metadata != "" {
		vector.Metadata = input.Metadata
	}

	upsertReq.Vectors = append(upsertReq.Vectors, vector)

	return upsertReq
}

func convertOutput(resp upsertResp) *taskUpsertOutput {
	return &taskUpsertOutput{
		UpsertedCount: resp.RecordsUpserted,
	}
}
