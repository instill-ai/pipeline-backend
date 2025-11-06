package data

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
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

	"github.com/instill-ai/pipeline-backend/pkg/data/cgo"
)

func convertImage(raw []byte, sourceContentType, targetContentType string) ([]byte, error) {
	// Handle HEIC/HEIF formats specially
	if sourceContentType == HEIC || sourceContentType == HEIF {
		return convertFromHEIF(raw, targetContentType)
	}
	if targetContentType == HEIC || targetContentType == HEIF {
		return convertToHEIF(raw, sourceContentType, targetContentType)
	}

	// Handle AVIF formats specially
	if sourceContentType == AVIF {
		return convertFromAVIF(raw, targetContentType)
	}
	if targetContentType == AVIF {
		return convertToAVIF(raw, sourceContentType)
	}

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

// convertFromHEIF converts HEIC/HEIF to other image formats
func convertFromHEIF(raw []byte, targetContentType string) ([]byte, error) {
	// Decode HEIF to RGB
	rgbData, width, height, err := cgo.DecodeHEIF(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIF: %w", err)
	}

	// Convert RGB data to Go image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			i := (y*width + x) * 3
			if i+2 < len(rgbData) {
				img.Set(x, y, color.RGBA{
					R: rgbData[i],
					G: rgbData[i+1],
					B: rgbData[i+2],
					A: 255,
				})
			}
		}
	}

	// Encode to target format
	buf := new(bytes.Buffer)
	switch targetContentType {
	case PNG:
		err = png.Encode(buf, img)
	case JPEG:
		err = jpeg.Encode(buf, img, nil)
	case GIF:
		err = gif.Encode(buf, img, nil)
	case TIFF:
		err = tiff.Encode(buf, img, nil)
	case BMP:
		err = bmp.Encode(buf, img)
	default:
		return nil, fmt.Errorf("unsupported target format: %s", targetContentType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode to %s: %w", targetContentType, err)
	}

	return buf.Bytes(), nil
}

// convertToHEIF converts other image formats to HEIC/HEIF
func convertToHEIF(raw []byte, sourceContentType, _ string) ([]byte, error) {
	// First decode source format to Go image
	var img image.Image
	var err error

	switch sourceContentType {
	case PNG:
		img, err = png.Decode(bytes.NewReader(raw))
	case JPEG:
		img, err = jpeg.Decode(bytes.NewReader(raw))
	case GIF:
		img, err = gif.Decode(bytes.NewReader(raw))
	case WEBP:
		img, err = webp.Decode(bytes.NewReader(raw))
	case TIFF:
		img, err = tiff.Decode(bytes.NewReader(raw))
	case BMP:
		img, err = bmp.Decode(bytes.NewReader(raw))
	default:
		return nil, fmt.Errorf("unsupported source format: %s", sourceContentType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode source image: %w", err)
	}

	// Convert Go image to RGB data
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	rgbData := make([]byte, width*height*3)

	for y := range height {
		for x := range width {
			r, g, b, _ := img.At(x, y).RGBA()
			i := (y*width + x) * 3
			rgbData[i] = byte(r >> 8)
			rgbData[i+1] = byte(g >> 8)
			rgbData[i+2] = byte(b >> 8)
		}
	}

	// Encode to HEIF
	quality := 100 // Default quality
	heifData, err := cgo.EncodeHEIF(rgbData, width, height, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to encode HEIF: %w", err)
	}

	return heifData, nil
}

// convertFromAVIF converts AVIF to other image formats
func convertFromAVIF(raw []byte, targetContentType string) ([]byte, error) {
	// Decode AVIF to RGB data
	rgbData, width, height, err := cgo.DecodeAVIF(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode AVIF: %w", err)
	}

	// Convert RGB data to Go image
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := (y*width + x) * 3
			r := rgbData[i]
			g := rgbData[i+1]
			b := rgbData[i+2]
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Encode to target format
	buf := new(bytes.Buffer)
	switch targetContentType {
	case PNG:
		err = png.Encode(buf, img)
	case JPEG:
		err = jpeg.Encode(buf, img, nil)
	case GIF:
		err = gif.Encode(buf, img, nil)
	case TIFF:
		err = tiff.Encode(buf, img, nil)
	case BMP:
		err = bmp.Encode(buf, img)
	case HEIC, HEIF:
		return convertToHEIF(rgbData, "rgb", targetContentType)
	default:
		return nil, fmt.Errorf("unsupported target format: %s", targetContentType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode to %s: %w", targetContentType, err)
	}

	return buf.Bytes(), nil
}

// convertToAVIF converts other image formats to AVIF
func convertToAVIF(raw []byte, sourceContentType string) ([]byte, error) {
	// First decode source format to Go image
	var img image.Image
	var err error

	switch sourceContentType {
	case PNG:
		img, err = png.Decode(bytes.NewReader(raw))
	case JPEG:
		img, err = jpeg.Decode(bytes.NewReader(raw))
	case GIF:
		img, err = gif.Decode(bytes.NewReader(raw))
	case WEBP:
		img, err = webp.Decode(bytes.NewReader(raw))
	case TIFF:
		img, err = tiff.Decode(bytes.NewReader(raw))
	case BMP:
		img, err = bmp.Decode(bytes.NewReader(raw))
	case HEIC, HEIF:
		// Decode HEIF to RGB data first
		rgbData, width, height, heifErr := cgo.DecodeHEIF(raw)
		if heifErr != nil {
			return nil, fmt.Errorf("failed to decode HEIF: %w", heifErr)
		}
		// Convert RGB data to AVIF
		quality := 80 // Default quality
		return cgo.EncodeAVIF(rgbData, width, height, quality)
	default:
		return nil, fmt.Errorf("unsupported source format: %s", sourceContentType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decode source image: %w", err)
	}

	// Convert Go image to RGB data
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	rgbData := make([]byte, width*height*3)

	for y := range height {
		for x := range width {
			r, g, b, _ := img.At(x, y).RGBA()
			i := (y*width + x) * 3
			rgbData[i] = byte(r >> 8)
			rgbData[i+1] = byte(g >> 8)
			rgbData[i+2] = byte(b >> 8)
		}
	}

	// Encode to AVIF
	quality := 80 // Default quality
	avifData, err := cgo.EncodeAVIF(rgbData, width, height, quality)
	if err != nil {
		return nil, fmt.Errorf("failed to encode AVIF: %w", err)
	}

	return avifData, nil
}

func convertAudio(raw []byte, sourceContentType, targetContentType string) ([]byte, error) {
	supportedFormats := []string{MP3, WAV, AAC, OGG, FLAC, M4A, WMA, AIFF, WEBMAUDIO, OCTETSTREAM, "video/x-ms-asf", "video/mp4"}
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
	supportedFormats := []string{MP4, AVI, MOV, WEBMVIDEO, MKV, FLV, WMV, MPEG, OCTETSTREAM, "video/x-ms-asf"}
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
