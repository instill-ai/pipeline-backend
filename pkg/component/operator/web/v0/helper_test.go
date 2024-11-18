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
		filter   filter
		expected bool
	}{
		// Test case for filter combination
		{
			name:     "no filter",
			link:     "https://www.example.com",
			filter:   filter{},
			expected: true,
		},
		{
			name: "include pattern match",
			link: "https://www.example.com",
			filter: filter{
				IncludePattern: "example.com",
			},
			expected: true,
		},
		{
			name: "include pattern not match",
			link: "https://www.example.co",
			filter: filter{
				IncludePattern: "example.com",
			},
			expected: false,
		},
		{
			name: "exclude pattern match",
			link: "https://www.example.com",
			filter: filter{
				ExcludePattern: "example.com",
			},
			expected: false,
		},
		{
			name: "exclude pattern not match",
			link: "https://www.example.com",
			filter: filter{
				ExcludePattern: "example.cos",
			},
			expected: true,
		},
		{
			name: "include pattern match and exclude pattern not match",
			link: "https://www.example.com",
			filter: filter{
				IncludePattern: "example.com",
				ExcludePattern: "example.cos",
			},
			expected: true,
		},
		{
			name: "include pattern not match and exclude pattern match",
			link: "https://www.example.co",
			filter: filter{
				IncludePattern: "example.com",
				ExcludePattern: "example.co",
			},
			expected: false,
		},
		{
			name: "include and exclude pattern both not match",
			link: "https://www.example.c",
			filter: filter{
				IncludePattern: "example.com",
				ExcludePattern: "example.co",
			},
			expected: false,
		},
		{
			name: "include and exclude pattern both match",
			link: "https://www.example.com",
			filter: filter{
				IncludePattern: "example.com",
				ExcludePattern: "example.co",
			},
			expected: false,
		},
		// Test case for regex match. There are only some test cases here. It should be well tested in regexp package.
		{
			name: "digit match",
			link: "https://example1.com",
			filter: filter{
				IncludePattern: "example[\\d].com",
			},
			expected: true,
		},
		{
			name: "disjunction match",
			link: "https://exampleA.com",
			filter: filter{
				IncludePattern: "example[A|B|C].com",
			},
			expected: true,
		},
		{
			name: "match all subdomains of example.com",
			link: "https://blog.example.com",
			filter: filter{
				IncludePattern: ".*\\.example\\.com",
			},
			expected: true,
		},
		{
			name: "match specific file extensions",
			link: "https://example.com/document.pdf",
			filter: filter{
				IncludePattern: ".*\\.(pdf|doc|docx)$",
			},
			expected: true,
		},
		{
			name: "match specific keywords in path",
			link: "https://example.com/blog/post-1",
			filter: filter{
				IncludePattern: ".*(blog|news|article).*",
			},
			expected: true,
		},
		{
			name: "match specific ports",
			link: "https://example.com:8080/api",
			filter: filter{
				IncludePattern: ".*:(8080|8443)($|/.*)",
			},
			expected: true,
		},
		{
			name: "match https only",
			link: "https://example.com",
			filter: filter{
				IncludePattern: "^https://.*",
			},
			expected: true,
		},
		{
			name: "exclude http protocol",
			link: "http://example.com",
			filter: filter{
				IncludePattern: "^https://.*",
			},
			expected: false,
		},
		{
			name: "match specific country TLDs",
			link: "https://example.uk",
			filter: filter{
				IncludePattern: ".*\\.(uk|fr|de)$",
			},
			expected: true,
		},
		{
			name: "match URLs without query parameters",
			link: "https://example.com/path",
			filter: filter{
				IncludePattern: "^[^?]*$",
			},
			expected: true,
		},
		{
			name: "not match URLs without query parameters",
			link: "https://example.com/path?id=123",
			filter: filter{
				IncludePattern: "^[^?]*$",
			},
			expected: false,
		},
		{
			name: "match specific query parameters",
			link: "https://example.com/path?id=123",
			filter: filter{
				IncludePattern: ".*[?&]id=[0-9]+.*",
			},
			expected: true,
		},
		// Add negative test cases
		{
			name: "non-matching file extension",
			link: "https://example.com/document.txt",
			filter: filter{
				IncludePattern: ".*\\.(pdf|doc|docx)$",
			},
			expected: false,
		},
		{
			name: "non-matching country TLD",
			link: "https://example.us",
			filter: filter{
				IncludePattern: ".*\\.(uk|fr|de)$",
			},
			expected: false,
		},
	}

	for _, testCase := range testCases {
		c.Run(testCase.name, func(c *quicktest.C) {
			err := testCase.filter.compile()
			c.Assert(err, quicktest.IsNil)
			c.Assert(targetLink(testCase.link, testCase.filter), quicktest.Equals, testCase.expected)
		})
	}

}
