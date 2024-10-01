package gen

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"

	qt "github.com/frankban/quicktest"
)

func TestDefinition_Validate(t *testing.T) {
	c := qt.New(t)

	validate := validator.New(validator.WithRequiredStructEnabled())

	// Returns a valid struct
	validStruct := func() *definition {
		return &definition{
			ID:             "foo",
			Title:          "Foo",
			Description:    "Foo bar",
			Public:         false,
			ReleaseStage:   3,
			AvailableTasks: []string{"TASK_1", "TASK_2"},
			SourceURL:      "https://github.com/instill-ai",
		}
	}

	c.Run("ok", func(c *qt.C) {
		err := validate.Struct(validStruct())
		c.Check(err, qt.IsNil)
	})

	testcases := []struct {
		name     string
		modifier func(*definition)
		wantErr  string
	}{
		{
			name: "nok - no ID",
			modifier: func(d *definition) {
				d.ID = ""
			},
			wantErr: "definition.ID: ID field is required",
		},
		{
			name: "nok - no title",
			modifier: func(d *definition) {
				d.Title = ""
			},
			wantErr: "definition.Title: Title field is required",
		},
		{
			name: "nok - no description",
			modifier: func(d *definition) {
				d.Description = ""
			},
			wantErr: "definition.Description: Description field is required",
		},
		{
			name: "nok - no release stage",
			modifier: func(d *definition) {
				d.ReleaseStage = 0
			},
			wantErr: "definition.ReleaseStage: ReleaseStage field is required",
		},
		{
			name: "nok - zero tasks",
			modifier: func(d *definition) {
				d.AvailableTasks = []string{}
			},
			wantErr: "definition.AvailableTasks: AvailableTasks field doesn't reach the minimum value / number of elements",
		},
		{
			name: "nok - invalid source URL",
			modifier: func(d *definition) {
				d.SourceURL = "github.com/instill-ai"
			},
			wantErr: "definition.SourceURL: SourceURL field must be a valid URL",
		},
		{
			name: "nok - multiple errors",
			modifier: func(d *definition) {
				d.Title = ""
				d.Description = ""
			},
			wantErr: "definition.Title: Title field is required\ndefinition.Description: Description field is required",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			got := validStruct()
			tc.modifier(got)

			err := validate.Struct(got)
			c.Check(err, qt.IsNotNil)
			c.Check(asValidationError(err), qt.ErrorMatches, tc.wantErr)
		})
	}
}

func TestReleaseStage_UnmarshalAndStringify(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		in   string
		want string
	}{
		{in: "OTHER", want: "Unspecified"},
		{in: "RELEASE_STAGE_OPEN_FOR_CONTRIBUTION", want: "Open For Contribution"},
		{in: "RELEASE_STAGE_COMING_SOON", want: "Coming Soon"},
		{in: "RELEASE_STAGE_ALPHA", want: "Alpha"},
		{in: "RELEASE_STAGE_BETA", want: "Beta"},
		{in: "RELEASE_STAGE_GA", want: "GA"},
	}

	for _, tc := range testcases {
		c.Run(tc.in, func(c *qt.C) {
			def := definition{}
			j := json.RawMessage(`{"releaseStage": "` + tc.in + `"}`)

			err := json.Unmarshal(j, &def)
			c.Check(err, qt.IsNil)
			c.Check(def.ReleaseStage.String(), qt.Equals, tc.want)
		})
	}
}
