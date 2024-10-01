package image

import (
	"image"
	"image/color"
	"testing"

	"github.com/frankban/quicktest"
)

func TestBoundingBoxSize(t *testing.T) {
	c := quicktest.New(t)
	bbox := &boundingBox{Top: 10, Left: 20, Width: 100, Height: 50}
	expected := 5000
	c.Assert(bbox.Size(), quicktest.Equals, expected)
}

func TestIndexUniqueCategories(t *testing.T) {
	c := quicktest.New(t)
	objs := []*detectionObject{
		{Category: "cat"},
		{Category: "dog"},
		{Category: "cat"},
		{Category: "bird"},
	}
	expected := map[string]int{"cat": 0, "dog": 1, "bird": 2}
	result := indexUniqueCategories(objs)
	c.Assert(result, quicktest.DeepEquals, expected)
}

func TestRandomColor(t *testing.T) {
	c := quicktest.New(t)
	color1 := randomColor(1, 255)
	color2 := randomColor(1, 255)
	c.Assert(color1, quicktest.Equals, color2)

	color3 := randomColor(2, 255)
	c.Assert(color1, quicktest.Not(quicktest.Equals), color3)
}

func TestBlendColors(t *testing.T) {
	c := quicktest.New(t)
	c1 := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	c2 := color.RGBA{R: 0, G: 255, B: 0, A: 128}
	expected := color.RGBA{R: 127, G: 128, B: 0, A: 255}
	result := blendColors(c1, c2)
	c.Assert(result, quicktest.Equals, expected)
}

func TestHasFalseNeighbor(t *testing.T) {
	c := quicktest.New(t)
	mask := [][]bool{
		{true, true, false},
		{true, true, true},
		{false, true, true},
	}
	tests := []struct {
		x, y     int
		expected bool
	}{
		{1, 1, true},
		{0, 0, true},
		{2, 1, true},
		{1, 1, true},
	}
	for _, tt := range tests {
		result := hasFalseNeighbor(mask, tt.x, tt.y)
		c.Assert(result, quicktest.Equals, tt.expected)
	}
}

func TestFindContour(t *testing.T) {
	c := quicktest.New(t)
	mask := [][]bool{
		{false, true, false},
		{true, true, true},
		{false, true, false},
	}
	expected := []image.Point{
		{X: 1, Y: 0},
		{X: 0, Y: 1},
		{X: 1, Y: 1},
		{X: 2, Y: 1},
		{X: 1, Y: 2},
	}
	result := findContour(mask)
	c.Assert(result, quicktest.DeepEquals, expected)
}

func TestRleDecode(t *testing.T) {
	c := quicktest.New(t)
	rle := []int{3, 2, 1}
	width, height := 3, 2
	expected := [][]bool{
		{false, false, true},
		{false, true, false},
	}
	result := rleDecode(rle, width, height)
	c.Assert(result, quicktest.DeepEquals, expected)
}
