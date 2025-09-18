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

func subsample(ctx context.Context, job *base.Job) error {
	var inputStruct subsampleInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	if inputStruct.VideoBitrate < 0 || inputStruct.AudioBitrate < 0 ||
		inputStruct.FPS < 0 || inputStruct.Width < 0 || inputStruct.Height < 0 {
		return fmt.Errorf("invalid parameter: bitrates, FPS, width, and height must be non-negative")
	}

	hasModification := inputStruct.VideoBitrate > 0 || inputStruct.AudioBitrate > 0 ||
		inputStruct.FPS > 0 || inputStruct.Width > 0 || inputStruct.Height > 0

	if !hasModification {
		return fmt.Errorf("at least one of video-bitrate, audio-bitrate, fps, width or height must be provided")
	}

	videoBytes, err := inputStruct.Video.Binary()
	if err != nil {
		return fmt.Errorf("getting video bytes: %w", err)
	}

	tempInputFile, err := os.CreateTemp("", "temp-input-*.mp4")
	if err != nil {
		return fmt.Errorf("creating temp input file: %w", err)
	}
	defer os.Remove(tempInputFile.Name())

	if err := os.WriteFile(tempInputFile.Name(), videoBytes.ByteArray(), 0600); err != nil {
		return fmt.Errorf("writing to temp input file: %w", err)
	}

	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("subsampled-%s.mp4", uuid.New().String()))
	defer os.Remove(outputFile)

	ffmpegArgs := ffmpeg.KwArgs{}

	if inputStruct.FPS != 0 {
		ffmpegArgs["r"] = inputStruct.FPS
	}
	if inputStruct.VideoBitrate != 0 {
		ffmpegArgs["b:v"] = fmt.Sprintf("%dk", int(inputStruct.VideoBitrate))
	}
	if inputStruct.AudioBitrate != 0 {
		ffmpegArgs["b:a"] = fmt.Sprintf("%dk", int(inputStruct.AudioBitrate))
	}
	if inputStruct.Width != 0 || inputStruct.Height != 0 {
		if inputStruct.Width != 0 && inputStruct.Height != 0 {
			ffmpegArgs["s"] = fmt.Sprintf("%dx%d", inputStruct.Width, inputStruct.Height)
		} else if inputStruct.Width != 0 {
			ffmpegArgs["filter:v"] = fmt.Sprintf("scale=%d:-2", inputStruct.Width)
		} else if inputStruct.Height != 0 {
			ffmpegArgs["filter:v"] = fmt.Sprintf("scale=-2:%d", inputStruct.Height)
		}
	}

	// Protect ffmpeg library calls with mutex to prevent data races
	// in the library's internal global state initialization
	ffmpegMutex.Lock()
	err = ffmpeg.Input(tempInputFile.Name()).
		Output(outputFile, ffmpegArgs).
		OverWriteOutput().
		Run()
	ffmpegMutex.Unlock()

	if err != nil {
		return fmt.Errorf("subsampling video: %w", err)
	}

	outputBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return fmt.Errorf("reading output file: %w", err)
	}

	videoData, err := data.NewVideoFromBytes(outputBytes, data.MP4, "subsampled.mp4", true)
	if err != nil {
		return fmt.Errorf("creating video data: %w", err)
	}

	outputData := subsampleOutput{
		Video: videoData,
	}
	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return fmt.Errorf("writing output data: %w", err)
	}

	return nil
}
