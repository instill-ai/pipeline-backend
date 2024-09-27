package openai

const (
	createSpeechPath = "/v1/audio/speech"
)

type TextToSpeechInput struct {
	Text           string   `json:"text"`
	Model          string   `json:"model"`
	Voice          string   `json:"voice"`
	ResponseFormat *string  `json:"response-format,omitempty"`
	Speed          *float64 `json:"speed,omitempty"`
}

type TextToSpeechOutput struct {
	Audio string `json:"audio"`
}

type TextToSpeechReq struct {
	Input          string   `json:"input"`
	Model          string   `json:"model"`
	Voice          string   `json:"voice"`
	ResponseFormat *string  `json:"response_format,omitempty"`
	Speed          *float64 `json:"speed,omitempty"`
}
