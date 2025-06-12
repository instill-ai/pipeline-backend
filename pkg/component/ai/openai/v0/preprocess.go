package openai

func resizeImage(width, height int) (int, int) {
	// Resize image to match OpenAI's model requirements.
	// OpenAI requires images to:
	// 1. Scale to fit in a 2048px x 2048px square, maintaining original aspect ratio
	// 2. Scale so that the image's shortest side is 768px long
	// We pre-resize to reduce payload size and improve performance.
	// Reference: https://platform.openai.com/docs/guides/images-vision#gpt-4o-gpt-4-1-gpt-4o-mini-cua-and-o-series-except-o4-mini
	// First, ensure the image fits within 2048px x 2048px square
	maxDimension := 2048
	if width > maxDimension || height > maxDimension {
		ratio := float64(width) / float64(height)
		if width > height {
			width = maxDimension
			height = int(float64(width) / ratio)
		} else {
			height = maxDimension
			width = int(float64(height) * ratio)
		}
	}

	// Then, ensure the shortest side is exactly 768px
	minDimension := 768
	if width != minDimension && height != minDimension {
		ratio := float64(width) / float64(height)
		if width < height {
			// Width is shorter side
			width = minDimension
			height = int(float64(width) / ratio)
		} else {
			// Height is shorter side
			height = minDimension
			width = int(float64(height) * ratio)
		}
	}
	return width, height
}
