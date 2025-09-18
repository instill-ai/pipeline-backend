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

func extractFrames(ctx context.Context, job *base.Job) error {
	var inputStruct extractFramesInput
	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	// Add input validation
	if inputStruct.Interval != 0 && len(inputStruct.Timestamps) > 0 {
		return fmt.Errorf("interval and timestamps cannot be used together")
	}

	if inputStruct.Interval == 0 && len(inputStruct.Timestamps) == 0 {
		return fmt.Errorf("either interval or timestamps must be provided")
	}

	// Create temporary input file
	tempInputFile, err := os.CreateTemp("", "temp-input-*.mp4")
	if err != nil {
		return fmt.Errorf("creating temp input file: %w", err)
	}
	defer os.Remove(tempInputFile.Name())

	videoBytes, err := inputStruct.Video.Binary()
	if err != nil {
		return fmt.Errorf("getting video bytes: %w", err)
	}

	if err := os.WriteFile(tempInputFile.Name(), videoBytes.ByteArray(), 0600); err != nil {
		return fmt.Errorf("writing to temp input file: %w", err)
	}

	var frames []format.Image
	if inputStruct.Interval != 0 {
		frames, err = extractFramesAtInterval(tempInputFile.Name(), inputStruct.Interval)
	} else {
		frames, err = extractFramesAtTimestamps(tempInputFile.Name(), inputStruct.Timestamps)
	}

	if err != nil {
		return err
	}

	outputData := extractFramesOutput{
		Frames: frames,
	}

	if err := job.Output.WriteData(ctx, outputData); err != nil {
		return fmt.Errorf("writing output data: %w", err)
	}

	return nil
}

func extractFramesAtInterval(inputFile string, interval float64) ([]format.Image, error) {
	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("frames-%s", uuid.New().String()))
	defer os.RemoveAll(outputDir)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	outputPattern := filepath.Join(outputDir, "frame-%04d.png")

	// Protect ffmpeg library calls with mutex to prevent data races
	// in the library's internal global state initialization
	ffmpegMutex.Lock()
	err := ffmpeg.Input(inputFile).
		Output(outputPattern, ffmpeg.KwArgs{
			"vf": fmt.Sprintf("select='not(mod(t,%f))',setpts=N/FRAME_RATE/TB", interval),
		}).
		OverWriteOutput().
		Run()
	ffmpegMutex.Unlock()

	if err != nil {
		return nil, fmt.Errorf("extracting frames from video: %w", err)
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, fmt.Errorf("reading frames directory: %w", err)
	}

	var frames []format.Image
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".png" {
			continue
		}

		framePath := filepath.Join(outputDir, file.Name())
		frameBytes, err := os.ReadFile(framePath)
		if err != nil {
			return nil, fmt.Errorf("reading frame file: %w", err)
		}

		frame, err := data.NewImageFromBytes(frameBytes, data.PNG, file.Name(), true)
		if err != nil {
			return nil, fmt.Errorf("creating image data: %w", err)
		}
		frames = append(frames, frame)
	}

	return frames, nil
}

func extractFramesAtTimestamps(inputFile string, timestamps []float64) ([]format.Image, error) {
	outputDir := filepath.Join(os.TempDir(), fmt.Sprintf("frames-%s", uuid.New().String()))
	defer os.RemoveAll(outputDir)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	var frames []format.Image
	for i, timestamp := range timestamps {
		outputFile := filepath.Join(outputDir, fmt.Sprintf("frame-%04d.png", i))

		// Protect ffmpeg library calls with mutex to prevent data races
		// in the library's internal global state initialization
		ffmpegMutex.Lock()
		err := ffmpeg.Input(inputFile, ffmpeg.KwArgs{"ss": fmt.Sprintf("%f", timestamp)}).
			Output(outputFile, ffmpeg.KwArgs{
				"vframes": 1,
			}).
			OverWriteOutput().
			Run()
		ffmpegMutex.Unlock()

		if err != nil {
			return nil, fmt.Errorf("extracting frame at timestamp %f: %w", timestamp, err)
		}

		frameBytes, err := os.ReadFile(outputFile)
		if err != nil {
			return nil, fmt.Errorf("reading frame file: %w", err)
		}

		frame, err := data.NewImageFromBytes(frameBytes, data.PNG, fmt.Sprintf("frame-%04d.png", i), true)
		if err != nil {
			return nil, fmt.Errorf("creating image data: %w", err)
		}
		frames = append(frames, frame)
	}

	return frames, nil
}
