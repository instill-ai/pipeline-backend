// TASK_QUERY and TASK_RERANK are not refactored yet, they will be addressed in ins-7102
package pinecone

type taskUpsertInput struct {
	ID        string    `instill:"id"`
	Metadata  string    `instill:"metadata"`
	Values    []float64 `instill:"values"`
	Namespace string    `instill:"namespace"`
}

type taskUpsertOutput struct {
	UpsertedCount int64 `instill:"upserted-count"`
}
