package path

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestNewPath(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name     string
		path     string
		expected *Path
		wantErr  bool
	}{
		{
			name: "Simple key path",
			path: "a.b.c",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: KeySegment, Key: "b", OriginalString: ".b"},
					{SegmentType: KeySegment, Key: "c", OriginalString: ".c"},
				},
				Source: "a.b.c",
			},
			wantErr: false,
		},
		{
			name: "Path with index",
			path: "a.b[1].c",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: KeySegment, Key: "b", OriginalString: ".b"},
					{SegmentType: IndexSegment, Index: 1, OriginalString: "[1]"},
					{SegmentType: KeySegment, Key: "c", OriginalString: ".c"},
				},
				Source: "a.b[1].c",
			},
			wantErr: false,
		},
		{
			name: "Path with quoted key",
			path: "a[\"b\"].c",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: KeySegment, Key: "b", OriginalString: "[\"b\"]"},
					{SegmentType: KeySegment, Key: "c", OriginalString: ".c"},
				},
				Source: "a[\"b\"].c",
			},
			wantErr: false,
		},
		{
			name: "Path with attribute",
			path: "a.b.c[1]:attr-a",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: KeySegment, Key: "b", OriginalString: ".b"},
					{SegmentType: KeySegment, Key: "c", OriginalString: ".c"},
					{SegmentType: IndexSegment, Index: 1, OriginalString: "[1]"},
					{SegmentType: AttributeSegment, Attribute: "attr-a", OriginalString: ":attr-a"},
				},
				Source: "a.b.c[1]:attr-a",
			},
			wantErr: false,
		},
		{
			name: "Path with hyphenated keys",
			path: "a-b.b[1].c:attr-b",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a-b", OriginalString: "a-b"},
					{SegmentType: KeySegment, Key: "b", OriginalString: ".b"},
					{SegmentType: IndexSegment, Index: 1, OriginalString: "[1]"},
					{SegmentType: KeySegment, Key: "c", OriginalString: ".c"},
					{SegmentType: AttributeSegment, Attribute: "attr-b", OriginalString: ":attr-b"},
				},
				Source: "a-b.b[1].c:attr-b",
			},
			wantErr: false,
		},
		{
			name: "Path with quoted key and attribute",
			path: "a[\"b\"].c:attr-c",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: KeySegment, Key: "b", OriginalString: "[\"b\"]"},
					{SegmentType: KeySegment, Key: "c", OriginalString: ".c"},
					{SegmentType: AttributeSegment, Attribute: "attr-c", OriginalString: ":attr-c"},
				},
				Source: "a[\"b\"].c:attr-c",
			},
			wantErr: false,
		},
		{
			name:     "Invalid path with mismatched quotes",
			path:     "a[\"b].c",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "Invalid path with non-numeric index",
			path:     "a[b].c",
			expected: nil,
			wantErr:  true,
		},
		{
			name: "Path with multiple indices",
			path: "a[1][2][3].b",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: IndexSegment, Index: 1, OriginalString: "[1]"},
					{SegmentType: IndexSegment, Index: 2, OriginalString: "[2]"},
					{SegmentType: IndexSegment, Index: 3, OriginalString: "[3]"},
					{SegmentType: KeySegment, Key: "b", OriginalString: ".b"},
				},
				Source: "a[1][2][3].b",
			},
			wantErr: false,
		},
		{
			name: "Path with spaces in quoted key",
			path: "a[\"b c\"].d",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
					{SegmentType: KeySegment, Key: "b c", OriginalString: "[\"b c\"]"},
					{SegmentType: KeySegment, Key: "d", OriginalString: ".d"},
				},
				Source: "a[\"b c\"].d",
			},
			wantErr: false,
		},
		{
			name: "Path with underscore in key",
			path: "a_b.c_d[1].e_f",
			expected: &Path{
				Segments: []*Segment{
					{SegmentType: KeySegment, Key: "a_b", OriginalString: "a_b"},
					{SegmentType: KeySegment, Key: "c_d", OriginalString: ".c_d"},
					{SegmentType: IndexSegment, Index: 1, OriginalString: "[1]"},
					{SegmentType: KeySegment, Key: "e_f", OriginalString: ".e_f"},
				},
				Source: "a_b.c_d[1].e_f",
			},
			wantErr: false,
		},
		{
			name:     "Path with numeric keys",
			path:     "0.1.2[3].4",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			got, err := NewPath(tt.path)
			if tt.wantErr {
				c.Assert(err, qt.IsNotNil)
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(got, qt.DeepEquals, tt.expected)
			}
		})
	}
}

