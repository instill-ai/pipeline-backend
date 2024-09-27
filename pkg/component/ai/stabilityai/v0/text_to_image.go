package stabilityai

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

const (
	successFinishReason     = "SUCCESS"
	textToImagePathTemplate = "/v1/generation/%s/text-to-image"
)

func textToImagePath(engine string) string {
	return fmt.Sprintf(textToImagePathTemplate, engine)
}

type TextToImageInput struct {
	Task               string     `json:"task"`
	Prompts            []string   `json:"prompts"`
	Engine             string     `json:"engine"`
	Weights            *[]float64 `json:"weights,omitempty"`
	Height             *uint32    `json:"height,omitempty"`
	Width              *uint32    `json:"width,omitempty"`
	CfgScale           *float64   `json:"cfg-scale,omitempty"`
	ClipGuidancePreset *string    `json:"clip-guidance-preset,omitempty"`
	Sampler            *string    `json:"sampler,omitempty"`
	Samples            *uint32    `json:"samples,omitempty"`
	Seed               *uint32    `json:"seed,omitempty"`
	Steps              *uint32    `json:"steps,omitempty"`
	StylePreset        *string    `json:"style-preset,omitempty"`
}

type TextToImageOutput struct {
	Images []string `json:"images"`
	Seeds  []uint32 `json:"seeds"`
}

// TextToImageReq represents the request body for text-to-image API
type TextToImageReq struct {
	TextPrompts        []TextPrompt `json:"text_prompts"`
	CFGScale           *float64     `json:"cfg_scale,omitempty"`
	ClipGuidancePreset *string      `json:"clip_guidance_preset,omitempty"`
	Sampler            *string      `json:"sampler,omitempty"`
	Samples            *uint32      `json:"samples,omitempty"`
	Seed               *uint32      `json:"seed,omitempty"`
	Steps              *uint32      `json:"steps,omitempty"`
	StylePreset        *string      `json:"style_preset,omitempty"`
	Height             *uint32      `json:"height,omitempty"`
	Width              *uint32      `json:"width,omitempty"`

	path string
}

// TextPrompt holds a prompt's text and its weight.
type TextPrompt struct {
	Text   string   `json:"text"`
	Weight *float64 `json:"weight"`
}

// Image represents a single image.
type Image struct {
	Base64       string `json:"base64"`
	Seed         uint32 `json:"seed"`
	FinishReason string `json:"finishReason"`
}

// ImageTaskRes represents the response body for text-to-image API.
type ImageTaskRes struct {
	Images []Image `json:"artifacts"`
}

func parseTextToImageReq(from *structpb.Struct) (TextToImageReq, error) {
	// Parse from pb.
	input := TextToImageInput{}
	if err := base.ConvertFromStructpb(from, &input); err != nil {
		return TextToImageReq{}, err
	}

	// Validate input.
	nPrompts := len(input.Prompts)
	if nPrompts <= 0 {
		return TextToImageReq{}, fmt.Errorf("no text prompts given")
	}

	if input.Engine == "" {
		return TextToImageReq{}, fmt.Errorf("no engine selected")
	}

	// Convert to req.
	req := TextToImageReq{
		CFGScale:           input.CfgScale,
		ClipGuidancePreset: input.ClipGuidancePreset,
		Sampler:            input.Sampler,
		Samples:            input.Samples,
		Seed:               input.Seed,
		Steps:              input.Steps,
		StylePreset:        input.StylePreset,
		Height:             input.Height,
		Width:              input.Width,

		path: textToImagePath(input.Engine),
	}

	req.TextPrompts = make([]TextPrompt, 0, nPrompts)
	for index, t := range input.Prompts {
		// If weight isn't provided, set to 1.
		w := 1.0
		if input.Weights != nil && len(*input.Weights) > index {
			w = (*input.Weights)[index]
		}

		req.TextPrompts = append(req.TextPrompts, TextPrompt{
			Text:   t,
			Weight: &w,
		})
	}

	return req, nil
}

func textToImageOutput(from ImageTaskRes) (*structpb.Struct, error) {
	output := TextToImageOutput{
		Images: []string{},
		Seeds:  []uint32{},
	}

	for _, image := range from.Images {
		if image.FinishReason != successFinishReason {
			continue
		}

		output.Images = append(output.Images, fmt.Sprintf("data:image/png;base64,%s", image.Base64))
		output.Seeds = append(output.Seeds, image.Seed)
	}

	return base.ConvertToStructpb(output)
}
