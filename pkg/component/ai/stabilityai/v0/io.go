package stabilityai

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type taskTextToImageInput struct {
	Prompts            []string  `instill:"prompts"`
	Engine             string    `instill:"engine"`
	CfgScale           *float64  `instill:"cfg-scale"`
	ClipGuidancePreset *string   `instill:"clip-guidance-preset"`
	Height             *uint32   `instill:"height"`
	Sampler            *string   `instill:"sampler"`
	Samples            *uint32   `instill:"samples"`
	Seed               *uint32   `instill:"seed"`
	Steps              *uint32   `instill:"steps"`
	StylePreset        *string   `instill:"style-preset"`
	Weights            []float64 `instill:"weights"`
	Width              *uint32   `instill:"width"`
}

type taskOutput struct {
	Images []format.Image `instill:"images"`
	Seeds  []int          `instill:"seeds"`
}

type taskImageToImageInput struct {
	Prompts            []string     `instill:"prompts"`
	Engine             string       `instill:"engine"`
	CfgScale           *float64     `instill:"cfg-scale"`
	ClipGuidancePreset *string      `instill:"clip-guidance-preset"`
	ImageStrength      *float64     `instill:"image-strength"`
	InitImage          format.Image `instill:"init-image"`
	InitImageMode      *string      `instill:"init-image-mode"`
	Sampler            *string      `instill:"sampler"`
	Samples            *uint32      `instill:"samples"`
	Seed               *uint32      `instill:"seed"`
	StepScheduleEnd    *float64     `instill:"step-schedule-end"`
	StepScheduleStart  *float64     `instill:"step-schedule-start"`
	Steps              *uint32      `instill:"steps"`
	StylePreset        *string      `instill:"style-preset"`
	Weights            []float64    `instill:"weights"`
}
