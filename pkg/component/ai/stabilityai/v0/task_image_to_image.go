package stabilityai

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime/multipart"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

const imageToImagePathTemplate = "/v1/generation/%s/image-to-image"

func imageToImagePath(engine string) string {
	return fmt.Sprintf(imageToImagePathTemplate, engine)
}

func (e *execution) handleImageToImage(ctx context.Context, job *base.Job) error {
	input := &taskImageToImageInput{}
	if err := job.Input.ReadData(ctx, input); err != nil {
		return err
	}

	params, err := parseImageToImageReq(input)
	if err != nil {
		return err
	}

	b, contentType, err := params.getBytes()
	if err != nil {
		return err
	}

	resp := ImageTaskRes{}
	req := e.client.R().
		SetResult(&resp).
		SetBody(b).
		SetHeader("Content-Type", contentType)

	if _, err := req.Post(params.path); err != nil {
		return err
	}

	output, err := imageToImageOutput(resp)
	if err != nil {
		return err
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
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

func parseImageToImageReq(input *taskImageToImageInput) (ImageToImageReq, error) {
	// Validate input.
	nPrompts := len(input.Prompts)
	if nPrompts <= 0 {
		return ImageToImageReq{}, fmt.Errorf("no text prompts given")
	}

	if input.Engine == "" {
		return ImageToImageReq{}, fmt.Errorf("no engine selected")
	}

	// Convert to req.
	if input.InitImage == nil {
		return ImageToImageReq{}, fmt.Errorf("no init image provided")
	}
	initImage, err := input.InitImage.DataURI()
	if err != nil {
		return ImageToImageReq{}, fmt.Errorf("failed to get data URI: %w", err)
	}
	req := ImageToImageReq{
		InitImage:          initImage.String(),
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
		// If weight isn't provided, set to 1.
		w := 1.0
		if input.Weights != nil && len(input.Weights) > index {
			w = input.Weights[index]
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

func imageToImageOutput(from ImageTaskRes) (*taskOutput, error) {
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
		img, err := data.NewImageFromBytes(imgBytes, data.PNG, "", true)
		if err != nil {
			return nil, fmt.Errorf("failed to create image from bytes: %w", err)
		}

		output.Images = append(output.Images, img)
		output.Seeds = append(output.Seeds, int(image.Seed))
	}

	return &output, nil
}
