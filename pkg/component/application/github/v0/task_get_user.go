package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) getUser(ctx context.Context, job *base.Job) error {
	var input getUserInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	var user *github.User
	var err error

	if input.UserID != 0 {
		user, _, err = client.Users.GetByID(ctx, input.UserID)
	} else {
		user, _, err = client.Users.Get(ctx, input.Username)
	}
	if err != nil {
		return addErrMsgToClientError(err)
	}

	var output getUserOutput
	output.User = client.extractUser(user)
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
