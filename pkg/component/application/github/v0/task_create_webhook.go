package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) createWebhook(ctx context.Context, job *base.Job) error {
	var input createWebHookInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}
	hookURL := input.HookURL
	hookSecret := input.HookSecret
	originalEvents := input.Events
	active := input.Active
	contentType := input.ContentType
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

	hook, _, err = client.Repositories.CreateHook(ctx, owner, repository, hook)
	if err != nil {
		return addErrMsgToClientError(err)
	}

	var output createWebHookOutput
	output.HookInfo = client.extractHook(hook)
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
