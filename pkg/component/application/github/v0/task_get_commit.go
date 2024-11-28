package github

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) getCommit(ctx context.Context, job *base.Job) error {
	var input getCommitInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}
	owner, repository, err := parseTargetRepo(input)
	if err != nil {
		return err
	}
	sha := input.SHA
	commit, _, err := client.Repositories.GetCommit(ctx, owner, repository, sha, nil)
	if err != nil {
		return addErrMsgToClientError(err)
	}
	var output getCommitOutput
	output.Commit, err = client.extractCommitInformation(ctx, owner, repository, commit, true)
	if err != nil {
		return err
	}
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
