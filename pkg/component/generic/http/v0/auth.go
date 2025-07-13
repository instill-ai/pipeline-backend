package http

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"

	errorsx "github.com/instill-ai/x/errors"
)

type authType string

const (
	noAuthType      authType = "NO_AUTH"
	basicAuthType   authType = "BASIC_AUTH"
	apiKeyType      authType = "API_KEY"
	bearerTokenType authType = "BEARER_TOKEN"
)

type authentication interface {
	setAuthInClient(c *httpclient.Client) error
}

type noAuth struct {
	AuthType authType `json:"auth-type"`
}

func (a noAuth) setAuthInClient(_ *httpclient.Client) error {
	return nil
}

type basicAuth struct {
	AuthType authType `json:"auth-type"`
	Username string   `json:"username"`
	Password string   `json:"password"`
}

func (a basicAuth) setAuthInClient(c *httpclient.Client) error {
	if a.Username == "" || a.Password == "" {
		return errorsx.AddMessage(
			fmt.Errorf("invalid auth"),
			"Basic Auth error: username or password is empty.",
		)
	}

	c.SetBasicAuth(a.Username, a.Password)

	return nil
}

type authLocation string

const (
	header authLocation = "header"
	query  authLocation = "query"
)

type apiKeyAuth struct {
	AuthType     authType     `json:"auth-type"`
	Key          string       `json:"key"`
	Value        string       `json:"value"`
	AuthLocation authLocation `json:"auth-location"`
}

func (a apiKeyAuth) setAuthInClient(c *httpclient.Client) error {
	if a.Key == "" || a.Value == "" {
		return errorsx.AddMessage(
			fmt.Errorf("invalid auth"),
			"API Key Auth error: key or value is empty.",
		)
	}

	if a.AuthLocation == header {
		c.SetHeader(a.Key, a.Value)
		return nil
	}

	c.SetQueryParam(a.Key, a.Value)

	return nil
}

type bearerTokenAuth struct {
	AuthType authType `json:"auth-type"`
	Token    string   `json:"token"`
}

func (a bearerTokenAuth) setAuthInClient(c *httpclient.Client) error {
	if a.Token == "" {
		return errorsx.AddMessage(
			fmt.Errorf("invalid auth"),
			"Bearer Token Auth error: token is empty.",
		)
	}

	c.SetAuthToken(a.Token)

	return nil
}
