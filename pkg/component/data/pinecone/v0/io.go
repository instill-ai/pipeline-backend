// TASK_QUERY and TASK_RERANK are not refactored yet, they will be addressed in ins-7102
package pinecone

type taskUpsertInput struct {
	ID        string      `instill:"id"`
	Metadata  interface{} `instill:"metadata"`
	Values    []float64   `instill:"values"`
	Namespace string      `instill:"namespace"`
}

type taskUpsertOutput struct {
	UpsertedCount int64 `instill:"upserted-count"`
}

type taskBatchUpsertInput struct {
	IDs           []string      `instill:"ids"`
	ArrayMetadata []interface{} `instill:"array-metadata"`
	ArrayValues   [][]float64   `instill:"array-values"`
	Namespace     string        `instill:"namespace"`
}

type taskBatchUpsertOutput struct {
	UpsertedCount int64 `instill:"upserted-count"`
}

type vector struct {
	ID       string      `instill:"id" json:"id"`
	Values   []float64   `instill:"values" json:"values,omitempty"`
	Metadata interface{} `instill:"metadata" json:"metadata,omitempty"`
}
