package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	updateCampaignPath = "campaigns/{campaignID}/status?api_key={apiKey}"
)

func (e *execution) updateCampaignStatus(ctx context.Context, job *base.Job) error {
	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct updateCampaignStatusInput

	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	campaignID, err := getCampaignID(client, logger, inputStruct.CampaignName)

	if err != nil {
		err = fmt.Errorf("getting campaign id: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	pathParams := map[string]string{
		"campaignID": campaignID,
	}

	client.SetPathParams(pathParams)

	updateCampaignStatusReq := buildUpdateCampaignStatusReq(inputStruct.Status)

	res, err := client.R().SetBody(updateCampaignStatusReq).Post(updateCampaignPath)

	if err != nil {
		err = fmt.Errorf("update campaign status: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("update campaign status: %s", res.String())
		logger.Error("Failed to update campaign status",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	outputStruct := updateCampaignStatusOutput{
		Result: "success",
	}

	if err := job.Output.WriteData(ctx, &outputStruct); err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildUpdateCampaignStatusReq(status string) updateCampaignStatusReq {
	return updateCampaignStatusReq{
		Status: status,
	}
}

type updateCampaignStatusReq struct {
	Status string `json:"status"`
}
