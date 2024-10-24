package openai

const (
	createSpeechPath = "/v1/audio/speech"
)

type TextToSpeechReq struct {
	Input          string   `json:"input"`
	Model          string   `json:"model"`
	Voice          string   `json:"voice"`
	ResponseFormat *string  `json:"response_format,omitempty"`
	Speed          *float32 `json:"speed,omitempty"`
}
