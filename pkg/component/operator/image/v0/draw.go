package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/opentype"
	"google.golang.org/protobuf/types/known/structpb"
)

// BoundingBox holds the coordinates of a bounding box.
type BoundingBox struct {
	Top    int
	Left   int
	Width  int
	Height int
}

// Size returns the area of the bounding box.
func (b *BoundingBox) Size() int {
	return b.Width * b.Height
}

func structpbToBoundingBox(s *structpb.Struct) *BoundingBox {
	return &BoundingBox{
		Top:    int(s.Fields["top"].GetNumberValue()),
		Left:   int(s.Fields["left"].GetNumberValue()),
		Width:  int(s.Fields["width"].GetNumberValue()),
		Height: int(s.Fields["height"].GetNumberValue()),
	}
}

// Keypoint holds the coordinates of a keypoint.
type Keypoint struct {
	x float64
	y float64
	v float64
}

func structpbToKeypoint(s *structpb.Struct) *Keypoint {
	return &Keypoint{
		x: s.Fields["x"].GetNumberValue(),
		y: s.Fields["y"].GetNumberValue(),
		v: s.Fields["v"].GetNumberValue(),
	}
}

// Use the same color palette defined in yolov7: https://github.com/WongKinYiu/yolov7/blob/main/utils/plots.py#L449-L462
var palette = []color.RGBA{
	{255, 128, 0, 255},
	{255, 153, 51, 255},
	{255, 178, 102, 255},
	{230, 230, 0, 255},
	{255, 153, 255, 255},
	{153, 204, 255, 255},
	{255, 102, 255, 255},
	{255, 51, 255, 255},
	{102, 178, 255, 255},
	{51, 153, 255, 255},
	{255, 153, 153, 255},
	{255, 102, 102, 255},
	{255, 51, 51, 255},
	{153, 255, 153, 255},
	{102, 255, 102, 255},
	{51, 255, 51, 255},
	{0, 255, 0, 255},
	{0, 0, 255, 255},
	{255, 0, 0, 255},
	{255, 255, 255, 255},
}

var skeleton = [][]int{{16, 14}, {14, 12}, {17, 15}, {15, 13}, {12, 13}, {6, 12},
	{7, 13}, {6, 7}, {6, 8}, {7, 9}, {8, 10}, {9, 11}, {2, 3}, {1, 2}, {1, 3}, {2, 4}, {3, 5}, {4, 6}, {5, 7},
}

var keypointLimbColorIdx = []int{9, 9, 9, 9, 7, 7, 7, 0, 0, 0, 0, 0, 16, 16, 16, 16, 16, 16, 16}
var keypointColorIdx = []int{16, 16, 16, 16, 16, 0, 0, 0, 0, 0, 0, 9, 9, 9, 9, 9, 9}

func convertToBase64(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100})
	if err != nil {
		return nil, err
	}
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
	base64Bytes := []byte(base64Str)
	return base64Bytes, nil
}

func convertToRGBA(img image.Image) *image.RGBA {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			rgba.Set(x, y, color.RGBAModel.Convert(originalColor))
		}
	}
	return rgba
}

func indexUniqueCategories(objs []*structpb.Value) map[string]int {
	catIdx := make(map[string]int)
	for _, obj := range objs {
		_, exist := catIdx[obj.GetStructValue().Fields["category"].GetStringValue()]
		if !exist {
			catIdx[obj.GetStructValue().Fields["category"].GetStringValue()] = len(catIdx)
		}
	}
	return catIdx
}

// randomColor generates a random color with full opacity.
func randomColor(seed int, alpha uint8) color.RGBA {
	// Seed random number generator with current time
	r := rand.New(rand.NewSource(int64(seed)))
	return color.RGBA{
		R: uint8(r.Intn(256)), // Random number between 0 and 255
		G: uint8(r.Intn(256)),
		B: uint8(r.Intn(256)),
		A: alpha,
	}
}

// blendColors blends two RGBA colors together.
func blendColors(c1, c2 color.RGBA) color.RGBA {
	// Simple alpha blending
	alpha := float64(c2.A) / 255
	return color.RGBA{
		R: uint8((float64(c1.R)*(1-alpha) + float64(c2.R)*alpha)),
		G: uint8((float64(c1.G)*(1-alpha) + float64(c2.G)*alpha)),
		B: uint8((float64(c1.B)*(1-alpha) + float64(c2.B)*alpha)),
		A: 255, // you might want to adjust this if you need transparency
	}
}

