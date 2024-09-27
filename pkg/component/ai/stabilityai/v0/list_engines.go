package stabilityai

const (
	listEnginesPath = "/v1/engines/list"
)

// Engine represents a Stability AI Engine.
type Engine struct {
	Description string `json:"description"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
}
