package pinecone

type queryInput struct {
	Namespace       string      `json:"namespace"`
	TopK            int64       `json:"top-k"`
	Vector          []float64   `json:"vector"`
	IncludeValues   bool        `json:"include-values"`
	IncludeMetadata bool        `json:"include-metadata"`
	ID              string      `json:"id"`
	Filter          interface{} `json:"filter"`
	MinScore        float64     `json:"min-score"`
}

type queryReq struct {
	Namespace       string      `json:"namespace"`
	TopK            int64       `json:"topK"`
	Vector          []float64   `json:"vector,omitempty"`
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
	vector
	Score float64 `json:"score"`
}

type upsertReq struct {
	Vectors   []vector `json:"vectors"`
	Namespace string   `json:"namespace,omitempty"`
}

type upsertInput struct {
	vector
	Namespace string `json:"namespace"`
}

type vector struct {
	ID       string      `json:"id"`
	Values   []float64   `json:"values,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

type upsertResp struct {
	RecordsUpserted int64 `json:"upsertedCount"`
}

type upsertOutput struct {
	RecordsUpserted int64 `json:"upserted-count"`
}

type errBody struct {
	Msg string `json:"message"`
}

func (e errBody) Message() string {
	return e.Msg
}
