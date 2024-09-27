package stabilityai

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

const imageToImagePathTemplate = "/v1/generation/%s/image-to-image"

func imageToImagePath(engine string) string {
	return fmt.Sprintf(imageToImagePathTemplate, engine)
}

type ImageToImageInput struct {
	Task               string     `json:"task"`
	Engine             string     `json:"engine"`
	Prompts            []string   `json:"prompts"`
	InitImage          string     `json:"init-image"`
	Weights            *[]float64 `json:"weights,omitempty"`
	InitImageMode      *string    `json:"init-image-mode,omitempty"`
	ImageStrength      *float64   `json:"image-strength,omitempty"`
	StepScheduleStart  *float64   `json:"step-schedule-start,omitempty"`
	StepScheduleEnd    *float64   `json:"step-schedule-end,omitempty"`
	CfgScale           *float64   `json:"cfg-scale,omitempty"`
	ClipGuidancePreset *string    `json:"clip-guidance-preset,omitempty"`
	Sampler            *string    `json:"sampler,omitempty"`
	Samples            *uint32    `json:"samples,omitempty"`
	Seed               *uint32    `json:"seed,omitempty"`
	Steps              *uint32    `json:"steps,omitempty"`
	StylePreset        *string    `json:"style-preset,omitempty"`
}

type ImageToImageOutput struct {
	Images []string `json:"images"`
	Seeds  []uint32 `json:"seeds"`
}

// ImageToImageReq represents the request body for image-to-image API
type ImageToImageReq struct {
	TextPrompts        []TextPrompt `json:"text_prompts"`
	InitImage          string       `json:"init_image"`
	CFGScale           *float64     `json:"cfg_scale,omitempty"`
	ClipGuidancePreset *string      `json:"clip_guidance_preset,omitempty"`
	Sampler            *string      `json:"sampler,omitempty"`
	Samples            *uint32      `json:"samples,omitempty"`
	Seed               *uint32      `json:"seed,omitempty"`
	Steps              *uint32      `json:"steps,omitempty"`
	StylePreset        *string      `json:"style_preset,omitempty"`
	InitImageMode      *string      `json:"init_image_mode,omitempty"`
	ImageStrength      *float64     `json:"image_strength,omitempty"`
	StepScheduleStart  *float64     `json:"step_schedule_start,omitempty"`
	StepScheduleEnd    *float64     `json:"step_schedule_end,omitempty"`

	path string
}

func parseImageToImageReq(from *structpb.Struct) (ImageToImageReq, error) {
	// Parse from pb.
	input := ImageToImageInput{}
	if err := base.ConvertFromStructpb(from, &input); err != nil {
		return ImageToImageReq{}, err
	}

	// Validate input.
	nPrompts := len(input.Prompts)
	if nPrompts <= 0 {
		return ImageToImageReq{}, fmt.Errorf("no text prompts given")
	}

	if input.Engine == "" {
		return ImageToImageReq{}, fmt.Errorf("no engine selected")
	}

	// Convert to req.
	req := ImageToImageReq{
		InitImage:          input.InitImage,
		InitImageMode:      input.InitImageMode,
		ImageStrength:      input.ImageStrength,
		StepScheduleStart:  input.StepScheduleStart,
		StepScheduleEnd:    input.StepScheduleEnd,
		CFGScale:           input.CfgScale,
		ClipGuidancePreset: input.ClipGuidancePreset,
		Sampler:            input.Sampler,
		Samples:            input.Samples,
		Seed:               input.Seed,
		Steps:              input.Steps,
		StylePreset:        input.StylePreset,

		path: imageToImagePath(input.Engine),
	}

	req.TextPrompts = make([]TextPrompt, 0, nPrompts)
	for index, t := range input.Prompts {
		var w float64
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

func (req ImageToImageReq) getBytes() (b *bytes.Reader, contentType string, err error) {
	data := &bytes.Buffer{}
	initImage, err := util.DecodeBase64(req.InitImage)
	if err != nil {
		return nil, "", err
	}
	writer := multipart.NewWriter(data)
	err = util.WriteFile(writer, "init_image", initImage)
	if err != nil {
		return nil, "", err
	}
	if req.CFGScale != nil {
		util.WriteField(writer, "cfg_scale", fmt.Sprintf("%f", *req.CFGScale))
	}
	if req.ClipGuidancePreset != nil {
		util.WriteField(writer, "clip_guidance_preset", *req.ClipGuidancePreset)
	}
	if req.Sampler != nil {
		util.WriteField(writer, "sampler", *req.Sampler)
	}
	if req.Seed != nil {
		util.WriteField(writer, "seed", fmt.Sprintf("%d", *req.Seed))
	}
	if req.StylePreset != nil {
		util.WriteField(writer, "style_preset", *req.StylePreset)
	}
	if req.InitImageMode != nil {
		util.WriteField(writer, "init_image_mode", *req.InitImageMode)
	}
	if req.ImageStrength != nil {
		util.WriteField(writer, "image_strength", fmt.Sprintf("%f", *req.ImageStrength))
	}
	if req.Samples != nil {
		util.WriteField(writer, "samples", fmt.Sprintf("%d", *req.Samples))
	}
	if req.Steps != nil {
		util.WriteField(writer, "steps", fmt.Sprintf("%d", *req.Steps))
	}

	i := 0
	for _, t := range req.TextPrompts {
		if t.Text == "" {
			continue
		}
		util.WriteField(writer, fmt.Sprintf("text_prompts[%d][text]", i), t.Text)
		if t.Weight != nil {
			util.WriteField(writer, fmt.Sprintf("text_prompts[%d][weight]", i), fmt.Sprintf("%f", *t.Weight))
		}
		i++
	}
	writer.Close()
	return bytes.NewReader(data.Bytes()), writer.FormDataContentType(), nil
}

func imageToImageOutput(from ImageTaskRes) (*structpb.Struct, error) {
	output := ImageToImageOutput{
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
