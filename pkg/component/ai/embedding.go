package ai

// EmbeddingInput is the standardized input for the embedding model.
type EmbeddingInput struct {
	// Data is the the standardized input data for the embedding model.
	Data      EmbeddingInputData `json:"data"`
	// Parameter is the standardized parameter for the embedding model.
	Parameter EmbeddingParameter `json:"parameter"`
}

// EmbeddingInputData is the standardized input data for the embedding model.
type EmbeddingInputData struct {
	// Model is the model name.
	Model      string           `json:"model"`
	// Embeddings is the list of data to be embedded.
	Embeddings []InputEmbedding `json:"embeddings"`
}

// InputEmbedding is the standardized input data to be embedded.
type InputEmbedding struct {
	// Type is the type of the input data. It can be either "text", "image-url", or "image-base64".
	Type        string `json:"type"`
	// Text is the text to be embedded.
	Text        string `json:"text"`
	// ImageURL is the URL of the image to be embedded.
	ImageURL    string `json:"image-url"`
	// ImageBase64 is the base64 encoded image to be embedded.
	ImageBase64 string `json:"image-base64"`
}

// EmbeddingParameter is the standardized parameter for the embedding model.
type EmbeddingParameter struct {
	// Format is the format of the output embeddings. Default is "float", can be "float" or "base64".
	Format     string `json:"format"`
	// Dimensions is the number of dimensions of the output embeddings.
	Dimensions int    `json:"dimensions"`
	// InputType is the type of the input data. It can be "query" or "data".
	InputType  string `json:"input-type"`
	// Truncate is how to handle inputs longer than the max token length. Defaults to 'End'. Can be 'End', 'Start', or 'None'.
	Truncate   string `json:"truncate"`
}

// EmbeddingOutput is the standardized output for the embedding model.
type EmbeddingOutput struct {
	// Data is the standardized output data for the embedding model.
	Data EmbeddingOutputData `json:"data"`
}

// EmbeddingOutputData is the standardized output data for the embedding model.
type EmbeddingOutputData struct {
	// Embeddings is the list of output embeddings.
	Embeddings []OutputEmbedding `json:"embeddings"`
}

// OutputEmbedding is the standardized output embedding.
type OutputEmbedding struct {
	// Index is the index of the output embedding.
	Index   int   `json:"index"`
	// Vector is the output embedding.
	Vector  []any `json:"vector"`
	// Created is the Unix timestamp (in seconds) of when the embedding was created.
	Created int   `json:"created"`
}
