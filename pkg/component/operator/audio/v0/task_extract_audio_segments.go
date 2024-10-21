package audio

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type extractAudioSegmentsInput struct {
	Audio    Audio     `json:"audio"`
	Segments []segment `json:"segments"`
}

type extractAudioSegmentsOutput struct {
	AudioSegments []string `json:"audio-segments"`
}

func extractAudioSegments(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {
	var inputStruct extractAudioSegmentsInput
	if err := base.ConvertFromStructpb(input, &inputStruct); err != nil {
		return nil, fmt.Errorf("converting input to struct: %w", err)
	}

	audioBuf, dec, err := decodeAudio(string(inputStruct.Audio))
	if err != nil {
		return nil, err
	}

	output := extractAudioSegmentsOutput{
		AudioSegments: make([]string, len(inputStruct.Segments)),
	}

	for i, seg := range inputStruct.Segments {
		segment, err := extractSegment(audioBuf, seg)
		if err != nil {
			return nil, err
		}

		encodedSegment, err := encodeSegment(segment, audioBuf.Format, dec)
		if err != nil {
			return nil, err
		}

		output.AudioSegments[i] = "data:audio/wav;base64," + encodedSegment
	}

	return base.ConvertToStructpb(output)
}

func decodeAudio(audioData string) (*audio.IntBuffer, *wav.Decoder, error) {
	buf, err := base64.StdEncoding.DecodeString(base.TrimBase64Mime(audioData))
	if err != nil {
		return nil, nil, fmt.Errorf("decoding audio data: %w", err)
	}

	dec := wav.NewDecoder(bytes.NewReader(buf))
	if !dec.IsValidFile() {
		return nil, nil, fmt.Errorf("invalid WAV file")
	}

	audioBuf := &audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: int(dec.NumChans),
			SampleRate:  int(dec.SampleRate),
		},
		Data:           make([]int, len(buf)),
		SourceBitDepth: int(dec.BitDepth),
	}

	if _, err := dec.PCMBuffer(audioBuf); err != nil {
		return nil, nil, fmt.Errorf("reading audio data: %w", err)
	}

	return audioBuf, dec, nil
}

func extractSegment(audioBuf *audio.IntBuffer, seg segment) ([]int, error) {
	startSample := int(seg.StartTime * float64(audioBuf.Format.SampleRate) * float64(audioBuf.Format.NumChannels))
	endSample := int(seg.EndTime * float64(audioBuf.Format.SampleRate) * float64(audioBuf.Format.NumChannels))

	if startSample < 0 {
		startSample = 0
	}
	if endSample > len(audioBuf.Data) {
		endSample = len(audioBuf.Data)
	}

	return audioBuf.Data[startSample:endSample], nil
}

func encodeSegment(segment []int, format *audio.Format, dec *wav.Decoder) (string, error) {
	// Use a temporary file instead of a buffer
	tempFile, err := os.CreateTemp("", "audio_segment_*.wav")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up the temp file when we're done

	encoder := wav.NewEncoder(tempFile, format.SampleRate, int(dec.BitDepth), format.NumChannels, int(dec.WavAudioFormat))

	segmentBuf := &audio.IntBuffer{
		Format:         format,
		Data:           segment,
		SourceBitDepth: int(dec.BitDepth),
	}

	if err := encoder.Write(segmentBuf); err != nil {
		return "", fmt.Errorf("failed to write segment to buffer: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("failed to close the encoder: %w", err)
	}

	// Read the contents of the temp file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return "", fmt.Errorf("failed to seek to the beginning of temp file: %w", err)
	}

	fileContents, err := io.ReadAll(tempFile)
	if err != nil {
		return "", fmt.Errorf("failed to read temp file: %w", err)
	}

	return base64.StdEncoding.EncodeToString(fileContents), nil
}
