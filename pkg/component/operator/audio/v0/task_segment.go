package audio

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func segment(ctx context.Context, job *base.Job) error {

	var input segmentInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return err
	}

	audioBuf, dec, err := decodeAudioWAV(input.Audio)
	if err != nil {
		return err
	}

	output := segmentOutput{
		AudioSegments: make([]format.Audio, len(input.Segments)),
	}

	for i, seg := range input.Segments {
		seg, err := extractSegment(audioBuf, seg)
		if err != nil {
			return err
		}
		encSeg, err := encodeSegment(seg, audioBuf.Format, dec)
		if err != nil {
			return err
		}
		ad, err := data.NewAudioFromBytes(encSeg, "audio/wav", fmt.Sprintf("audio-segment-%d.wav", i))
		if err != nil {
			return err
		}
		output.AudioSegments[i] = ad
	}

	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}

	return nil

}

func extractSegment(audioBuf *audio.IntBuffer, seg segmentData) ([]int, error) {
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

func encodeSegment(segment []int, format *audio.Format, dec *wav.Decoder) ([]byte, error) {
	// Use a temporary file instead of a buffer
	tempFile, err := os.CreateTemp("", "audio_segment_*.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up the temp file when we're done

	encoder := wav.NewEncoder(tempFile, format.SampleRate, int(dec.BitDepth), format.NumChannels, int(dec.WavAudioFormat))

	segmentBuf := &audio.IntBuffer{
		Format:         format,
		Data:           segment,
		SourceBitDepth: int(dec.BitDepth),
	}

	if err := encoder.Write(segmentBuf); err != nil {
		return nil, fmt.Errorf("failed to write segment to buffer: %w", err)
	}

	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to close the encoder: %w", err)
	}

	// Read the contents of the temp file
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to the beginning of temp file: %w", err)
	}

	fileContents, err := io.ReadAll(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read temp file: %w", err)
	}

	return fileContents, nil
}
