package datamodel

import (
	"testing"

	"github.com/frankban/quicktest"
)

func TestDatamodel_TagNames(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		pipeline *Pipeline
		expected []string
	}{
		{
			pipeline: &Pipeline{
				Tags: []*Tag{
					{
						TagName: "tag1",
					},
					{
						TagName: "tag2",
					},
				},
			},
			expected: []string{"tag1", "tag2"},
		},
		{
			pipeline: &Pipeline{},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		tagNames := tc.pipeline.TagNames()
		c.Assert(tagNames, quicktest.DeepEquals, tc.expected)
	}
}
