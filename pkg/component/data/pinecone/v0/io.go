// TODO: TASK_QUERY and TASK_RERANK are not refactored yet, they will be
// addressed in INS-7102.
package pinecone

import (
	"fmt"

	"github.com/pinecone-io/go-pinecone/pinecone"

	"github.com/instill-ai/pipeline-backend/pkg/data"
)

type vector struct {
	ID       string    `instill:"id"`
	Values   []float32 `instill:"values"`
	Metadata data.Map  `instill:"metadata"`
}

func (v vector) toPinecone() (*pinecone.Vector, error) {
	metadata, err := v.Metadata.ToStructValue()
	if err != nil {
		return nil, fmt.Errorf("converting input metadata to request: %w", err)
	}

	return &pinecone.Vector{
		Id:       v.ID,
		Values:   v.Values,
		Metadata: metadata.GetStructValue(),
	}, nil
}

type taskUpsertInput struct {
	vector
	Namespace string `instill:"namespace"`
}

func (in *taskUpsertInput) asRequest() (*upsertReq, error) {
	pv, err := in.vector.toPinecone()
	if err != nil {
		return nil, err
	}

	return &upsertReq{
		Vectors:   []*pinecone.Vector{pv},
		Namespace: in.Namespace,
	}, nil
}

type taskBatchUpsertInput struct {
	Vectors   []vector `instill:"vectors"`
	Namespace string   `instill:"namespace"`
}

func (in *taskBatchUpsertInput) asRequest() (*upsertReq, error) {
	req := &upsertReq{
		Vectors:   make([]*pinecone.Vector, 0, len(in.Vectors)),
		Namespace: in.Namespace,
	}

	for _, v := range in.Vectors {
		pv, err := v.toPinecone()
		if err != nil {
			return nil, err
		}

		req.Vectors = append(req.Vectors, pv)
	}

	return req, nil
}

type taskUpsertOutput struct {
	UpsertedCount int64 `instill:"upserted-count"`
}
