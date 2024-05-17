package definitionupdater

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func Test_ShouldSkipUpsert(t *testing.T) {
	c := qt.New(t)

	v := "0.1.0-alpha"
	d := &pb.ConnectorDefinition{
		Id:      "my-conn",
		Version: v,
	}

	db := &datamodel.ComponentDefinition{
		Version: v,
	}

	testcases := []struct {
		name string

		def  definition
		inDB *datamodel.ComponentDefinition

		want    bool
		wantErr string
	}{
		{
			name: "don't skip - no record in DB",
			def:  d,
			want: false,
		},
		{
			name: "skip - same version",
			def:  d,
			inDB: db,
			want: true,
		},
		{
			name: "skip - older version",
			def:  d,
			inDB: &datamodel.ComponentDefinition{Version: "0.1.0-alpha.1"},
			want: true,
		},
		{
			name: "don't skip - newer version",
			def:  &pb.ConnectorDefinition{Id: "my-conn", Version: "0.1.0-alpha.1"},
			inDB: db,
			want: false,
		},
		{
			name: "don't skip - score change",
			def:  d,
			inDB: &datamodel.ComponentDefinition{Version: v, FeatureScore: 10},
			want: false,
		},
		{
			name:    "err - malformed version",
			def:     &pb.ConnectorDefinition{Id: "my-conn", Version: "v0.1.0-alpha"},
			inDB:    db,
			wantErr: "failed to parse version.*",
		},
		{
			name:    "err - malformed version in DB",
			def:     d,
			inDB:    &datamodel.ComponentDefinition{Version: "v0.1.0-alpha"},
			wantErr: "failed to parse version.*",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			got, err := shouldSkipUpsert(tc.def, tc.inDB)
			if tc.wantErr != "" {
				c.Check(err, qt.ErrorMatches, tc.wantErr)
				return
			}
			c.Check(err, qt.IsNil)
			c.Check(got, qt.Equals, tc.want)
		})
	}
}
