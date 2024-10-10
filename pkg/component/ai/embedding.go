package ai

type EmbeddingInput struct {
	Data      EmbeddingInputData `json:"data"`
	Parameter EmbeddingParameter `json:"parameter"`
}

type EmbeddingInputData struct {
	Model      string           `json:"model"`
	Embeddings []InputEmbedding `json:"embeddings"`
}

type InputEmbedding struct {
	Type        string `json:"type"`
	Text        string `json:"text"`
	ImageURL    string `json:"image-url"`
	ImageBase64 string `json:"image-base64"`
}

type EmbeddingParameter struct {
	Format     string `json:"format"`
	Dimensions int    `json:"dimensions"`
	InputType  string `json:"input-type"`
	Truncate   string `json:"truncate"`
}

type EmbeddingOutput struct {
	Data EmbeddingOutputData `json:"data"`
}

type EmbeddingOutputData struct {
	Embeddings []OutputEmbedding `json:"embeddings"`
}

type OutputEmbedding struct {
	Index   int   `json:"index"`
	Vector  []any `json:"vector"`
	Created int   `json:"created"`
}
