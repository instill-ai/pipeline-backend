package data

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

func convertImage(raw []byte, sourceContentType, targetContentType string) ([]byte, error) {
	// Define supported formats and their corresponding decode/encode functions
	formats := map[string]struct {
		decode func(io.Reader) (image.Image, error)
		encode func(io.Writer, image.Image) error
	}{
		PNG:  {png.Decode, png.Encode},
		JPEG: {jpeg.Decode, func(w io.Writer, m image.Image) error { return jpeg.Encode(w, m, nil) }},
		GIF:  {gif.Decode, func(w io.Writer, m image.Image) error { return gif.Encode(w, m, nil) }},
		WEBP: {webp.Decode, nil}, // WEBP doesn't have a standard encoder in Go
		TIFF: {tiff.Decode, func(w io.Writer, m image.Image) error { return tiff.Encode(w, m, nil) }},
		BMP:  {bmp.Decode, bmp.Encode},
	}

	// Check if source and target formats are supported
	srcFormat, srcOk := formats[sourceContentType]
	tgtFormat, tgtOk := formats[targetContentType]
	if !srcOk || !tgtOk {
		return nil, fmt.Errorf("convert image: unsupported format: source=%s, target=%s", sourceContentType, targetContentType)
	}

	// Decode source image
	img, err := srcFormat.decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("convert image: failed to decode source image: %w", err)
	}

	// Encode to target format
	buf := new(bytes.Buffer)
	if err := tgtFormat.encode(buf, img); err != nil {
		return nil, fmt.Errorf("convert image: failed to encode to target format: %w", err)
	}

	return buf.Bytes(), nil
}

func convertAudio(raw []byte, sourceContentType, targetContentType string) ([]byte, error) {
	supportedFormats := []string{MP3, WAV, AAC, OGG, FLAC, M4A, WMA, AIFF, OCTETSTREAM}
	if !slices.Contains(supportedFormats, sourceContentType) || !slices.Contains(supportedFormats, targetContentType) {
		return nil, fmt.Errorf("convert audio: unsupported format: source=%s, target=%s", sourceContentType, targetContentType)
	}

	tempDir, err := os.MkdirTemp("", "audio_conversion_*")
	if err != nil {
		return nil, fmt.Errorf("convert audio: failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sourceExt := getExtensionFromMIME(sourceContentType)
	targetExt := getExtensionFromMIME(targetContentType)

	inputFile := filepath.Join(tempDir, "input"+sourceExt)
	outputFile := filepath.Join(tempDir, "output"+targetExt)

	if err := os.WriteFile(inputFile, raw, 0644); err != nil {
		return nil, fmt.Errorf("convert audio: failed to write input file: %w", err)
	}

	cmd := exec.Command("ffmpeg", "-i", inputFile, outputFile)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("convert audio: ffmpeg conversion failed: %w", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("convert audio: failed to read output file: %w", err)
	}

	return output, nil
}

func convertVideo(raw []byte, sourceContentType, targetContentType string) ([]byte, error) {
	supportedFormats := []string{MP4, AVI, MOV, WEBM, MKV, FLV, WMV, MPEG, OCTETSTREAM}
	if !slices.Contains(supportedFormats, sourceContentType) || !slices.Contains(supportedFormats, targetContentType) {
		return nil, fmt.Errorf("convert video: unsupported format: source=%s, target=%s", sourceContentType, targetContentType)
	}

	tempDir, err := os.MkdirTemp("", "video_conversion_*")
	if err != nil {
		return nil, fmt.Errorf("convert video: failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	sourceExt := getExtensionFromMIME(sourceContentType)
	targetExt := getExtensionFromMIME(targetContentType)

	inputFile := filepath.Join(tempDir, "input"+sourceExt)
	outputFile := filepath.Join(tempDir, "output"+targetExt)

	if err := os.WriteFile(inputFile, raw, 0644); err != nil {
		return nil, fmt.Errorf("convert video: failed to write input file: %w", err)
	}

	cmd := exec.Command("ffmpeg", "-i", inputFile, outputFile)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("convert video: ffmpeg conversion failed: %w", err)
	}

	output, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("convert video: failed to read output file: %w", err)
	}

	return output, nil
}

func getExtensionFromMIME(mimeType string) string {
	parts := strings.Split(mimeType, "/")
	if len(parts) != 2 {
		return ""
	}
	return "." + parts[1]
}
