package openai

const (
	imgGenerationPath = "/v1/images/generations"
)

type ImagesGenerationInput struct {
	Prompt  string  `json:"prompt"`
	Model   string  `json:"model"`
	N       *int    `json:"n,omitempty"`
	Quality *string `json:"quality,omitempty"`
	Size    *string `json:"size,omitempty"`
	Style   *string `json:"style,omitempty"`
}

type ImageGenerationsOutputResult struct {
	Image         string `json:"image"`
	RevisedPrompt string `json:"revised-prompt"`
}
type ImageGenerationsOutput struct {
	Results []ImageGenerationsOutputResult `json:"results"`
}

type ImageGenerationsReq struct {
	Prompt         string  `json:"prompt"`
	Model          string  `json:"model"`
	N              *int    `json:"n,omitempty"`
	Quality        *string `json:"quality,omitempty"`
	Size           *string `json:"size,omitempty"`
	Style          *string `json:"style,omitempty"`
	ResponseFormat string  `json:"response_format"`
}

type ImageGenerationsRespData struct {
	Image         string `json:"b64_json"`
	RevisedPrompt string `json:"revised_prompt"`
}
type ImageGenerationsResp struct {
	Data []ImageGenerationsRespData `json:"data"`
}
