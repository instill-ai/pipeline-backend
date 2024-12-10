// TODO: TASK_QUERY and TASK_RERANK are not refactored yet, they will be
// addressed in INS-7102.
package pinecone

type vector struct {
	ID       string            `json:"id" instill:"id"`
	Values   []float64         `json:"values,omitempty" instill:"values"`
	Metadata map[string]string `json:"metadata,omitempty" instill:"metadata"`
}

type taskUpsertInput struct {
	vector
	Namespace string `instill:"namespace"`
}

type taskBatchUpsertInput struct {
	Vectors   []vector `instill:"vectors"`
	Namespace string   `instill:"namespace"`
}

type taskUpsertOutput struct {
	UpsertedCount int64 `instill:"upserted-count"`
}
