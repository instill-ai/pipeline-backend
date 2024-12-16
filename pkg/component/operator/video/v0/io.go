package video

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type segmentData struct {
	StartTime float64 `instill:"start-time" json:"start-time"`
	EndTime   float64 `instill:"end-time" json:"end-time"`
}

type segmentInput struct {
	Video    format.Video   `instill:"video"`
	Segments []*segmentData `instill:"segments"`
}

type segmentOutput struct {
	VideoSegments []format.Video `instill:"video-segments"`
}

type subsampleInput struct {
	Video        format.Video `instill:"video"`
	VideoBitrate float64      `instill:"video-bitrate"`
	AudioBitrate float64      `instill:"audio-bitrate"`
	FPS          float64      `instill:"fps"`
	Width        int          `instill:"width"`
	Height       int          `instill:"height"`
}

type subsampleOutput struct {
	Video format.Video `instill:"video"`
}

type extractAudioInput struct {
	Video format.Video `instill:"video"`
}

type extractAudioOutput struct {
	Audio format.Audio `instill:"audio"`
}

type extractFramesInput struct {
	Video      format.Video `instill:"video"`
	Interval   float64      `instill:"interval"`
	Timestamps []float64    `instill:"timestamps"`
}

type extractFramesOutput struct {
	Frames []format.Image `instill:"frames"`
}

type embedAudioInput struct {
	Video format.Video `instill:"video"`
	Audio format.Audio `instill:"audio"`
}

type embedAudioOutput struct {
	Video format.Video `instill:"video"`
}