func TestPath_TrimFirst(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name            string
		path            string
		expectedSeg     *Segment
		expectedNewPath string
		wantErr         bool
	}{
		{
			name:            "Simple key path",
			path:            "a.b.c",
			expectedSeg:     &Segment{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
			expectedNewPath: "b.c",
			wantErr:         false,
		},
		{
			name:            "Path with index",
			path:            "a[1].b.c",
			expectedSeg:     &Segment{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
			expectedNewPath: "[1].b.c",
			wantErr:         false,
		},
		{
			name:            "Path with single segment",
			path:            "a",
			expectedSeg:     &Segment{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
			expectedNewPath: "",
			wantErr:         false,
		},
		{
			name:            "Empty path",
			path:            "",
			expectedSeg:     nil,
			expectedNewPath: "",
			wantErr:         true,
		},
		{
			name:            "Path starting with index",
			path:            "[0].a.b",
			expectedSeg:     &Segment{SegmentType: IndexSegment, Index: 0, OriginalString: "[0]"},
			expectedNewPath: "a.b",
			wantErr:         false,
		},
		{
			name:            "Path with quoted key",
			path:            "a[\"b\"].c.d",
			expectedSeg:     &Segment{SegmentType: KeySegment, Key: "a", OriginalString: "a"},
			expectedNewPath: "[\"b\"].c.d",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			p, err := NewPath(tt.path)
			c.Assert(err, qt.IsNil)

			firstSeg, newP, err := p.TrimFirst()
			if tt.wantErr {
				c.Assert(err, qt.IsNotNil)
			} else {
				c.Assert(err, qt.IsNil)
				c.Assert(firstSeg, qt.DeepEquals, tt.expectedSeg)
				c.Assert(newP.String(), qt.Equals, tt.expectedNewPath)
				c.Assert(newP.Source, qt.Equals, tt.path)
			}
		})
	}
}

func TestPath_String(t *testing.T) {
	c := qt.New(t)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Simple key path",
			path:     "a.b.c",
			expected: "a.b.c",
		},
		{
			name:     "Path with index",
			path:     "a[1].b.c",
			expected: "a[1].b.c",
		},
		{
			name:     "Path with quoted key",
			path:     "a[\"b\"].c",
			expected: "a[\"b\"].c",
		},
		{
			name:     "Path with attribute",
			path:     "a.b.c:attr",
			expected: "a.b.c:attr",
		},
		{
			name:     "Path with multiple indices",
			path:     "a[1][2][3].b",
			expected: "a[1][2][3].b",
		},
		{
			name:     "Path with spaces in quoted key",
			path:     "a[\"b c\"].d",
			expected: "a[\"b c\"].d",
		},
		{
			name:     "Path with underscore in key",
			path:     "a_b.c_d[1].e_f",
			expected: "a_b.c_d[1].e_f",
		},
	}

	for _, tt := range tests {
		c.Run(tt.name, func(c *qt.C) {
			p, err := NewPath(tt.path)
			c.Assert(err, qt.IsNil)
			c.Assert(p.String(), qt.Equals, tt.expected)
			c.Assert(p.Source, qt.Equals, tt.path)
		})
	}
}
