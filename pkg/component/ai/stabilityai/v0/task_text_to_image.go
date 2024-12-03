package stabilityai

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

const (
	successFinishReason     = "SUCCESS"
	textToImagePathTemplate = "/v1/generation/%s/text-to-image"
)

func textToImagePath(engine string) string {
	return fmt.Sprintf(textToImagePathTemplate, engine)
}

func (e *execution) handleTextToImage(ctx context.Context, job *base.Job) error {
	input := &taskTextToImageInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}
	params, err := parseTextToImageReq(input)
	if err != nil {
		return err
	}

	resp := ImageTaskRes{}
	req := e.client.R().SetResult(&resp).SetBody(params)

	if _, err := req.Post(params.path); err != nil {
		return err
	}

	output, err := textToImageOutput(resp)
	if err != nil {
		return err
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
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

func parseTextToImageReq(input *taskTextToImageInput) (TextToImageReq, error) {

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
		if input.Weights != nil && len(input.Weights) > index {
			w = (input.Weights)[index]
		}

		req.TextPrompts = append(req.TextPrompts, TextPrompt{
			Text:   t,
			Weight: &w,
		})
	}

	return req, nil
}

func textToImageOutput(from ImageTaskRes) (*taskOutput, error) {
	output := taskOutput{
		Images: []format.Image{},
		Seeds:  []int{},
	}

	for _, image := range from.Images {
		if image.FinishReason != successFinishReason {
			continue
		}
		imgBytes, err := base64.StdEncoding.DecodeString(image.Base64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 image: %w", err)
		}
		img, err := data.NewImageFromBytes(imgBytes, "image/png", "")
		if err != nil {
			return nil, fmt.Errorf("failed to create image from bytes: %w", err)
		}

		output.Images = append(output.Images, img)
		output.Seeds = append(output.Seeds, int(image.Seed))
	}

	return &output, nil
}
