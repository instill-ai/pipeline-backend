package gen

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestFirstToLower(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		in   string
		want string
	}{
		{in: "Hello world!", want: "hello world!"},
		{in: "hello world!", want: "hello world!"},
	}

	for _, tc := range testcases {
		c.Run(tc.in, func(c *qt.C) {
			got := firstToLower(tc.in)
			c.Check(got, qt.Equals, tc.want)
		})
	}
}

func TestComponentType_IndefiniteArticle(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		in   ComponentType
		want string
	}{
		{in: cstOperator, want: "an"},
		{in: cstAI, want: "an"},
		{in: cstApplication, want: "an"},
		{in: cstData, want: "a"},
	}

	for _, tc := range testcases {
		c.Run(string(tc.in), func(c *qt.C) {
			c.Check(tc.in.IndefiniteArticle(), qt.Equals, tc.want)
		})
	}
}

func TestTitleCase(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		in   string
		want string
	}{
		{in: "the quick brown fox jumps over a lazy dog", want: "The Quick Brown Fox Jumps over a Lazy Dog"},
		// Dashes are respected.
		{in: "One-Time password", want: "One-Time Password"},
		// Conjunctions are lowercase, acronyms are uppercase.
		{in: "Number Of pr comments", want: "Number of PR Comments"},
		// lowercase words are upcased at the beginning of the title.
		{in: "A single sign", want: "A Single Sign"},
		// lowercase words are upcased at the end of the title.
		{in: "Updated At", want: "Updated At"},
		// Pluralized acronyms are respected.
		{in: "Agent IDs", want: "Agent IDs"},
		// Respect special words.
		{in: "Bot OAuth Token", want: "Bot OAuth Token"},
	}

	for _, tc := range testcases {
		c.Run(tc.in, func(c *qt.C) {
			got := titleCase(tc.in)
			c.Check(got, qt.Equals, tc.want)
		})
	}
}

func TestParseObjectPropertiesInto_NormalizesSchemaTitle(t *testing.T) {
	c := qt.New(t)

	rt := &readmeTask{
		SignatureToCanonical:  map[string]string{},
		FieldKeyToCanonical:   map[string]string{},
		CanonicalToParents:    map[string]map[string]bool{},
		CanonicalToParentsIn:  map[string]map[string]bool{},
		CanonicalToParentsOut: map[string]map[string]bool{},
		CanonicalToParentMeta: map[string]map[string]parentMeta{},
		RootKeys:              map[string]bool{},
		parseContext:          "input",
	}

	properties := map[string]property{
		"parameters": {
			Title:       "Parameters",
			Type:        "object",
			Description: "A schema-like object",
			Properties: map[string]property{
				"type":        {Title: "Type", Type: "string"},
				"properties":  {Title: "Properties", Type: "object"},
				"required":    {Title: "Required", Type: "array"},
				"items":       {Title: "Items", Type: "object"},
				"format":      {Title: "Format", Type: "string"},
				"description": {Title: "Description", Type: "string"},
			},
		},
	}

	rt.parseObjectPropertiesInto(properties, &rt.InputObjects, "")

	// Expect one object captured with key "parameters" but Title normalized to "Schema"
	c.Assert(len(rt.InputObjects) > 0, qt.IsTrue)
	found := false
	for _, m := range rt.InputObjects {
		if obj, ok := m["parameters"]; ok {
			found = true
			c.Check(obj.Title, qt.Equals, "Schema")
		}
	}
	c.Check(found, qt.IsTrue)
}

func TestGetParents_ReturnsPropertyKeyForBacklink(t *testing.T) {
	c := qt.New(t)

	// Child key and parent container
	childKey := "time-range-filter"
	parentContainer := "web"

	rt := readmeTask{
		FieldKeyToCanonical:   map[string]string{},
		CanonicalToParentMeta: map[string]map[string]parentMeta{childKey: {parentContainer: {IsRoot: true}}},
		AllObjects: []map[string]objectSchema{
			{
				parentContainer: {
					Title: parentContainer,
					Properties: map[string]property{
						childKey: {Title: "Time Range Filter", Type: "object", Properties: map[string]property{"start-time": {Title: "Start Time"}}},
					},
				},
			},
		},
	}

	parents := getParents(childKey, rt, "")
	c.Assert(len(parents), qt.Equals, 1)
	c.Check(parents[0], qt.Equals, childKey)
}

func TestDedupeObjects_DifferentTitlesNotMerged(t *testing.T) {
	c := qt.New(t)

	rt := &readmeTask{
		RootKeys: map[string]bool{},
	}
	props := map[string]property{"a": {Title: "A", Type: "string"}}
	src := []map[string]objectSchema{
		{"top-candidates": {Title: "Top Candidates", Properties: props}},
		{"dynamic-retrieval-config": {Title: "Dynamic Retrieval Config", Properties: props}},
	}
	out := rt.dedupeObjects(src)
	c.Assert(len(out), qt.Equals, 2)
	// Ensure both keys are present
	seen := map[string]bool{}
	for _, m := range out {
		for k := range m {
			seen[k] = true
		}
	}
	c.Check(seen["top-candidates"], qt.IsTrue)
	c.Check(seen["dynamic-retrieval-config"], qt.IsTrue)
}
