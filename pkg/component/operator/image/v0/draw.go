package image

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/opentype"
)

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

func indexUniqueCategories(objs []*detectionObject) map[string]int {
	catIdx := make(map[string]int)
	for _, obj := range objs {
		_, exist := catIdx[obj.Category]
		if !exist {
			catIdx[obj.Category] = len(catIdx)

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

func drawImageLabel(img *image.RGBA, category string, score float64, showScore bool) error {

	dc := gg.NewContextForRGBA(img)

	font, err := opentype.Parse(IBMPlexSansRegular)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(font, &opentype.FaceOptions{
		Size: 20,
		DPI:  72,
	})
	if err != nil {
		return err
	}

	dc.SetFontFace(face)

	labelText := category
	if showScore {
		labelText = fmt.Sprintf("%s: %.0f%%", category, score*100)
	}

	w, h := dc.MeasureString(labelText)

	padding := 2.0
	x := padding
	y := padding
	w += 6 * padding
	h += padding
	dc.SetRGB(0, 0, 0)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	dc.SetColor(color.RGBA{255, 255, 255, 255})
	dc.DrawString(labelText, 4*padding, 11*padding)
	return nil
}

func drawObjectLabel(img *image.RGBA, bbox *boundingBox, category string, score float64, showScore bool, colorSeed int) error {

	dc := gg.NewContextForRGBA(img)

	font, err := opentype.Parse(IBMPlexSansRegular)
	if err != nil {
		return err
	}

	face, err := opentype.NewFace(font, &opentype.FaceOptions{
		Size: 20,
		DPI:  72,
	})
	if err != nil {
		return err
	}

	dc.SetFontFace(face)

	labelText := category
	if showScore {
		labelText = fmt.Sprintf("%s: %.0f%%", category, score*100)
	}

	w, h := dc.MeasureString(labelText)

	padding := 2.0
	x := float64(bbox.Left) - 2*padding
	y := float64(bbox.Top) - 1.1*h - padding
	w += 4 * padding
	h += padding
	dc.SetRGBA(0, 0, 0, 128)
	dc.DrawRoundedRectangle(x, y, w, h, 4)
	dc.Fill()
	originalColor := color.RGBA{255, 255, 255, 255}
	blendedColor := blendColors(originalColor, randomColor(colorSeed, 64))
	dc.SetColor(blendedColor)
	dc.DrawString(labelText, float64(bbox.Left), float64(bbox.Top)-h/3-padding)

	return nil
}

var skeleton = [][]int{{16, 14}, {14, 12}, {17, 15}, {15, 13}, {12, 13}, {6, 12},
	{7, 13}, {6, 7}, {6, 8}, {7, 9}, {8, 10}, {9, 11}, {2, 3}, {1, 2}, {1, 3}, {2, 4}, {3, 5}, {4, 6}, {5, 7},
}

var keypointLimbColorIdx = []int{9, 9, 9, 9, 7, 7, 7, 0, 0, 0, 0, 0, 16, 16, 16, 16, 16, 16, 16}
var keypointColorIdx = []int{16, 16, 16, 16, 16, 0, 0, 0, 0, 0, 0, 9, 9, 9, 9, 9, 9}

func drawSkeleton(img *image.RGBA, kpts []*keypoint) error {
	dc := gg.NewContextForRGBA(img)
	for idx, kpt := range kpts {
		if kpt.V > 0.5 {
			dc.SetColor(palette[keypointColorIdx[idx]])
			dc.DrawPoint(kpt.X, kpt.Y, 2)
			dc.Fill()
		}
	}
	for idx, sk := range skeleton {
		if kpts[sk[0]-1].V > 0.5 && kpts[sk[1]-1].V > 0.5 {
			dc.SetColor(palette[keypointLimbColorIdx[idx]])
			dc.SetLineWidth(2)
			dc.DrawLine(kpts[sk[0]-1].X, kpts[sk[0]-1].Y, kpts[sk[1]-1].X, kpts[sk[1]-1].Y)
			dc.Stroke()
		}
	}
	return nil
}
