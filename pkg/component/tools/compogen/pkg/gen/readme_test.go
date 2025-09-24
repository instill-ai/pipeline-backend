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

func TestEscapeCurlyBracesForReadme(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single curly braces should be escaped",
			in:   "Format: cachedContents/{cachedContent}",
			want: "Format: cachedContents/\\{cachedContent\\}",
		},
		{
			name: "template variables should be wrapped in backticks",
			in:   "Use {{variable_name}} in templates",
			want: "Use `{{variable_name}}` in templates",
		},
		{
			name: "JSON examples should be wrapped in backticks",
			in:   `This object should comply with the format {"tortor": "something", "arcu": "something else"}`,
			want: "This object should comply with the format `{\"tortor\": \"something\", \"arcu\": \"something else\"}`",
		},
		{
			name: "mixed JSON and variable placeholders",
			in:   `Use format {"key": "value"} with {placeholder}`,
			want: "Use format `{\"key\": \"value\"}` with \\{placeholder\\}",
		},
		{
			name: "already escaped braces should not be double-escaped",
			in:   "Already escaped \\{item\\} should stay",
			want: "Already escaped \\{item\\} should stay",
		},
		{
			name: "template variables already in backticks should not be re-processed",
			in:   "Already processed `{{var}}` template",
			want: "Already processed `{{var}}` template",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "no braces",
			in:   "Plain text without braces",
			want: "Plain text without braces",
		},
		{
			name: "multiple single braces",
			in:   "First {item1} and second {item2}",
			want: "First \\{item1\\} and second \\{item2\\}",
		},
		{
			name: "complex example from Gemini",
			in:   "The name of a cached content to use as context. Format: cachedContents/{cachedContent}.",
			want: "The name of a cached content to use as context. Format: cachedContents/\\{cachedContent\\}.",
		},
		{
			name: "SmartLead template variable explanation",
			in:   "You can use {{variable_name}} in your email templates",
			want: "You can use `{{variable_name}}` in your email templates",
		},
		{
			name: "hyphenated identifiers",
			in:   "Use {cache-name} for the cache identifier",
			want: "Use \\{cache-name\\} for the cache identifier",
		},
		{
			name: "underscore identifiers",
			in:   "Reference {snake_case} variables",
			want: "Reference \\{snake_case\\} variables",
		},
		{
			name: "JSON array with objects",
			in:   `Input: [{"a": 1}, {"b": 2}]`,
			want: "Input: `[{\"a\": 1}, {\"b\": 2}]`",
		},
		{
			name: "simple JSON object",
			in:   `Config: {"name": "John", "age": 25}`,
			want: "Config: `{\"name\": \"John\", \"age\": 25}`",
		},
		{
			name: "complex JSON with nested quotes and backticks should be wrapped",
			in:   "Format: {\"role\": \"The message role, i.e. `system`, `user` or `assistant`\", \"content\": \"message content\"}",
			want: "Format: `{\"role\": \"The message role, i.e. `system`, `user` or `assistant`\", \"content\": \"message content\"}`",
		},
		{
			name: "nested JSON objects should be wrapped",
			in:   "starts with {\"mappings\": {\"properties\"}} field",
			want: "starts with `{\"mappings\": {\"properties\"}}` field",
		},
	}

	for _, tc := range testcases {
		c.Run(tc.name, func(c *qt.C) {
			got := escapeCurlyBracesForReadme(tc.in)
			c.Check(got, qt.Equals, tc.want)
		})
	}
}
