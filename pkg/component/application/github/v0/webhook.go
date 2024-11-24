package github

import (
	"github.com/google/go-github/v62/github"
)

type HookConfig struct {
	URL         string `json:"url"`
	InsecureSSL string `json:"insecure-ssl"`
	Secret      string `json:"secret,omitempty"`
	ContentType string `json:"content-type"`
}

type HookInfo struct {
	ID      int64      `json:"id"`
	URL     string     `json:"url"`
	PingURL string     `json:"ping-url"`
	TestURL string     `json:"test-url"`
	Config  HookConfig `json:"config"`
}

func (client *Client) extractHook(originalHook *github.Hook) HookInfo {
	return HookInfo{
		ID:      originalHook.GetID(),
		URL:     originalHook.GetURL(),
		PingURL: originalHook.GetPingURL(),
		TestURL: originalHook.GetTestURL(),
		Config: HookConfig{
			URL:         originalHook.GetConfig().GetURL(),
			InsecureSSL: originalHook.GetConfig().GetInsecureSSL(),
			Secret:      originalHook.GetConfig().GetSecret(),
			ContentType: originalHook.GetConfig().GetContentType(),
		},
	}
}
