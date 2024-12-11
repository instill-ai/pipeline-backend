package pinecone

import "github.com/pinecone-io/go-pinecone/pinecone"

type queryInput struct {
	Namespace       string      `json:"namespace"`
	TopK            int64       `json:"top-k"`
	Vector          []float32   `json:"vector"`
	IncludeValues   bool        `json:"include-values"`
	IncludeMetadata bool        `json:"include-metadata"`
	ID              string      `json:"id"`
	Filter          interface{} `json:"filter"`
	MinScore        float64     `json:"min-score"`
}

type queryReq struct {
	Namespace       string      `json:"namespace"`
	TopK            int64       `json:"topK"`
	Vector          []float32   `json:"vector,omitempty"`
	IncludeValues   bool        `json:"includeValues"`
	IncludeMetadata bool        `json:"includeMetadata"`
	ID              string      `json:"id,omitempty"`
	Filter          interface{} `json:"filter,omitempty"`
}

func (q queryInput) asRequest() queryReq {
	return queryReq{
		Namespace:       q.Namespace,
		TopK:            q.TopK,
		Vector:          q.Vector,
		IncludeValues:   q.IncludeValues,
		IncludeMetadata: q.IncludeMetadata,
		ID:              q.ID,
		Filter:          q.Filter,
	}
}

type queryResp struct {
	Namespace string  `json:"namespace"`
	Matches   []match `json:"matches"`
}

func (r queryResp) filterOutBelowThreshold(th float64) queryResp {
	if th <= 0 {
		return r
	}

	matches := make([]match, 0, len(r.Matches))
	for _, match := range r.Matches {
		if match.Score >= th {
			matches = append(matches, match)
		}
	}
	r.Matches = matches

	return r
}

type match struct {
	*pinecone.Vector
	Score float64 `json:"score"`
}

type Document struct {
	Text string `json:"text"`
}

type rerankInput struct {
	// not taking model as input for now as only one model is supported for rerank task: https://docs.pinecone.io/guides/inference/understanding-inference#models
	//ModelName string   `json:"model-name"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top-n"`
}

func (r *rerankInput) asRequest() *rerankReq {
	reqDocuments := make([]Document, 0, len(r.Documents))
	for _, doc := range r.Documents {
		reqDocuments = append(reqDocuments, Document{Text: doc})
	}

	// TODO: make model configurable in tasks.json
	return &rerankReq{
		Model:     "bge-reranker-v2-m3",
		Query:     r.Query,
		TopN:      r.TopN,
		Documents: reqDocuments,
	}
}

type rerankReq struct {
	Model     string     `json:"model"`
	Query     string     `json:"query"`
	TopN      int        `json:"top_n,omitempty"`
	Documents []Document `json:"documents"`
}

type rerankResp struct {
	Data []struct {
		Index    int      `json:"index"`
		Document Document `json:"document"`
		Score    float64  `json:"score"`
	} `json:"data"`
}

func (r *rerankResp) toOutput() rerankOutput {
	documents := make([]string, 0, len(r.Data))
	scores := make([]float64, 0, len(r.Data))
	for _, d := range r.Data {
		documents = append(documents, d.Document.Text)
		scores = append(scores, d.Score)
	}
	return rerankOutput{
		Documents: documents,
		Scores:    scores,
	}
}

type rerankOutput struct {
	Documents []string  `json:"documents"`
	Scores    []float64 `json:"scores"`
}

type errBody struct {
	Msg string `json:"message"`
}

func (e errBody) Message() string {
	return e.Msg
}
