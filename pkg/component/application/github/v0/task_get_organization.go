package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v62/github"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func (client *Client) getOrganization(ctx context.Context, job *base.Job) error {
	var input getOrganizationInput
	if err := job.Input.ReadData(ctx, &input); err != nil {
		return fmt.Errorf("reading input data: %w", err)
	}

	var org *github.Organization
	var err error

	if input.OrgID != 0 {
		org, _, err = client.Organizations.GetByID(ctx, input.OrgID)
	} else {
		org, _, err = client.Organizations.Get(ctx, input.OrgName)
	}
	if err != nil {
		return addErrMsgToClientError(err)
	}

	var output getOrganizationOutput
	output.Organization = client.extractOrganization(org)
	if err := job.Output.WriteData(ctx, output); err != nil {
		return err
	}
	return nil
}
