package video

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	ffmpeg "github.com/u2takey/ffmpeg-go"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func extractAudio(ctx context.Context, job *base.Job) error {
	var inputStruct extractAudioInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	// Create temporary input file
	tempInputFile, err := os.CreateTemp("", "temp-input-*.mp4")
	if err != nil {
		return fmt.Errorf("creating temp input file: %w", err)
	}
	defer func() {
		_ = os.Remove(tempInputFile.Name())
	}()

	videoBytes, err := inputStruct.Video.Binary()
	if err != nil {
		return fmt.Errorf("getting video bytes: %w", err)
	}

	if err := os.WriteFile(tempInputFile.Name(), videoBytes.ByteArray(), 0600); err != nil {
		return fmt.Errorf("writing to temp input file: %w", err)
	}

	// Extract audio
	audioFilePath, err := extractAudioFromVideo(tempInputFile.Name())
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(audioFilePath)
	}()

	// Read the audio file
	audioBytes, err := os.ReadFile(audioFilePath)
	if err != nil {
		return fmt.Errorf("reading audio file: %w", err)
	}

	audioData, err := data.NewAudioFromBytes(audioBytes, "audio/ogg", fmt.Sprintf("audio-%s.ogg", uuid.New().String()))
	if err != nil {
		return fmt.Errorf("creating audio data: %w", err)
	}

	outputData := extractAudioOutput{
		Audio: audioData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return fmt.Errorf("writing output data: %w", err)
	}

	return nil
}

func extractAudioFromVideo(inputFile string) (string, error) {
	outputFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("audio-%s.ogg", uuid.New().String()))

	err := ffmpeg.Input(inputFile).
		Output(outputFilePath, ffmpeg.KwArgs{
			"vn":  "",
			"q:a": "0",
		}).
		OverWriteOutput().
		Run()

	if err != nil {
		return "", fmt.Errorf("extracting audio from video: %w", err)
	}

	return outputFilePath, nil
}
