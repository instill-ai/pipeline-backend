package web

import (
	"testing"

	"github.com/frankban/quicktest"
)

// TestTargetLink tests the targetLink function
func TestTargetLink(t *testing.T) {
	c := quicktest.New(t)

	testCases := []struct {
		name string

		link     string
		filter   Filter
		expected bool
	}{
		// Test case for filter combination
		{
			name:     "no filter",
			link:     "https://www.example.com",
			filter:   Filter{},
			expected: true,
		},
		{
			name: "include pattern match",
			link: "https://www.example.com",
			filter: Filter{
				IncludePatterns: []string{"example.com"},
			},
			expected: true,
		},
		{
			name: "include pattern not match",
			link: "https://www.example.co",
			filter: Filter{
				IncludePatterns: []string{"example.com"},
			},
			expected: false,
		},
		{
			name: "exclude pattern match",
			link: "https://www.example.com",
			filter: Filter{
				ExcludePatterns: []string{"example.com"},
			},
			expected: false,
		},
		{
			name: "exclude pattern not match",
			link: "https://www.example.com",
			filter: Filter{
				ExcludePatterns: []string{"example.cos"},
			},
			expected: true,
		},
		{
			name: "include pattern match and exclude pattern not match",
			link: "https://www.example.com",
			filter: Filter{
				IncludePatterns: []string{"example.com"},
				ExcludePatterns: []string{"example.cos"},
			},
			expected: true,
		},
		{
			name: "include pattern not match and exclude pattern match",
			link: "https://www.example.co",
			filter: Filter{
				IncludePatterns: []string{"example.com"},
				ExcludePatterns: []string{"example.co"},
			},
			expected: false,
		},
		{
			name: "include and exclude pattern both not match",
			link: "https://www.example.c",
			filter: Filter{
				IncludePatterns: []string{"example.com"},
				ExcludePatterns: []string{"example.co"},
			},
			expected: false,
		},
		{
			name: "include and exclude pattern both match",
			link: "https://www.example.com",
			filter: Filter{
				IncludePatterns: []string{"example.com"},
				ExcludePatterns: []string{"example.co"},
			},
			expected: false,
		},
		// Test case for regex match. There are only some test cases here. It should be well tested in regexp package.
		{
			name: "digit match",
			link: "https://example1.com",
			filter: Filter{
				IncludePatterns: []string{"example[\\d].com"},
			},
			expected: true,
		},
		{
			name: "disjunction match",
			link: "https://exampleA.com",
			filter: Filter{
				IncludePatterns: []string{"example[A|B|C].com"},
			},
			expected: true,
		},
		{
			name: "match all subdomains of example.com",
			link: "https://blog.example.com",
			filter: Filter{
				IncludePatterns: []string{".*\\.example\\.com"},
			},
			expected: true,
		},
		{
			name: "match specific file extensions",
			link: "https://example.com/document.pdf",
			filter: Filter{
				IncludePatterns: []string{".*\\.(pdf|doc|docx)$"},
			},
			expected: true,
		},
		{
			name: "match specific keywords in path",
			link: "https://example.com/blog/post-1",
			filter: Filter{
				IncludePatterns: []string{".*(blog|news|article).*"},
			},
			expected: true,
		},
		{
			name: "match specific ports",
			link: "https://example.com:8080/api",
			filter: Filter{
				IncludePatterns: []string{".*:(8080|8443)($|/.*)"},
			},
			expected: true,
		},
		{
			name: "match https only",
			link: "https://example.com",
			filter: Filter{
				IncludePatterns: []string{"^https://.*"},
			},
			expected: true,
		},
		{
			name: "exclude http protocol",
			link: "http://example.com",
			filter: Filter{
				IncludePatterns: []string{"^https://.*"},
			},
			expected: false,
		},
		{
			name: "match specific country TLDs",
			link: "https://example.uk",
			filter: Filter{
				IncludePatterns: []string{".*\\.(uk|fr|de)$"},
			},
			expected: true,
		},
		{
			name: "match URLs without query parameters",
			link: "https://example.com/path",
			filter: Filter{
				IncludePatterns: []string{"^[^?]*$"},
			},
			expected: true,
		},
		{
			name: "exclude URLs with query parameters",
			link: "https://example.com/path?id=123",
			filter: Filter{
				IncludePatterns: []string{"^[^?]*$"},
			},
			expected: false,
		},
		{
			name: "match specific query parameters",
			link: "https://example.com/path?id=123",
			filter: Filter{
				IncludePatterns: []string{".*[?&]id=[0-9]+.*"},
			},
			expected: true,
		},
		// Add negative test cases
		{
			name: "non-matching file extension",
			link: "https://example.com/document.txt",
			filter: Filter{
				IncludePatterns: []string{".*\\.(pdf|doc|docx)$"},
			},
			expected: false,
		},
		{
			name: "non-matching country TLD",
			link: "https://example.us",
			filter: Filter{
				IncludePatterns: []string{".*\\.(uk|fr|de)$"},
			},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		c.Run(testCase.name, func(c *quicktest.C) {
			c.Assert(targetLink(testCase.link, testCase.filter), quicktest.Equals, testCase.expected)
		})
	}

}
