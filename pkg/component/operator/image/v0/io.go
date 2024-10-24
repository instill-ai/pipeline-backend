package image

import (
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

type concatInput struct {
	Images     []format.Image `key:"images"`
	GridWidth  int            `key:"grid-width"`
	GridHeight int            `key:"grid-height"`
	Padding    int            `key:"padding"`
}

type concatOutput struct {
	Image format.Image `key:"image"`
}

type cropInput struct {
	Image        format.Image `key:"image"`
	CornerRadius int          `key:"corner-radius"`
	CircleRadius int          `key:"circle-radius"`
	TopOffset    int          `key:"top-offset"`
	RightOffset  int          `key:"right-offset"`
	BottomOffset int          `key:"bottom-offset"`
	LeftOffset   int          `key:"left-offset"`
}

type cropOutput struct {
	Image format.Image `key:"image"`
}

type resizeInput struct {
	Image  format.Image `key:"image"`
	Width  int          `key:"width"`
	Height int          `key:"height"`
	Ratio  float64      `key:"ratio"`
}

type resizeOutput struct {
	Image format.Image `key:"image"`
}

type drawClassificationInput struct {
	Image     format.Image `key:"image"`
	Category  string       `key:"category"`
	Score     float64      `key:"score"`
	ShowScore bool         `key:"show-score"`
}

type drawClassificationOutput struct {
	Image format.Image `key:"image"`
}

type detectionObject struct {
	BoundingBox *boundingBox `key:"bounding-box" json:"bounding-box"`
	Category    string       `key:"category" json:"category"`
	Score       float64      `key:"score" json:"score"`
}

type drawDetectionInput struct {
	Image     format.Image       `key:"image"`
	Objects   []*detectionObject `key:"objects"`
	ShowScore bool               `key:"show-score"`
}

type drawDetectionOutput struct {
	Image format.Image `key:"image"`
}

type keypoint struct {
	X float64 `key:"x" json:"x"`
	Y float64 `key:"y" json:"y"`
	V float64 `key:"v" json:"v"`
}

type keypointObject struct {
	BoundingBox *boundingBox `key:"bounding-box" json:"bounding-box"`
	Keypoints   []*keypoint  `key:"keypoints" json:"keypoints"`
	Score       float64      `key:"score" json:"score"`
}

type drawKeypointInput struct {
	Image     format.Image      `key:"image"`
	Objects   []*keypointObject `key:"objects"`
	ShowScore bool              `key:"show-score"`
}

type drawKeypointOutput struct {
	Image format.Image `key:"image"`
}

// type drawOCRInput struct {
// 	Image format.Image `key:"image"`
// 	OCR   []ocrResult  `key:"ocr"`
// }

// type drawOCROutput struct {
// 	Image format.Image `key:"image"`
// }

type instanceSegmentationObject struct {
	BoundingBox *boundingBox `key:"bounding-box" json:"bounding-box"`
	Category    string       `key:"category" json:"category"`
	RLE         string       `key:"rle" json:"rle"`
	Score       float64      `key:"score" json:"score"`
}

type drawInstanceSegmentationInput struct {
	Image     format.Image                  `key:"image"`
	Objects   []*instanceSegmentationObject `key:"objects"`
	ShowScore bool                          `key:"show-score"`
}

type drawInstanceSegmentationOutput struct {
	Image format.Image `key:"image"`
}

type semanticSegmentationStuff struct {
	Category string `key:"category" json:"category"`
	RLE      string `key:"rle" json:"rle"`
}

type drawSemanticSegmentationInput struct {
	Image  format.Image                 `key:"image"`
	Stuffs []*semanticSegmentationStuff `key:"stuffs"`
}

type drawSemanticSegmentationOutput struct {
	Image format.Image `key:"image"`
}

type ocrObject struct {
	BoundingBox *boundingBox `key:"bounding-box" json:"bounding-box"`
	Text        string       `key:"text" json:"text"`
	Score       float64      `key:"score" json:"score"`
}

type drawOCRInput struct {
	Image     format.Image `key:"image"`
	Objects   []*ocrObject `key:"objects"`
	ShowScore bool         `key:"show-score"`
}

type drawOCROutput struct {
	Image format.Image `key:"image"`
}

// type drawSemanticSegmentationInput struct {
// 	Image        format.Image `key:"image"`
// 	Segmentation format.Image `key:"segmentation"`
// }

// type drawSemanticSegmentationOutput struct {
// 	Image format.Image `key:"image"`
// }

// type detection struct {
// 	Label       string  `key:"label"`
// 	Confidence  float64 `key:"confidence"`
// 	BoundingBox bbox    `key:"bounding-box"`
// }

// type keypoint struct {
// 	X     float64 `key:"x"`
// 	Y     float64 `key:"y"`
// 	Label string  `key:"label"`
// }

// type ocrResult struct {
// 	Text        string  `key:"text"`
// 	Confidence  float64 `key:"confidence"`
// 	BoundingBox bbox    `key:"bounding-box"`
// }

// type instanceSegmentationObject struct {
// 	Label       string  `key:"label"`
// 	Confidence  float64 `key:"confidence"`
// 	BoundingBox bbox    `key:"bounding-box"`
// 	Mask        string  `key:"mask"`
// }

type boundingBox struct {
	Top    int `key:"top" json:"top"`
	Left   int `key:"left" json:"left"`
	Width  int `key:"width" json:"width"`
	Height int `key:"height" json:"height"`
}

// Size returns the area of the bounding box.
func (b *boundingBox) Size() int {
	return b.Width * b.Height
}
