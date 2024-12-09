package pinecone

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) batchUpsert(ctx context.Context, job *base.Job) error {

	input := taskBatchUpsertInput{}
	if err := job.Input.ReadData(ctx, &input); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if err := valid(input); err != nil {
		err = fmt.Errorf("validate input: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	req := newIndexClient(e.Setup, e.GetLogger()).R()

	upsertReq := convertBatchUpsertInput(input)

	resp := upsertResp{}

	req.SetResult(&resp).SetBody(upsertReq)
	if _, err := req.Post(upsertPath); err != nil {
		err = fmt.Errorf("upserting vectors: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	output := taskBatchUpsertOutput{
		UpsertedCount: resp.RecordsUpserted,
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func valid(input taskBatchUpsertInput) error {
	if len(input.IDs) != len(input.ArrayValues) {
		return fmt.Errorf("ids and array-values must have the same length")
	}
	if len(input.ArrayValues) == 0 {
		return fmt.Errorf("array-values must not be empty")
	}
	return nil
}

func convertBatchUpsertInput(input taskBatchUpsertInput) upsertReq {

	vectors := make([]vector, len(input.IDs))
	for i, id := range input.IDs {
		vectors[i] = vector{
			ID:       id,
			Values:   input.ArrayValues[i],
			Metadata: input.ArrayMetadata[i],
		}
	}

	return upsertReq{
		Vectors:   vectors,
		Namespace: input.Namespace,
	}
}
