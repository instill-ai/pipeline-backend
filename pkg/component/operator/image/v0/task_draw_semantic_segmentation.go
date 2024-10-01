package image

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type semanticSegmentationStuff struct {
	Category string `json:"category"`
	RLE      string `json:"rle"`
}

type drawSemanticSegmentationInput struct {
	Image  base64Image                  `json:"image"`
	Stuffs []*semanticSegmentationStuff `json:"stuffs"`
}

type drawSemanticSegmentationOutput struct {
	Image base64Image `json:"image"`
}

func drawSemanticMask(img *image.RGBA, rle string, colorSeed int) error {
	// Split the string by commas to get the individual number strings.
	numberStrings := strings.Split(rle, ",")

	// Allocate an array of integers with the same length as the number of numberStrings.
	rleInts := make([]int, len(numberStrings))

	// Convert each number string to an integer.
	for i, s := range numberStrings {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("failed to convert RLE string to int: %s, error: %v", s, err)
		}
		rleInts[i] = n
	}

	bound := img.Bounds()

	// Decode the RLE mask for the full image size.
	mask := rleDecode(rleInts, bound.Dx(), bound.Dy())

	// Iterate over the bounding box and draw the mask onto the image.
	for y := 0; y < bound.Dy(); y++ {
		for x := 0; x < bound.Dx(); x++ {
			if mask[y][x] {
				// The mask is present for this pixel, so draw it on the image.
				// Here you could set a specific color or just use the mask value.
				// For example, let's paint the mask as a red semi-transparent overlay:
				originalColor := img.At(x, y).(color.RGBA)
				// Blend the original color with the mask color.
				blendedColor := blendColors(originalColor, randomColor(colorSeed, 128))
				img.Set(x, y, blendedColor)
			}
		}
	}

	dc := gg.NewContextForRGBA(img)
	dc.SetColor(color.RGBA{255, 255, 255, 255})

	// Find contour points
	contourPoints := findContour(mask)

	// Draw the contour
	for _, pt := range contourPoints {
		// Scale points as needed for your canvas size
		dc.DrawPoint(float64(pt.X), float64(pt.Y), 0.5)
		dc.Fill()
	}

	return nil
}

func drawSemanticSegmentation(input *structpb.Struct, job *base.Job, ctx context.Context) (*structpb.Struct, error) {

	inputStruct := drawSemanticSegmentationInput{}

	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error converting input to struct: %v", err)
	}

	img, err := decodeBase64Image(string(inputStruct.Image))
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	imgRGBA := convertToRGBA(img)

	for idx, stuff := range inputStruct.Stuffs {
		if err := drawSemanticMask(imgRGBA, stuff.RLE, idx); err != nil {
			return nil, err
		}
	}

	base64Img, err := encodeBase64Image(imgRGBA)
	if err != nil {
		return nil, err
	}

	output := drawSemanticSegmentationOutput{
		Image: base64Image(fmt.Sprintf("data:image/png;base64,%s", base64Img)),
	}

	return base.ConvertToStructpb(output)
}
