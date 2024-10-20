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
