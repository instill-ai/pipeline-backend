package github

import (
	"context"

	"github.com/google/go-github/v62/github"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type CreateWebHookInput struct {
	RepoInfo
	HookURL     string   `json:"hook-url"`
	HookSecret  string   `json:"hook-secret"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
	ContentType string   `json:"content-type"` // including `json`, `form`
}

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
type CreateWebHookResp struct {
	HookInfo
}

func (githubClient *Client) extractHook(originalHook *github.Hook) HookInfo {
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

func (githubClient *Client) createWebhookTask(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateWebHookInput
	err := base.ConvertFromStructpb(props, &inputStruct)
	if err != nil {
		return nil, err
	}

	owner, repository, err := parseTargetRepo(inputStruct)
	if err != nil {
		return nil, err
	}
	hookURL := inputStruct.HookURL
	hookSecret := inputStruct.HookSecret
	originalEvents := inputStruct.Events
	active := inputStruct.Active
	contentType := inputStruct.ContentType
	if contentType != "json" && contentType != "form" {
		contentType = "json"
	}

	hook := &github.Hook{
		Name: github.String("web"), // only webhooks are supported
		Config: &github.HookConfig{
			InsecureSSL: github.String("0"), // SSL verification is required
			URL:         &hookURL,
			Secret:      &hookSecret,
			ContentType: &contentType,
		},
		Events: originalEvents,
		Active: &active,
	}

	hook, _, err = githubClient.Repositories.CreateHook(ctx, owner, repository, hook)
	if err != nil {
		return nil, err
	}

	var resp CreateWebHookResp
	hookInfo := githubClient.extractHook(hook)
	resp.HookInfo = hookInfo
	out, err := base.ConvertToStructpb(resp)
	if err != nil {
		return nil, err
	}
	return out, nil
}
