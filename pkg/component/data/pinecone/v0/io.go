// TASK_QUERY and TASK_RERANK are not refactored yet, they will be addressed in ins-7102
package pinecone

type taskUpsertInput struct {
	id        string    `instill:"id"`
	metadata  string    `instill:"metadata"`
	values    []float64 `instill:"values"`
	namespace string    `instill:"namespace"`
}

type taskUpsertOutput struct {
	upsertedCount int64 `instill:"upserted-count"`
}
