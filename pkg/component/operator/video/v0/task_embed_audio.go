package video

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	ffmpeg "github.com/warmans/ffmpeg-go"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func embedAudio(ctx context.Context, job *base.Job) error {
	var inputStruct embedAudioInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	// Create temporary input video file
	tempInputVideoFile, err := os.CreateTemp("", "temp-input-video-*.mp4")
	if err != nil {
		return fmt.Errorf("creating temp input video file: %w", err)
	}
	defer func() {
		_ = os.Remove(tempInputVideoFile.Name())
	}()

	videoBytes, err := inputStruct.Video.Binary()
	if err != nil {
		return fmt.Errorf("getting video bytes: %w", err)
	}

	if err := os.WriteFile(tempInputVideoFile.Name(), videoBytes.ByteArray(), 0600); err != nil {
		return fmt.Errorf("writing to temp input video file: %w", err)
	}

	// Create temporary input audio file
	tempInputAudioFile, err := os.CreateTemp("", "temp-input-audio-*.mp3")
	if err != nil {
		return fmt.Errorf("creating temp input audio file: %w", err)
	}
	defer func() {
		_ = os.Remove(tempInputAudioFile.Name())
	}()

	audioBytes, err := inputStruct.Audio.Binary()
	if err != nil {
		return fmt.Errorf("getting audio bytes: %w", err)
	}

	if err := os.WriteFile(tempInputAudioFile.Name(), audioBytes.ByteArray(), 0600); err != nil {
		return fmt.Errorf("writing to temp input audio file: %w", err)
	}

	// Embed audio to video and write to a file
	outputVideoFilePath, err := embedAudioToVideo(tempInputVideoFile.Name(), tempInputAudioFile.Name())
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(outputVideoFilePath)
	}()

	// Read the output video file and export to standard output
	outputVideoBytes, err := os.ReadFile(outputVideoFilePath)
	if err != nil {
		return fmt.Errorf("reading output video file: %w", err)
	}

	outputVideoData, err := data.NewVideoFromBytes(outputVideoBytes, data.MP4, fmt.Sprintf("video-%s.mp4", uuid.New().String()), true)
	if err != nil {
		return fmt.Errorf("creating output video data: %w", err)
	}

	outputData := embedAudioOutput{
		Video: outputVideoData,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return fmt.Errorf("writing output data: %w", err)
	}

	return nil
}

func embedAudioToVideo(inputVideoFile string, inputAudioFile string) (string, error) {
	outputFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("video-%s.mp4", uuid.New().String()))

	input := []*ffmpeg.Stream{ffmpeg.Input(inputVideoFile), ffmpeg.Input(inputAudioFile)}

	// https://www.mux.com/articles/merge-audio-and-video-files-with-ffmpeg
	// Workaround for multiple maps https://github.com/u2takey/ffmpeg-go/issues/1#issuecomment-2507904461
	err := ffmpeg.Output(input, outputFilePath, ffmpeg.KwArgs{
		"c:v":   "copy",
		"c:a":   "aac",
		"map_0": "0:v:0",
		"map_1": "1:a:0",
	}).OverWriteOutput().Run()

	if err != nil {
		return "", fmt.Errorf("embedding audio to video: %w", err)
	}

	return outputFilePath, nil
}
