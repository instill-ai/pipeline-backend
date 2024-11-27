package smartlead

import (
	"context"
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"go.uber.org/zap"
)

var (
	createCampaignPath = "campaigns/create?api_key={apiKey}"
)

func (e *execution) createCampaign(ctx context.Context, job *base.Job) error {

	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct createCampaignInput

	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	requestIn := buildCreateCampaignRequest(inputStruct)

	var response createCampaignResp

	req := client.R().SetBody(requestIn).SetResult(&response)

	res, err := req.Post(createCampaignPath)

	if err != nil {
		err = fmt.Errorf("create campaign: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("create campaign: %s", res.String())
		logger.Error("Failed to create campaign",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	outputStruct := createCampaignOutput{
		ID:        fmt.Sprintf("%d", response.ID),
		CreatedAt: response.CreatedAt,
	}

	err = job.Output.WriteData(ctx, outputStruct)

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildCreateCampaignRequest(input createCampaignInput) createCampaignReq {
	return createCampaignReq{
		Name: input.Name,
	}
}

type createCampaignReq struct {
	Name string `json:"name"`
}

type createCampaignResp struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
}
