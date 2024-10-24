package path

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type SegmentType string

const (
	IndexSegment     SegmentType = "index"
	KeySegment       SegmentType = "key"
	AttributeSegment SegmentType = "attribute"
)

type Path struct {
	Segments []*Segment
	Source   string
}

type Segment struct {
	SegmentType    SegmentType
	Key            string
	Index          int
	Attribute      string
	OriginalString string
}

func (p *Path) TrimFirst() (*Segment, *Path, error) {
	if len(p.Segments) == 0 {
		return nil, nil, fmt.Errorf("invalid path: no valid segments found %s %+v", p.Source, p.Segments)
	}

	firstSeg := p.Segments[0]
	newSegments := p.Segments[1:]
	if len(newSegments) > 0 && newSegments[0].SegmentType == KeySegment {
		// newSegments[0].Key = strings.TrimLeft(newSegments[0].Key, ".")
		newSegments[0].OriginalString = strings.TrimLeft(newSegments[0].OriginalString, ".")
	}
	newP := &Path{
		Segments: newSegments,
		Source:   p.Source,
	}
	return firstSeg, newP, nil
}

func (p Path) IsEmpty() bool {
	return len(p.Segments) == 0
}

func (p Path) String() string {
	var parts []string
	for _, seg := range p.Segments {
		parts = append(parts, seg.OriginalString)
	}
	return strings.Join(parts, "")
}

func NewPath(path string) (p *Path, err error) {
	if path == "" {
		return &Path{Source: path, Segments: make([]*Segment, 0)}, nil
	}

	p = &Path{Source: path, Segments: make([]*Segment, 0)}

	re := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_-]*|\.[a-zA-Z_][a-zA-Z0-9_-]*|\[\d+\]|\["[^"]+"\]|:[a-zA-Z0-9_-]+)`)
	parts := re.FindAllString(path, -1)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path: no valid segments found %s", path)
	}
	length := strings.Join(parts, "")
	if length != path {
		return nil, fmt.Errorf("invalid path: %s", path)
	}

	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			// Handle attribute segment
			attr := strings.TrimPrefix(part, ":")
			p.Segments = append(p.Segments, &Segment{
				SegmentType:    AttributeSegment,
				Attribute:      attr,
				OriginalString: part,
			})
		} else if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			// Handle index or quoted key segment
			inner := strings.Trim(part, "[]")
			if strings.HasPrefix(inner, "\"") && strings.HasSuffix(inner, "\"") {
				// Quoted key
				key := strings.Trim(inner, "\"")
				p.Segments = append(p.Segments, &Segment{
					SegmentType:    KeySegment,
					Key:            key,
					OriginalString: part,
				})
			} else {
				// Index
				index, err := strconv.Atoi(inner)
				if err != nil {
					return nil, fmt.Errorf("invalid index: %s", part)
				}
				p.Segments = append(p.Segments, &Segment{
					SegmentType:    IndexSegment,
					Index:          index,
					OriginalString: part,
				})
			}
		} else {
			// Handle regular key segment
			key := strings.TrimLeft(part, ".")
			p.Segments = append(p.Segments, &Segment{
				SegmentType:    KeySegment,
				Key:            key,
				OriginalString: part,
			})
		}
	}
	return p, nil
}
