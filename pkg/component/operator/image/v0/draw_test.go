package image

import (
	"image"
	"image/color"
	"reflect"
	"testing"
)

func TestBoundingBoxSize(t *testing.T) {
	bbox := &boundingBox{Top: 10, Left: 20, Width: 100, Height: 50}
	expected := 5000
	if size := bbox.Size(); size != expected {
		t.Errorf("Expected size %d, but got %d", expected, size)
	}
}

func TestIndexUniqueCategories(t *testing.T) {
	objs := []*detectionObject{
		{Category: "cat"},
		{Category: "dog"},
		{Category: "cat"},
		{Category: "bird"},
	}
	expected := map[string]int{"cat": 0, "dog": 1, "bird": 2}
	result := indexUniqueCategories(objs)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestRandomColor(t *testing.T) {
	color1 := randomColor(1, 255)
	color2 := randomColor(1, 255)
	if color1 != color2 {
		t.Errorf("Expected same colors for same seed, but got %v and %v", color1, color2)
	}

	color3 := randomColor(2, 255)
	if color1 == color3 {
		t.Errorf("Expected different colors for different seeds, but got %v and %v", color1, color3)
	}
}

func TestBlendColors(t *testing.T) {
	c1 := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	c2 := color.RGBA{R: 0, G: 255, B: 0, A: 128}
	expected := color.RGBA{R: 127, G: 128, B: 0, A: 255}
	result := blendColors(c1, c2)
	if result != expected {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestHasFalseNeighbor(t *testing.T) {
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
		if result != tt.expected {
			t.Errorf("For (%d, %d), expected %v, but got %v", tt.x, tt.y, tt.expected, result)
		}
	}
}

func TestFindContour(t *testing.T) {
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
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}

func TestRleDecode(t *testing.T) {
	rle := []int{3, 2, 1}
	width, height := 3, 2
	expected := [][]bool{
		{false, false, true},
		{false, true, false},
	}
	result := rleDecode(rle, width, height)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, but got %v", expected, result)
	}
}
