package gen

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestFirstToLower(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		in   string
		mod  func(rune) rune
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

func TestTitleCaseWithArticles(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		in   string
		mod  func(rune) rune
		want string
	}{
		{in: "the-quick_brown fox jumps-over a-lazy_dog", want: "The Quick Brown Fox Jumps Over a Lazy Dog"},
	}

	for _, tc := range testcases {
		c.Run(tc.in, func(c *qt.C) {
			got := titleCaseWithArticles(tc.in)
			c.Check(got, qt.Equals, tc.want)
		})
	}
}
