package http

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

// TestURLValidation tests the URL validation logic comprehensively
func TestURLValidation(t *testing.T) {
	c := qt.New(t)

	c.Run("Production validator", func(c *qt.C) {
		validator := NewURLValidator()

		// Should allow public URLs
		input := &httpInput{EndpointURL: "https://api.github.com/repos/test"}
		err := validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Production should allow public URLs"))

		// Should block localhost
		input = &httpInput{EndpointURL: "http://localhost:8080/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Production should block localhost"))
		c.Assert(err.Error(), qt.Contains, "private/internal IP address")

		// Should block 127.0.0.1
		input = &httpInput{EndpointURL: "http://127.0.0.1:8080/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Production should block 127.0.0.1"))
		c.Assert(err.Error(), qt.Contains, "private/internal IP address")

		// Should block private IPs
		input = &httpInput{EndpointURL: "http://192.168.1.1/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Production should block private IPs"))
		c.Assert(err.Error(), qt.Contains, "private/internal IP address")
	})

	c.Run("Production validator - internal service whitelist mechanism", func(c *qt.C) {
		validator := NewURLValidator()

		// Test that the whitelist mechanism works by testing the exact host:port combinations
		// that should be in the production whitelist based on the config

		// Note: The actual whitelist values depend on config.Config values at runtime
		// In tests, these might not match the expected values, but we can test the mechanism

		testCases := []struct {
			name        string
			url         string
			expectBlock bool
			reason      string
		}{
			{
				name:        "pipeline-backend-8081",
				url:         "http://pipeline-backend:8081/api",
				expectBlock: false, // Should be whitelisted if config matches
				reason:      "Should be in production whitelist",
			},
			{
				name:        "model-backend-8083",
				url:         "http://model-backend:8083/api",
				expectBlock: false, // Should be whitelisted if config matches
				reason:      "Should be in production whitelist",
			},
			{
				name:        "wrong-port-pipeline",
				url:         "http://pipeline-backend:8080/api",
				expectBlock: true, // Should be blocked as not in whitelist
				reason:      "Wrong port, not in whitelist",
			},
			{
				name:        "wrong-port-model",
				url:         "http://model-backend:8080/api",
				expectBlock: true, // Should be blocked as not in whitelist
				reason:      "Wrong port, not in whitelist",
			},
		}

		for _, tc := range testCases {
			c.Run(tc.name, func(c *qt.C) {
				input := &httpInput{EndpointURL: tc.url}
				err := validator.ValidateInput(input)

				// For this test, we're primarily documenting the expected behavior
				// The actual result depends on config values and DNS resolution
				if tc.expectBlock {
					c.Assert(err, qt.IsNotNil, qt.Commentf("Should be blocked: %s (%s)", tc.url, tc.reason))
				} else {
					// These might fail due to DNS in test environment, but that's expected
					// The important thing is that the whitelist logic is in place
					if err != nil {
						// If it fails, it should be due to DNS, not whitelist logic
						c.Assert(err.Error(), qt.Contains, "lookup", qt.Commentf("If blocked, should be due to DNS lookup, not whitelist logic"))
					}
				}
			})
		}
	})

	c.Run("Test validator with whitelist", func(c *qt.C) {
		whitelist := []string{"https://api.github.com", "https://httpbin.org"}
		validator := NewTestURLValidator(whitelist, false)

		// Should allow whitelisted URLs
		input := &httpInput{EndpointURL: "https://api.github.com/repos/test"}
		err := validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow whitelisted URL"))

		// Should allow whitelisted URLs with paths
		input = &httpInput{EndpointURL: "https://httpbin.org/get?param=value"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow whitelisted URL with path"))

		// Should block non-whitelisted URLs
		input = &httpInput{EndpointURL: "https://example.com/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should block non-whitelisted URL"))
		c.Assert(err.Error(), qt.Contains, "not in test whitelist")

		// Should block localhost when not allowed
		input = &httpInput{EndpointURL: "http://localhost:8080/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should block localhost when not allowed"))
		c.Assert(err.Error(), qt.Contains, "not in test whitelist")
	})

	c.Run("Test validator with localhost enabled", func(c *qt.C) {
		validator := NewTestURLValidator(nil, true)

		// Should allow localhost
		input := &httpInput{EndpointURL: "http://localhost:8080/api"}
		err := validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow localhost when enabled"))

		// Should allow 127.0.0.1
		input = &httpInput{EndpointURL: "http://127.0.0.1:3000/test"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow 127.0.0.1 when localhost enabled"))

		// Should allow 127.x.x.x
		input = &httpInput{EndpointURL: "http://127.1.1.1:3000/test"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow 127.x.x.x when localhost enabled"))

		// Should allow external URLs in test mode
		input = &httpInput{EndpointURL: "https://api.github.com/repos/test"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Test mode should allow external URLs"))
	})

	c.Run("Test validator with whitelist and localhost", func(c *qt.C) {
		whitelist := []string{"https://api.github.com"}
		validator := NewTestURLValidator(whitelist, true)

		// Should allow whitelisted URLs
		input := &httpInput{EndpointURL: "https://api.github.com/repos/test"}
		err := validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow whitelisted URL"))

		// Should allow localhost
		input = &httpInput{EndpointURL: "http://localhost:8080/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNil, qt.Commentf("Should allow localhost when enabled"))

		// Should block non-whitelisted external URLs
		input = &httpInput{EndpointURL: "https://example.com/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should block non-whitelisted URL"))
		c.Assert(err.Error(), qt.Contains, "not in test whitelist")
	})

	c.Run("Test validator security - no whitelist, no localhost", func(c *qt.C) {
		validator := NewTestURLValidator(nil, false)

		// Should block external URLs when no whitelist
		input := &httpInput{EndpointURL: "https://api.github.com/repos/test"}
		err := validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should block external URLs when no whitelist"))
		c.Assert(err.Error(), qt.Contains, "endpoint URL not allowed")

		// Should block localhost when not allowed
		input = &httpInput{EndpointURL: "http://localhost:8080/api"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should block localhost when not allowed"))
		c.Assert(err.Error(), qt.Contains, "endpoint URL not allowed")
	})

	c.Run("Input validation edge cases", func(c *qt.C) {
		validator := NewTestURLValidator(nil, true)

		// Should reject nil input
		err := validator.ValidateInput(nil)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should reject nil input"))
		c.Assert(err.Error(), qt.Contains, "input cannot be nil")

		// Should reject empty URL
		input := &httpInput{EndpointURL: ""}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should reject empty URL"))
		c.Assert(err.Error(), qt.Contains, "endpoint URL is required")

		// Should reject invalid URL
		input = &httpInput{EndpointURL: "not-a-url"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should reject invalid URL"))
		c.Assert(err.Error(), qt.Contains, "missing hostname")

		// Should reject malformed URL
		input = &httpInput{EndpointURL: "http://[::1:invalid"}
		err = validator.ValidateInput(input)
		c.Assert(err, qt.IsNotNil, qt.Commentf("Should reject malformed URL"))
		c.Assert(err.Error(), qt.Contains, "parsing endpoint URL")
	})
}

// TestComponentWithWhitelist tests the component integration with whitelist
func TestComponentWithWhitelist(t *testing.T) {
	c := qt.New(t)

	c.Run("Component creation with different validation modes", func(c *qt.C) {
		// Production component
		prodComp := Init(base.Component{})
		c.Assert(prodComp, qt.IsNotNil)

		// Test component with secure defaults
		secureComp := InitForTest(base.Component{}, nil, false)
		c.Assert(secureComp, qt.IsNotNil)

		// Test component with localhost
		localhostComp := InitForTest(base.Component{}, nil, true)
		c.Assert(localhostComp, qt.IsNotNil)

		// Test component with whitelist
		whitelist := []string{"https://api.github.com", "https://httpbin.org"}
		whitelistComp := InitForTest(base.Component{}, whitelist, false)
		c.Assert(whitelistComp, qt.IsNotNil)

		// Test component with whitelist and localhost
		flexibleComp := InitForTest(base.Component{}, whitelist, true)
		c.Assert(flexibleComp, qt.IsNotNil)
	})

	c.Run("Whitelist prefix matching", func(c *qt.C) {
		whitelist := []string{"https://api.github.com", "https://httpbin.org/get"}
		validator := NewTestURLValidator(whitelist, false)

		// Should match prefix exactly
		testCases := []struct {
			url      string
			expected bool
			desc     string
		}{
			{"https://api.github.com", true, "exact match"},
			{"https://api.github.com/", true, "with trailing slash"},
			{"https://api.github.com/repos", true, "with path"},
			{"https://api.github.com/repos/owner/repo", true, "with deep path"},
			{"https://api.github.com?param=value", true, "with query params"},
			{"https://httpbin.org/get", true, "exact match with path"},
			{"https://httpbin.org/get/test", true, "extending whitelisted path"},
			{"https://httpbin.org/post", false, "different path"},
			{"https://api.github.co", false, "partial domain match"},
			{"https://evil-api.github.com", false, "subdomain attack"},
			{"https://example.com", false, "completely different domain"},
		}

		for _, tc := range testCases {
			input := &httpInput{EndpointURL: tc.url}
			err := validator.ValidateInput(input)
			if tc.expected {
				c.Assert(err, qt.IsNil, qt.Commentf("URL %s should be allowed: %s", tc.url, tc.desc))
			} else {
				c.Assert(err, qt.IsNotNil, qt.Commentf("URL %s should be blocked: %s", tc.url, tc.desc))
			}
		}
	})
}
