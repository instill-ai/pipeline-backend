package pinecone

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (e *execution) upsert(ctx context.Context, job *base.Job) error {

	input := &taskUpsertInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	req := newIndexClient(e.Setup, e.GetLogger()).R()

	upsertReq := convertInput(input)

	resp := upsertResp{}

	req.SetResult(&resp).SetBody(upsertReq)

	if _, err := req.Post(upsertPath); err != nil {
		return err
	}

	output := convertOutput(resp)

	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil
}

func convertInput(input *taskUpsertInput) upsertReq {

	upsertReq := upsertReq{
		Vectors:   []vector{},
		Namespace: input.namespace,
	}

	for _, v := range input.values {
		upsertReq.Vectors = append(upsertReq.Vectors, vector{
			ID:       input.id,
			Values:   []float64{v},
			Metadata: input.metadata,
		})
	}

	return upsertReq
}

func convertOutput(resp upsertResp) *taskUpsertOutput {
	return &taskUpsertOutput{
		upsertedCount: resp.RecordsUpserted,
	}
}

// case taskUpsert:
// 	req := newIndexClient(e.Setup, e.GetLogger()).R()

// 	v := upsertInput{}
// 	err := base.ConvertFromStructpb(input, &v)
// 	if err != nil {
// 		job.Error.Error(ctx, err)
// 		continue
// 	}

// 	resp := upsertResp{}
// 	req.SetResult(&resp).SetBody(upsertReq{
// 		Vectors:   []vector{v.vector},
// 		Namespace: v.Namespace,
// 	})

// 	if _, err := req.Post(upsertPath); err != nil {
// 		job.Error.Error(ctx, httpclient.WrapURLError(err))
// 		continue
// 	}

// 	output, err = base.ConvertToStructpb(upsertOutput(resp))
// 	if err != nil {
// 		job.Error.Error(ctx, err)
// 		continue
// 	}
