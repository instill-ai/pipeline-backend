package audio

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type segmentData struct {
	StartTime float64 `instill:"start-time"`
	EndTime   float64 `instill:"end-time"`
}

type detectActivityInput struct {
	Audio              format.Audio `instill:"audio"`
	MinSilenceDuration int          `instill:"min-silence-duration,default=100"`
	SpeechPad          int          `instill:"speech-pad,default=30"`
}

type detectActivityOutput struct {
	Segments []segmentData `instill:"segments"`
}

type segmentInput struct {
	Audio    format.Audio  `instill:"audio"`
	Segments []segmentData `instill:"segments"`
}

type segmentOutput struct {
	AudioSegments []format.Audio `instill:"audio-segments"`
}
