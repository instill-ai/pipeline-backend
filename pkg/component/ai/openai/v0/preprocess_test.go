package openai

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestResizeImage(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name           string
		inputWidth     int
		inputHeight    int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "image already within bounds",
			inputWidth:     1024,
			inputHeight:    768,
			expectedWidth:  1024,
			expectedHeight: 768,
		},
		{
			name:           "image too large, width is larger",
			inputWidth:     3000,
			inputHeight:    2000,
			expectedWidth:  1152,
			expectedHeight: 768,
		},
		{
			name:           "image too large, height is larger",
			inputWidth:     2000,
			inputHeight:    3000,
			expectedWidth:  768,
			expectedHeight: 1152,
		},
		{
			name:           "image needs shortest side scaling, width is shorter",
			inputWidth:     512,
			inputHeight:    1024,
			expectedWidth:  768,
			expectedHeight: 1536,
		},
		{
			name:           "image needs shortest side scaling, height is shorter",
			inputWidth:     1024,
			inputHeight:    512,
			expectedWidth:  1536,
			expectedHeight: 768,
		},
		{
			name:           "square image",
			inputWidth:     1000,
			inputHeight:    1000,
			expectedWidth:  768,
			expectedHeight: 768,
		},
		{
			name:           "very large square image",
			inputWidth:     5000,
			inputHeight:    5000,
			expectedWidth:  768,
			expectedHeight: 768,
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			width, height := resizeImage(tt.inputWidth, tt.inputHeight)
			c.Assert(width, qt.Equals, tt.expectedWidth)
			c.Assert(height, qt.Equals, tt.expectedHeight)
		})
	}
}
