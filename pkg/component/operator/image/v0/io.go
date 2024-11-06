package image

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type concatInput struct {
	Images     []format.Image `instill:"images"`
	GridWidth  int            `instill:"grid-width"`
	GridHeight int            `instill:"grid-height"`
	Padding    int            `instill:"padding"`
}

type concatOutput struct {
	Image format.Image `instill:"image"`
}

type cropInput struct {
	Image        format.Image `instill:"image"`
	CornerRadius int          `instill:"corner-radius"`
	CircleRadius int          `instill:"circle-radius"`
	TopOffset    int          `instill:"top-offset"`
	RightOffset  int          `instill:"right-offset"`
	BottomOffset int          `instill:"bottom-offset"`
	LeftOffset   int          `instill:"left-offset"`
}

type cropOutput struct {
	Image format.Image `instill:"image"`
}

type resizeInput struct {
	Image  format.Image `instill:"image"`
	Width  int          `instill:"width"`
	Height int          `instill:"height"`
	Ratio  float64      `instill:"ratio"`
}

type resizeOutput struct {
	Image format.Image `instill:"image"`
}

type drawClassificationInput struct {
	Image     format.Image `instill:"image"`
	Category  string       `instill:"category"`
	Score     float64      `instill:"score"`
	ShowScore bool         `instill:"show-score"`
}

type drawClassificationOutput struct {
	Image format.Image `instill:"image"`
}

type detectionObject struct {
	BoundingBox *boundingBox `instill:"bounding-box" json:"bounding-box"`
	Category    string       `instill:"category" json:"category"`
	Score       float64      `instill:"score" json:"score"`
}

type drawDetectionInput struct {
	Image     format.Image       `instill:"image"`
	Objects   []*detectionObject `instill:"objects"`
	ShowScore bool               `instill:"show-score"`
}

type drawDetectionOutput struct {
	Image format.Image `instill:"image"`
}

type keypoint struct {
	X float64 `instill:"x" json:"x"`
	Y float64 `instill:"y" json:"y"`
	V float64 `instill:"v" json:"v"`
}

type keypointObject struct {
	BoundingBox *boundingBox `instill:"bounding-box" json:"bounding-box"`
	Keypoints   []*keypoint  `instill:"keypoints" json:"keypoints"`
	Score       float64      `instill:"score" json:"score"`
}

type drawKeypointInput struct {
	Image     format.Image      `instill:"image"`
	Objects   []*keypointObject `instill:"objects"`
	ShowScore bool              `instill:"show-score"`
}

type drawKeypointOutput struct {
	Image format.Image `instill:"image"`
}

type instanceSegmentationObject struct {
	BoundingBox *boundingBox `instill:"bounding-box" json:"bounding-box"`
	Category    string       `instill:"category" json:"category"`
	RLE         string       `instill:"rle" json:"rle"`
	Score       float64      `instill:"score" json:"score"`
}

type drawInstanceSegmentationInput struct {
	Image     format.Image                  `instill:"image"`
	Objects   []*instanceSegmentationObject `instill:"objects"`
	ShowScore bool                          `instill:"show-score"`
}

type drawInstanceSegmentationOutput struct {
	Image format.Image `instill:"image"`
}

type semanticSegmentationStuff struct {
	Category string `instill:"category" json:"category"`
	RLE      string `instill:"rle" json:"rle"`
}

type drawSemanticSegmentationInput struct {
	Image  format.Image                 `instill:"image"`
	Stuffs []*semanticSegmentationStuff `instill:"stuffs"`
}

type drawSemanticSegmentationOutput struct {
	Image format.Image `instill:"image"`
}

type ocrObject struct {
	BoundingBox *boundingBox `instill:"bounding-box" json:"bounding-box"`
	Text        string       `instill:"text" json:"text"`
	Score       float64      `instill:"score" json:"score"`
}

type drawOCRInput struct {
	Image     format.Image `instill:"image"`
	Objects   []*ocrObject `instill:"objects"`
	ShowScore bool         `instill:"show-score"`
}

type drawOCROutput struct {
	Image format.Image `instill:"image"`
}

type boundingBox struct {
	Top    int `instill:"top" json:"top"`
	Left   int `instill:"left" json:"left"`
	Width  int `instill:"width" json:"width"`
	Height int `instill:"height" json:"height"`
}

// Size returns the area of the bounding box.
func (b *boundingBox) Size() int {
	return b.Width * b.Height
}
