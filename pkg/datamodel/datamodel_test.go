package datamodel

import (
	"testing"

	"github.com/frankban/quicktest"
)

func TestDatamodel_TagNames(t *testing.T) {
	c := quicktest.New(t)
	testPipeline := &Pipeline{
		Tags: []*Tag{
			{
				TagName: "tag1",
			},
			{
				TagName: "tag2",
			},
		},
	}
	tagNames := testPipeline.TagNames()
	c.Assert(tagNames, quicktest.DeepEquals, []string{"tag1", "tag2"})

}
