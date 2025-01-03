package gen

import (
	"testing"

	"github.com/go-playground/validator/v10"

	qt "github.com/frankban/quicktest"
)

func TestObjectSchema_Validate(t *testing.T) {
	c := qt.New(t)

	validate := validator.New(validator.WithRequiredStructEnabled())

	zero, one, two := 0, 1, 2
	// Returns a valid struct
	validStruct := func() *objectSchema {
		return &objectSchema{
			Properties: map[string]property{
				"stringval": {
					Description: "a string",
					Title:       "String Value",
					Type:        "string",
					Order:       &zero,
				},
				"intval": {
					Description: "an integer number",
					Title:       "Integer value",
					Type:        "integer",
					Order:       &one,
				},
			},
			Required: []string{"a"},
			Title:    "Object Schema",
		}
	}

	c.Run("ok", func(c *qt.C) {
		err := validate.Struct(validStruct())
		c.Check(err, qt.IsNil)
	})

	testcases := []struct {
		name     string
		modifier func(*objectSchema)
		wantErr  string
	}{
		{
			name: "nok - no properties",
			modifier: func(rs *objectSchema) {
				rs.Properties = map[string]property{}
			},
			wantErr: "objectSchema.Properties: Properties field doesn't reach the minimum value / number of elements",
		},
		{
			name: "nok - no title",
			modifier: func(rs *objectSchema) {
				rs.Properties["wrong"] = property{
					Description: "foo",
					Type:        "zoot",
					Order:       &two,
				}
			},
			wantErr: `^objectSchema\.Properties\[wrong\]\.Title: Title field is required$`,
		},
		{
			name: "nok - no description",
			modifier: func(rs *objectSchema) {
				rs.Properties["wrong"] = property{
					Title: "bar",
					Type:  "zot",
					Order: &two,
				}
			},
			wantErr: `^objectSchema\.Properties\[wrong\]\.Description: Description field is required$`,
		},
		{
			name: "nok - no order",
			modifier: func(rs *objectSchema) {
				rs.Properties["wrong"] = property{
					Description: "foo",
					Title:       "bar",
					Type:        "zot",
				}
			},
			wantErr: `^objectSchema\.Properties\[wrong\]\.Order: Order field is required$`,
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
