package openai

const (
	embeddingsPath = "/v1/embeddings"
)

type TextEmbeddingsReq struct {
	Model      string   `json:"model"`
	Dimensions int      `json:"dimensions,omitempty"`
	Input      []string `json:"input"`
}

type TextEmbeddingsResp struct {
	Object string      `json:"object"`
	Data   []Data      `json:"data"`
	Model  string      `json:"model"`
	Usage  usageOpenAI `json:"usage"`
}

type Data struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}
