package http

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

	"github.com/instill-ai/pipeline-backend/config"
	errorsx "github.com/instill-ai/x/errors"
)

// URLValidator defines the interface for validating HTTP input
type URLValidator interface {
	ValidateInput(input *httpInput) error
}

// urlValidator provides common validation logic
type urlValidator struct {
	whitelistedEndpoints []string
	allowLocalhost       bool
	allowPrivateIPs      bool
}

// NewURLValidator creates a validator for production use
func NewURLValidator() URLValidator {
	return &urlValidator{allowPrivateIPs: false}
}

// ValidateInput implements the consolidated validation logic
func (v *urlValidator) ValidateInput(input *httpInput) error {
	if input == nil {
		return fmt.Errorf("input cannot be nil")
	}

	endpointURL := input.EndpointURL
	if endpointURL == "" {
		return errorsx.AddMessage(
			fmt.Errorf("endpoint URL is required"),
			"Endpoint URL must be provided",
		)
	}

	parsedURL, err := url.Parse(endpointURL)
	if err != nil {
		return errorsx.AddMessage(
			fmt.Errorf("parsing endpoint URL: %w", err),
			"Couldn't parse the endpoint URL as a valid URI reference",
		)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return errorsx.AddMessage(
			fmt.Errorf("missing hostname"),
			"Endpoint URL must have a hostname",
		)
	}

	// Check explicit whitelist first
	for _, allowedEndpoint := range v.whitelistedEndpoints {
		if strings.HasPrefix(endpointURL, allowedEndpoint) {
			return nil
		}
	}

	// Production-specific whitelisted hosts
	if !v.allowPrivateIPs {
		prodWhitelist := []string{
			// Pipeline's public port is exposed to call pipelines from pipelines.
			// When a `pipeline` component is implemented, this won't be necessary.
			fmt.Sprintf("%s:%d", config.Config.Server.InstanceID, config.Config.Server.PublicPort),
			// Model's public port is exposed until the model component allows
			// triggering models in the custom mode.
			fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort),
		}
		// Certain pipelines used by artifact-backend need to trigger pipelines and
		// models via this component.
		// TODO jvallesm: Remove this after INS-8119 is completed.
		if slices.Contains(prodWhitelist, parsedURL.Host) {
			return nil
		}
	}

	// Check localhost allowance
	if v.allowLocalhost && (host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "127.")) {
		return nil
	}

	// Production mode: check if IP is private and block private IPs
	if !v.allowPrivateIPs {
		ips, err := net.LookupIP(host)
		if err != nil {
			return fmt.Errorf("looking up IP: %w", err)
		}

		for _, ip := range ips {
			if ip.IsPrivate() || ip.IsLoopback() {
				return errorsx.AddMessage(
					fmt.Errorf("endpoint URL resolves to private/internal IP address"),
					"URL must point to a publicly available endpoint (no private/internal addresses)",
				)
			}
		}
		// Production mode: allow public IPs
		return nil
	}

	// Test mode: apply whitelist and localhost restrictions
	if len(v.whitelistedEndpoints) > 0 {
		return errorsx.AddMessage(
			fmt.Errorf("endpoint URL not in test whitelist"),
			fmt.Sprintf("URL %s is not in the whitelisted endpoints for testing", endpointURL),
		)
	}

	if !v.allowLocalhost {
		return errorsx.AddMessage(
			fmt.Errorf("endpoint URL not allowed"),
			"No endpoints are whitelisted for testing and localhost access is disabled",
		)
	}

	// Test mode: allow if localhost is enabled
	return nil
}
