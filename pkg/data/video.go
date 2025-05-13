package data

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

type videoData struct {
	fileData
	width     int
	height    int
	duration  time.Duration
	frameRate float64
}

func (videoData) IsValue() {}

const (
	MP4  = "video/mp4"
	AVI  = "video/x-msvideo"
	MOV  = "video/quicktime"
	WEBM = "video/webm"
	MKV  = "video/x-matroska"
	FLV  = "video/x-flv"
	WMV  = "video/x-ms-wmv"
	MPEG = "video/mpeg"
)

var videoGetters = map[string]func(*videoData) (format.Value, error){
	"duration":   func(v *videoData) (format.Value, error) { return v.Duration(), nil },
	"width":      func(v *videoData) (format.Value, error) { return v.Width(), nil },
	"height":     func(v *videoData) (format.Value, error) { return v.Height(), nil },
	"frame-rate": func(v *videoData) (format.Value, error) { return v.FrameRate(), nil },
	"mp4":        func(v *videoData) (format.Value, error) { return v.Convert(MP4) },
	"avi":        func(v *videoData) (format.Value, error) { return v.Convert(AVI) },
	"mov":        func(v *videoData) (format.Value, error) { return v.Convert(MOV) },
	"wmv":        func(v *videoData) (format.Value, error) { return v.Convert(WMV) },
	"flv":        func(v *videoData) (format.Value, error) { return v.Convert(FLV) },
	"webm":       func(v *videoData) (format.Value, error) { return v.Convert(WEBM) },
}

func NewVideoFromBytes(b []byte, contentType, filename string) (video *videoData, err error) {
	return createVideoData(b, contentType, filename)
}

func NewVideoFromURL(ctx context.Context, binaryFetcher external.BinaryFetcher, url string) (video *videoData, err error) {
	b, contentType, filename, err := binaryFetcher.FetchFromURL(ctx, url)
	if err != nil {
		return nil, err
	}
	return createVideoData(b, contentType, filename)
}

func createVideoData(b []byte, contentType, filename string) (*videoData, error) {
	if contentType != MP4 {
		var err error
		b, err = convertVideo(b, contentType, MP4)
		if err != nil {
			return nil, fmt.Errorf("failed to convert video to MP4: %w", err)
		}
		contentType = MP4
	}
	f, err := NewFileFromBytes(b, contentType, filename)
	if err != nil {
		return nil, err
	}
	return newVideo(f)
}

func newVideo(f *fileData) (*videoData, error) {
	width, height, duration, frameRate, err := getVideoProperties(f.raw, f.contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to get video properties: %w", err)
	}

	v := &videoData{
		fileData:  *f,
		width:     width,
		height:    height,
		duration:  duration,
		frameRate: frameRate,
	}

	return v, nil
}

func getVideoProperties(raw []byte, contentType string) (width, height int, duration time.Duration, frameRate float64, err error) {
	tempDir, err := os.MkdirTemp("", "video_properties_*")
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "video"+getExtensionFromMIME(contentType))
	if err := os.WriteFile(tempFile, raw, 0644); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to write to temp file: %w", err)
	}

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		tempFile)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeData struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
			FrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var videoStream *struct {
		CodecType string `json:"codec_type"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		FrameRate string `json:"r_frame_rate"`
	}

	for i := range probeData.Streams {
		if probeData.Streams[i].CodecType == "video" {
			videoStream = &probeData.Streams[i]
			break
		}
	}

	if videoStream == nil {
		return 0, 0, 0, 0, fmt.Errorf("no video stream found")
	}

	width = videoStream.Width
	height = videoStream.Height

	durationFloat, err := strconv.ParseFloat(probeData.Format.Duration, 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	duration = time.Duration(durationFloat * float64(time.Second))

	frameRateFraction := strings.Split(videoStream.FrameRate, "/")
	if len(frameRateFraction) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid frame rate format")
	}
	numerator, err := strconv.ParseFloat(frameRateFraction[0], 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse frame rate numerator: %w", err)
	}
	denominator, err := strconv.ParseFloat(frameRateFraction[1], 64)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to parse frame rate denominator: %w", err)
	}
	frameRate = numerator / denominator

	return width, height, duration, frameRate, nil
}

func (vid *videoData) Convert(contentType string) (format.Video, error) {
	b, err := convertVideo(vid.raw, vid.contentType, contentType)
	if err != nil {
		return nil, fmt.Errorf("can not convert data from %s to %s: %w", vid.contentType, contentType, err)
	}
	f, err := NewFileFromBytes(b, contentType, "")
	if err != nil {
		return nil, fmt.Errorf("can not create new video file after conversion: %w", err)
	}
	return newVideo(f)
}

func (vid *videoData) Width() format.Number {
	return NewNumberFromInteger(vid.width)
}

func (vid *videoData) Height() format.Number {
	return NewNumberFromInteger(vid.height)
}

func (vid *videoData) Duration() format.Number {
	return NewNumberFromFloat(vid.duration.Seconds())
}

func (vid *videoData) FrameRate() format.Number {
	return NewNumberFromFloat(float64(vid.frameRate))
}

func (vid *videoData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return vid, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}

	if firstSeg.SegmentType != path.AttributeSegment {
		return nil, fmt.Errorf("path not found: %s", p)
	}

	getter, exists := videoGetters[firstSeg.Attribute]
	if !exists {
		return vid.fileData.Get(p)
	}

	result, err := getter(vid)
	if err != nil {
		return nil, err
	}

	if remainingPath.IsEmpty() {
		return result, nil
	}

	return result.Get(remainingPath)
}

// videoData has unexported fields, which cannot be accessed by the regular
// encoder / decoder. A custom encode/decode method pair is defined to send and
// receive the type with the gob package.

// encVideoData is redundant with videoData but allows us not to modify the
// format.Image interface signature.
type encVideoData struct {
	encFileData
	Width     int
	Height    int
	Duration  time.Duration
	FrameRate float64
}

func (vid *videoData) GobEncode() ([]byte, error) {
	return json.Marshal(encVideoData{
		encFileData: vid.asEncodedStruct(),
		Width:       vid.width,
		Height:      vid.height,
		Duration:    vid.duration,
		FrameRate:   vid.frameRate,
	})
}

func (vid *videoData) GobDecode(b []byte) error {
	var ev encVideoData
	if err := json.Unmarshal(b, &ev); err != nil {
		return err
	}

	vid.fileData = ev.asFileData()
	vid.width = ev.Width
	vid.height = ev.Height
	vid.duration = ev.Duration
	vid.frameRate = ev.FrameRate

	return nil
}
