package video

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"

	qt "github.com/frankban/quicktest"
	ffmpeg "github.com/u2takey/ffmpeg-go"

	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func compareFrame(c *qt.C, actual, expected format.Image) {
	// Compare basic properties
	c.Assert(actual.ContentType().String(), qt.Equals, expected.ContentType().String())
	c.Assert(actual.Width().Float64(), qt.Equals, expected.Width().Float64())
	c.Assert(actual.Height().Float64(), qt.Equals, expected.Height().Float64())

	// Get and decode images
	actualBinary, err := actual.Binary()
	c.Assert(err, qt.IsNil)
	expectedBinary, err := expected.Binary()
	c.Assert(err, qt.IsNil)

	actualImg, _, err := image.Decode(bytes.NewReader(actualBinary.ByteArray()))
	c.Assert(err, qt.IsNil)
	expectedImg, _, err := image.Decode(bytes.NewReader(expectedBinary.ByteArray()))
	c.Assert(err, qt.IsNil)

	// Calculate MSE with pixel sampling for faster comparison
	bounds := actualImg.Bounds()
	var mse float64
	step := 3 // Sample every 3rd pixel for faster comparison
	sampledPixels := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			actualColor := actualImg.At(x, y)
			expectedColor := expectedImg.At(x, y)

			ar, ag, ab, aa := actualColor.RGBA()
			er, eg, eb, ea := expectedColor.RGBA()

			mse += float64((ar-er)*(ar-er) + (ag-eg)*(ag-eg) + (ab-eb)*(ab-eb) + (aa-ea)*(aa-ea))
			sampledPixels++
		}
	}
	mse /= float64(sampledPixels * 4) // 4 channels: R, G, B, A

	// Calculate PSNR
	if mse == 0 {
		c.Assert(true, qt.IsTrue, qt.Commentf("Frames are identical"))
	} else {
		psnr := 10 * math.Log10((65535*65535)/mse)
		c.Assert(psnr >= 30, qt.IsTrue, qt.Commentf("PSNR is too low: %f dB", psnr))
	}
}

func compareAudio(c *qt.C, actual, expected format.Audio) {
	// Compare audio properties
	c.Assert(actual.ContentType().String(), qt.Equals, expected.ContentType().String(), qt.Commentf("Content types do not match"))
	c.Assert(actual.SampleRate().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.01)), expected.SampleRate().Float64(), qt.Commentf("Sample rates do not match"))
	c.Assert(actual.Duration().Float64(), qt.CmpEquals(cmpopts.EquateApprox(0, 0.01)), expected.Duration().Float64(), qt.Commentf("Durations do not match"))

	// Get binary data
	actualBinary, err := actual.Binary()
	c.Assert(err, qt.IsNil)
	expectedBinary, err := expected.Binary()
	c.Assert(err, qt.IsNil)

	actualBytes := actualBinary.ByteArray()
	expectedBytes := expectedBinary.ByteArray()

	// Handle different audio lengths by using the minimum length
	minLen := len(actualBytes)
	if len(expectedBytes) < minLen {
		minLen = len(expectedBytes)
	}

	// For very different lengths, we should consider this a significant difference
	if math.Abs(float64(len(actualBytes)-len(expectedBytes)))/float64(minLen) > 0.1 {
		c.Assert(false, qt.IsTrue, qt.Commentf("Audio lengths differ significantly: actual=%d, expected=%d", len(actualBytes), len(expectedBytes)))
		return
	}

	var mse float64
	for i := 0; i < minLen; i++ {
		diff := float64(actualBytes[i]) - float64(expectedBytes[i])
		mse += diff * diff
	}
	mse /= float64(minLen)

	if mse == 0 {
		c.Assert(true, qt.IsTrue, qt.Commentf("Audio signals are identical"))
	} else {
		// For 16-bit audio samples (values range from -32768 to 32767)
		psnr := 10 * math.Log10(32767*32767/mse)
		c.Assert(psnr >= 30, qt.IsTrue, qt.Commentf("PSNR is too low: %f dB", psnr))
	}
}

func compareVideo(c *qt.C, actual, expected format.Video) {
	// Compare basic properties
	c.Assert(actual.ContentType().String(), qt.Equals, expected.ContentType().String())

	tolerance := 0.1 // 100ms tolerance
	actualDuration := actual.Duration().Float64()
	expectedDuration := expected.Duration().Float64()
	c.Assert(math.Abs(actualDuration-expectedDuration) <= tolerance, qt.IsTrue)

	// Extract frames from both videos using ffmpeg
	actualFrames, err := extractFramesWithFFmpeg(actual)
	c.Assert(err, qt.IsNil)
	expectedFrames, err := extractFramesWithFFmpeg(expected)
	c.Assert(err, qt.IsNil)

	// Compare frame counts with some tolerance
	frameDiff := math.Abs(float64(len(actualFrames) - len(expectedFrames)))
	c.Assert(frameDiff/float64(len(expectedFrames)) <= 0.01, qt.IsTrue,
		qt.Commentf("Frame count differs by more than 1%%: got %d, want %d",
			len(actualFrames), len(expectedFrames)))

	// Compare frames using smart sampling for efficiency
	maxFrames := len(actualFrames)
	if len(expectedFrames) < maxFrames {
		maxFrames = len(expectedFrames)
	}

	// For videos with many frames, sample key frames
	framesToCompare := []int{}
	if maxFrames <= 5 {
		// Compare all frames for short videos
		for i := 0; i < maxFrames; i++ {
			framesToCompare = append(framesToCompare, i)
		}
	} else {
		// For longer videos, compare first, last, middle, and a few samples
		framesToCompare = append(framesToCompare, 0)             // first frame
		framesToCompare = append(framesToCompare, maxFrames/4)   // quarter point
		framesToCompare = append(framesToCompare, maxFrames/2)   // middle
		framesToCompare = append(framesToCompare, 3*maxFrames/4) // three-quarter point
		framesToCompare = append(framesToCompare, maxFrames-1)   // last frame
	}

	for _, i := range framesToCompare {
		compareFrame(c, actualFrames[i], expectedFrames[i])
	}
}

func extractFramesWithFFmpeg(video format.Video) ([]format.Image, error) {
	// Use shorter UUID for temp directory
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("frames-%s", uuid.New().String()[:8]))
	defer func() {
		os.RemoveAll(tmpDir) // Ensure cleanup even on error
	}()

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	inputFile := filepath.Join(tmpDir, "input.mp4")
	outputPattern := filepath.Join(tmpDir, "frame-%04d.png")

	binary, err := video.Binary()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(inputFile, binary.ByteArray(), 0600); err != nil {
		return nil, err
	}

	err = ffmpeg.Input(inputFile).
		Output(outputPattern, ffmpeg.KwArgs{
			"vf":      "fps=1,scale=160:120", // Reduced resolution for faster processing
			"pix_fmt": "rgb24",
			"q:v":     "8", // Faster encoding with acceptable quality
		}).
		OverWriteOutput().
		Silent(true).
		Run()
	if err != nil {
		return nil, fmt.Errorf("extracting frames: %w", err)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("reading frames directory: %w", err)
	}

	// Pre-allocate slice for better performance
	frames := make([]format.Image, 0, len(files))
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".png" {
			continue
		}

		framePath := filepath.Join(tmpDir, file.Name())
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
