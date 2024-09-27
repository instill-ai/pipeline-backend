package asana

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type JobTaskOutput struct {
	Job
}

type JobTaskResp struct {
	Data struct {
		GID        string  `json:"gid"`
		NewTask    Task    `json:"new_task"`
		NewProject Project `json:"new_project"`
	} `json:"data"`
}

func JobResp2Output(resp *JobTaskResp) JobTaskOutput {
	out := JobTaskOutput{
		Job: Job{
			GID: resp.Data.GID,
			NewTask: Task{
				GID:  resp.Data.NewTask.GID,
				Name: resp.Data.NewTask.Name,
			},
			NewProject: Project{
				GID:  resp.Data.NewProject.GID,
				Name: resp.Data.NewProject.Name,
			},
		},
	}
	return out
}

type GetJobInput struct {
	Action string `json:"action"`
	ID     string `json:"job-gid"`
}

func (c *Client) GetJob(ctx context.Context, props *structpb.Struct) (*structpb.Struct, error) {
	var input GetJobInput
	if err := base.ConvertFromStructpb(props, &input); err != nil {
		return nil, err
	}

	apiEndpoint := fmt.Sprintf("/jobs/%s", input.ID)
	req := c.Client.R().SetResult(&JobTaskResp{})

	wantOptFields := parseWantOptionFields(Job{})
	if err := addQueryOptions(req, map[string]interface{}{"opt_fields": wantOptFields}); err != nil {
		return nil, err
	}
	resp, err := req.Get(apiEndpoint)
	if err != nil {
		return nil, err
	}

	job := resp.Result().(*JobTaskResp)
	out := JobResp2Output(job)

	return base.ConvertToStructpb(out)
}
