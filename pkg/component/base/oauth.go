package base

// OAuthConnector contains the OAuth configuration that a comopnent can use to
// support OAuth 2.0 connections. Such components must have an
// `instillOAuthConfig` object in their setup definition.
type OAuthConnector struct {
	oAuthClientID     string
	oAuthClientSecret string
}

const (
	cfgOAuthClientID     = "oauth-client-id"
	cfgOAuthClientSecret = "oauth-client-secret"
)

// WithOAuthConfig loads the OAuth 2.0 connection details into the connector,
// which can be used to determine if the Instill AI deployment supports OAuth
// connections for a given component.
// TODO jvallesm: this is a prerequisite for supporting refresh token when the
// component execution uses an OAuth connection.
func (c *OAuthConnector) WithOAuthConfig(s map[string]any) {
	c.oAuthClientID = ReadFromGlobalConfig(cfgOAuthClientID, s)
	c.oAuthClientSecret = ReadFromGlobalConfig(cfgOAuthClientSecret, s)
}

// SupportsOAuth checks whether the connector is configured to support OAuth.
func (c *OAuthConnector) SupportsOAuth() bool {
	return c.oAuthClientID != "" && c.oAuthClientSecret != ""
}

// GetOAuthClientID returns the OAuth client ID.
func (c *OAuthConnector) GetOAuthClientID() string {
	return c.oAuthClientID
}

// GetOAuthClientSecret returns the OAuth client secret.
func (c *OAuthConnector) GetOAuthClientSecret() string {
	return c.oAuthClientSecret
}