// This function checks if a given point has at least one false neighbor.
func hasFalseNeighbor(mask [][]bool, x, y int) bool {
	// Check all eight neighbors
	directions := []struct{ dx, dy int }{
		{-1, 0}, {1, 0}, // Horizontal neighbors
		{0, -1}, {0, 1}, // Vertical neighbors
		{-1, -1}, {1, -1}, // Diagonal neighbors
		{-1, 1}, {1, 1},
	}

	for _, dir := range directions {
		newX, newY := x+dir.dx, y+dir.dy
		// Check bounds
		if newX < 0 || newX >= len(mask[0]) || newY < 0 || newY >= len(mask) {
			return true // Outside bounds, so treat as a "false" neighbor
		}
		if !mask[newY][newX] {
			return true // It has a false neighbor
		}
	}

	return false
}

// This function finds the contour points of a boolean mask.
func findContour(mask [][]bool) []image.Point {
	var points []image.Point

	for y := 0; y < len(mask); y++ {
		for x := 0; x < len(mask[y]); x++ {
			// Check if the current point is true, and if it has a false neighbor
			if mask[y][x] && hasFalseNeighbor(mask, x, y) {
				points = append(points, image.Point{X: x, Y: y})
			}
		}
	}

	return points
}

func rleDecode(rle []int, width, height int) [][]bool {
	// Create a 2D slice to hold the mask.
	mask := make([][]bool, height)
	for i := range mask {
		mask[i] = make([]bool, width)
	}

	x, y := 0, 0
	fill := false

	for _, val := range rle {
		for v := 0; v < val; v++ {
			mask[y][x] = fill
			y++
			if y >= height {
				y = 0
				x++
			}
		}
		fill = !fill // Alternate between filling and skipping.
	}
	return mask
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

func drawInstanceMask(img *image.RGBA, bbox *BoundingBox, rle string, colorSeed int) error {

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

	// Decode the RLE mask for the full image size.
	mask := rleDecode(rleInts, bbox.Width, bbox.Height)

	// Iterate over the bounding box and draw the mask onto the image.
	for y := 0; y < bbox.Height; y++ {
		for x := 0; x < bbox.Width; x++ {
			if mask[y][x] {
				// The mask is present for this pixel, so draw it on the image.
				// Here you could set a specific color or just use the mask value.
				// For example, let's paint the mask as a red semi-transparent overlay:
				originalColor := img.At(x+bbox.Left, y+bbox.Top).(color.RGBA)
				// Blend the original color with the mask color.
				blendedColor := blendColors(originalColor, randomColor(colorSeed, 156))
				img.Set(x+bbox.Left, y+bbox.Top, blendedColor)
			}
		}
	}

	dc := gg.NewContextForRGBA(img)
	dc.SetColor(randomColor(colorSeed, 255))
	contourPoints := findContour(mask)
	for _, pt := range contourPoints {
		dc.DrawPoint(float64(pt.X+bbox.Left), float64(pt.Y+bbox.Top), 0.5)
		dc.Fill()
	}

	return nil
}

func drawBoundingBox(img *image.RGBA, bbox *BoundingBox, colorSeed int) error {
	dc := gg.NewContextForRGBA(img)
	originalColor := img.At(bbox.Left, bbox.Top).(color.RGBA)
	blendedColor := blendColors(originalColor, randomColor(colorSeed, 255))
	dc.SetColor(blendedColor)
	dc.SetLineWidth(3)
	dc.DrawRoundedRectangle(float64(bbox.Left), float64(bbox.Top), float64(bbox.Width), float64(bbox.Height), 4)
	dc.Stroke()
	return nil
}

func drawSkeleton(img *image.RGBA, kpts []*Keypoint) error {
	dc := gg.NewContextForRGBA(img)
	for idx, kpt := range kpts {
		if kpt.v > 0.5 {
			dc.SetColor(palette[keypointColorIdx[idx]])
			dc.DrawPoint(kpt.x, kpt.y, 2)
			dc.Fill()
		}
	}
	for idx, sk := range skeleton {
		if kpts[sk[0]-1].v > 0.5 && kpts[sk[1]-1].v > 0.5 {
			dc.SetColor(palette[keypointLimbColorIdx[idx]])
			dc.SetLineWidth(2)
			dc.DrawLine(kpts[sk[0]-1].x, kpts[sk[0]-1].y, kpts[sk[1]-1].x, kpts[sk[1]-1].y)
			dc.Stroke()
		}
	}
	return nil
}

func drawImageLabel(img *image.RGBA, category string, score float64) error {

	dc := gg.NewContextForRGBA(img)

	// Parse the font
	font, err := opentype.Parse(IBMPlexSansRegular)
	if err != nil {
		return err
	}

	// Create a font face
	face, err := opentype.NewFace(font, &opentype.FaceOptions{
		Size: 20,
		DPI:  72,
	})
	if err != nil {
		return err
	}

	// Set the font face
	dc.SetFontFace(face)

	w, h := dc.MeasureString(category)

	// Set the rectangle padding
	padding := 2.0

	x := padding
	y := padding
	w += 6 * padding
	h += padding
	dc.SetRGB(0, 0, 0)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawString(category, 4*padding, 11*padding)
	return nil
}

func drawObjectLabel(img *image.RGBA, bbox *BoundingBox, category string, maskAdjustment bool, colorSeed int) error {

	dc := gg.NewContextForRGBA(img)

	// Parse the font
	font, err := opentype.Parse(IBMPlexSansRegular)
	if err != nil {
		return err
	}

	// Create a font face
	face, err := opentype.NewFace(font, &opentype.FaceOptions{
		Size: 20,
		DPI:  72,
	})
	if err != nil {
		return err
	}

	// Set the font face
	dc.SetFontFace(face)

	w, h := dc.MeasureString(category)

	// Set the rectangle padding
	padding := 2.0

	if bbox.Size() > 10000 && maskAdjustment {
		x := float64(bbox.Left) - 2*padding
		y := float64(bbox.Top) + float64(bbox.Height)/2 - padding
		w += 4 * padding
		h += padding
		dc.SetRGBA(0, 0, 0, 128)
		dc.DrawRoundedRectangle(x, y, w, h, 4)
		dc.Fill()
		// Draw the text centered on the screen
		originalColor := color.RGBA{255, 255, 255, 255}
		// Blend the original color with the mask color.
		blendedColor := blendColors(originalColor, randomColor(colorSeed, 64))
		dc.SetColor(blendedColor)
		dc.DrawString(category, float64(bbox.Left), float64(bbox.Top)+float64(bbox.Height)/2+8*padding)
	} else {
		x := float64(bbox.Left) - 2*padding
		y := float64(bbox.Top) - 1.1*h - padding
		w += 4 * padding
		h += padding
		dc.SetRGBA(0, 0, 0, 128)
		dc.DrawRoundedRectangle(x, y, w, h, 4)
		dc.Fill()
		// Draw the text centered on the screen
		originalColor := color.RGBA{255, 255, 255, 255}
		// Blend the original color with the mask color.
		blendedColor := blendColors(originalColor, randomColor(colorSeed, 64))
		dc.SetColor(blendedColor)
		dc.DrawString(category, float64(bbox.Left), float64(bbox.Top)-h/3-padding)
	}

	return nil
}

func draOCRLabel(img *image.RGBA, bbox *BoundingBox, text string) error {

	dc := gg.NewContextForRGBA(img)

	// Parse the font
	font, err := opentype.Parse(IBMPlexSansRegular)
	if err != nil {
		return err
	}

	// Create a font face
	face, err := opentype.NewFace(font, &opentype.FaceOptions{
		Size: 20,
		DPI:  72,
	})
	if err != nil {
		return err
	}

	// Set the font face
	dc.SetFontFace(face)

	w, h := dc.MeasureString(text)

	// Set the rectangle padding
	padding := 2.0

	x := float64(bbox.Left)
	y := float64(bbox.Top)
	w += 4 * padding
	h += padding
	dc.SetRGBA(0, 0, 0, 128)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawString(text, float64(bbox.Left)+2*padding, float64(bbox.Top)+h-4*padding)

	return nil
}

func drawClassification(srcImg image.Image, category string, score float64) ([]byte, error) {
	img := convertToRGBA(srcImg)

	if err := drawImageLabel(img, category, score); err != nil {
		return nil, err
	}

	base64Img, err := convertToBase64(img)
	if err != nil {
		return nil, err
	}
	return base64Img, nil
}

func drawDetection(srcImg image.Image, objs []*structpb.Value) ([]byte, error) {
	img := convertToRGBA(srcImg)

	catIdx := indexUniqueCategories(objs)

	for _, obj := range objs {
		bbox := structpbToBoundingBox(obj.GetStructValue().Fields["bounding-box"].GetStructValue())
		if err := drawBoundingBox(img, bbox, catIdx[obj.GetStructValue().Fields["category"].GetStringValue()]); err != nil {
			return nil, err
		}
	}

	for _, obj := range objs {
		bbox := structpbToBoundingBox(obj.GetStructValue().Fields["bounding-box"].GetStructValue())
		if err := drawObjectLabel(img, bbox, obj.GetStructValue().Fields["category"].GetStringValue(), false, catIdx[obj.GetStructValue().Fields["category"].GetStringValue()]); err != nil {
			return nil, err
		}
	}

	base64Img, err := convertToBase64(img)
	if err != nil {
		return nil, err
	}
	return base64Img, nil
}

func drawKeypoint(srcImg image.Image, objs []*structpb.Value) ([]byte, error) {
	img := convertToRGBA(srcImg)
	for _, obj := range objs {
		kpts := make([]*Keypoint, len(obj.GetStructValue().Fields["keypoints"].GetListValue().Values))
		for idx, kpt := range obj.GetStructValue().Fields["keypoints"].GetListValue().Values {
			kpts[idx] = structpbToKeypoint(kpt.GetStructValue())
		}
		if err := drawSkeleton(img, kpts); err != nil {
			return nil, err
		}
	}

	base64Img, err := convertToBase64(img)
	if err != nil {
		return nil, err
	}
	return base64Img, nil
}

func drawOCR(srcImg image.Image, objs []*structpb.Value) ([]byte, error) {
	img := convertToRGBA(srcImg)

	for _, obj := range objs {
		bbox := structpbToBoundingBox(obj.GetStructValue().Fields["bounding-box"].GetStructValue())
		if err := draOCRLabel(img, bbox, obj.GetStructValue().Fields["text"].GetStringValue()); err != nil {
			return nil, err
		}
	}

	base64Img, err := convertToBase64(img)
	if err != nil {
		return nil, err
	}
	return base64Img, nil
}

func drawInstanceSegmentation(srcImg image.Image, objs []*structpb.Value) ([]byte, error) {

	img := convertToRGBA(srcImg)

	// Sort the objects by size.
	sort.Slice(objs, func(i, j int) bool {
		bbox1 := structpbToBoundingBox(objs[i].GetStructValue().Fields["bounding-box"].GetStructValue())
		bbox2 := structpbToBoundingBox(objs[j].GetStructValue().Fields["bounding-box"].GetStructValue())
		return bbox1.Size() > bbox2.Size()
	})

	for instIdx, obj := range objs {
		bbox := structpbToBoundingBox(obj.GetStructValue().Fields["bounding-box"].GetStructValue())
		if err := drawInstanceMask(img, bbox, obj.GetStructValue().Fields["rle"].GetStringValue(), instIdx); err != nil {
			return nil, err
		}
	}

	for instIdx, obj := range objs {
		bbox := structpbToBoundingBox(obj.GetStructValue().Fields["bounding-box"].GetStructValue())
		text := obj.GetStructValue().Fields["category"].GetStringValue()
		if err := drawObjectLabel(img, bbox, text, true, instIdx); err != nil {
			return nil, err
		}
	}

	base64Img, err := convertToBase64(img)
	if err != nil {
		return nil, err
	}
	return base64Img, nil
}

func drawSemanticSegmentation(srcImg image.Image, stuffs []*structpb.Value) ([]byte, error) {
	img := convertToRGBA(srcImg)

	for idx, stuff := range stuffs {
		if err := drawSemanticMask(img, stuff.GetStructValue().Fields["rle"].GetStringValue(), idx); err != nil {
			return nil, err
		}
	}

	base64Img, err := convertToBase64(img)
	if err != nil {
		return nil, err
	}
	return base64Img, nil
}
