package github

import (
	"github.com/google/go-github/v62/github"
)

type HookConfig struct {
	URL         string `instill:"url"`
	InsecureSSL string `instill:"insecure-ssl"`
	Secret      string `instill:"secret"`
	ContentType string `instill:"content-type"`
}

type HookInfo struct {
	ID      int64      `instill:"id"`
	URL     string     `instill:"url"`
	PingURL string     `instill:"ping-url"`
	TestURL string     `instill:"test-url"`
	Config  HookConfig `instill:"config"`
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
