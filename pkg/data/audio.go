package data

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type audioData struct {
	fileData

	duration   time.Duration
	sampleRate int
}

func (audioData) IsValue() {}

const (
	MP3  = "audio/mpeg"
	WAV  = "audio/wav"
	AAC  = "audio/aac"
	OGG  = "audio/ogg"
	FLAC = "audio/flac"
	M4A  = "audio/mp4"
	WMA  = "audio/x-ms-wma"
	AIFF = "audio/aiff"
)

var audioGetter = map[string]func(*audioData) (format.Value, error){
	"duration":    func(a *audioData) (format.Value, error) { return a.Duration(), nil },
	"sample-rate": func(a *audioData) (format.Value, error) { return a.SampleRate(), nil },
	"mp3":         func(a *audioData) (format.Value, error) { return a.Convert(MP3) },
	"wav":         func(a *audioData) (format.Value, error) { return a.Convert(WAV) },
	"aac":         func(a *audioData) (format.Value, error) { return a.Convert(AAC) },
	"ogg":         func(a *audioData) (format.Value, error) { return a.Convert(OGG) },
	"flac":        func(a *audioData) (format.Value, error) { return a.Convert(FLAC) },
	"m4a":         func(a *audioData) (format.Value, error) { return a.Convert(M4A) },
	"wma":         func(a *audioData) (format.Value, error) { return a.Convert(WMA) },
	"aiff":        func(a *audioData) (format.Value, error) { return a.Convert(AIFF) },
}

func NewAudioFromBytes(b []byte, contentType, filename string) (*audioData, error) {
	return createAudioData(b, contentType, filename)
}

func NewAudioFromURL(url string) (*audioData, error) {
	b, contentType, filename, err := convertURLToBytes(url)
	if err != nil {
		return nil, err
	}
	return createAudioData(b, contentType, filename)
}

func createAudioData(b []byte, contentType, filename string) (*audioData, error) {
	if contentType != OGG {
		var err error
		b, err = convertAudio(b, contentType, OGG)
		if err != nil {
			return nil, err
		}
		contentType = OGG
	}

	f, err := NewFileFromBytes(b, contentType, filename)
	if err != nil {
		return nil, err
	}

	return newAudio(f)
}

func newAudio(f *fileData) (*audioData, error) {
	duration, sampleRate, err := getAudioProperties(f.raw, f.contentType)
	if err != nil {
		return nil, err
	}
	a := &audioData{
		fileData:   *f,
		duration:   duration,
		sampleRate: sampleRate,
	}

	return a, nil
}

func getAudioProperties(b []byte, contentType string) (duration time.Duration, sampleRate int, err error) {
	tempDir, err := os.MkdirTemp("", "audio_properties_*")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "temp_audio"+getExtensionFromMIME(contentType))
	if err := os.WriteFile(tempFile, b, 0644); err != nil {
		return 0, 0, fmt.Errorf("failed to write temp file: %w", err)
	}

	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		tempFile)

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probeData struct {
		Streams []struct {
			CodecType  string `json:"codec_type"`
			SampleRate string `json:"sample_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &probeData); err != nil {
		return 0, 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	durationFloat, err := strconv.ParseFloat(probeData.Format.Duration, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse duration: %w", err)
	}
	duration = time.Duration(durationFloat * float64(time.Second))

	var audioStream *struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
	}

	for i := range probeData.Streams {
		if probeData.Streams[i].CodecType == "audio" {
			audioStream = &probeData.Streams[i]
			break
		}
	}

	if audioStream == nil {
		return 0, 0, fmt.Errorf("no audio stream found")
	}

	sampleRate, err = strconv.Atoi(audioStream.SampleRate)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse sample rate: %w", err)
	}

	return
}

func (a *audioData) Duration() format.Number {
	return NewNumberFromFloat(a.duration.Seconds())
}

func (a *audioData) SampleRate() format.Number {
	return NewNumberFromInteger(a.sampleRate)
}

func (a *audioData) Convert(contentType string) (format.Audio, error) {
	b, err := convertAudio(a.raw, a.contentType, contentType)
	if err != nil {
		return nil, fmt.Errorf("can not convert audio from %s to %s: %w", a.contentType, contentType, err)
	}
	f, err := NewFileFromBytes(b, contentType, "")
	if err != nil {
		return nil, fmt.Errorf("can not create new audio file after conversion: %w", err)
	}
	return newAudio(f)
}

func (a *audioData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return a, nil
	}

	firstSeg, remainingPath, err := p.TrimFirst()
	if err != nil {
		return nil, err
	}

	if firstSeg.SegmentType != path.AttributeSegment {
		return nil, fmt.Errorf("path not found: %s", p)
	}

	getter, exists := audioGetter[firstSeg.Attribute]
	if !exists {
		return a.fileData.Get(p)
	}

	result, err := getter(a)
	if err != nil {
		return nil, err
	}

	if remainingPath.IsEmpty() {
		return result, nil
	}

	return result.Get(remainingPath)
}
