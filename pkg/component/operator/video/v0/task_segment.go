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
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func segment(ctx context.Context, job *base.Job) error {
	var inputStruct segmentInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	// Handle the case when segments are not provided
	if len(inputStruct.Segments) == 0 {
		outputData := segmentOutput{
			VideoSegments: []format.Video{inputStruct.Video},
		}
		if err := job.Output.WriteData(ctx, outputData); err != nil {
			return fmt.Errorf("writing output data: %w", err)
		}
		return nil
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

	var videoSegments []format.Video
	for i, seg := range inputStruct.Segments {
		segmentFile, err := extractVideoSegment(tempInputFile.Name(), seg)
		if err != nil {
			return fmt.Errorf("extracting video segment %d: %w", i, err)
		}
		defer os.Remove(segmentFile)

		segmentBytes, err := os.ReadFile(segmentFile)
		if err != nil {
			return fmt.Errorf("reading segment file %d: %w", i, err)
		}

		videoData, err := data.NewVideoFromBytes(segmentBytes, data.MP4, fmt.Sprintf("segment-%d.mp4", i), true)
		if err != nil {
			return fmt.Errorf("creating video data for segment %d: %w", i, err)
		}

		videoSegments = append(videoSegments, videoData)
	}

	outputData := segmentOutput{
		VideoSegments: videoSegments,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return fmt.Errorf("writing output data: %w", err)
	}

	return nil
}

func extractVideoSegment(inputFile string, seg *segmentData) (string, error) {
	outputFile := filepath.Join(os.TempDir(), fmt.Sprintf("segment-%s.mp4", uuid.New().String()))

	// Protect ffmpeg library calls with mutex to prevent data races
	// in the library's internal global state initialization
	ffmpegMutex.Lock()
	err := ffmpeg.Input(inputFile, ffmpeg.KwArgs{
		"ss": seg.StartTime,
		"to": seg.EndTime,
	}).
		Output(outputFile, ffmpeg.KwArgs{
			"c": "copy",
		}).
		OverWriteOutput().
		Run()
	ffmpegMutex.Unlock()

	if err != nil {
		return "", fmt.Errorf("extracting video segment: %w", err)
	}

	return outputFile, nil
}
