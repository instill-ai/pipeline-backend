package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	listLeadsStatusPath = "campaigns/{campaignID}/leads?api_key={apiKey}&limit={limit}"
)

func (e *execution) listLeadsStatus(ctx context.Context, job *base.Job) error {

	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct listLeadsStatusInput

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
		"limit":      fmt.Sprintf("%d", inputStruct.Limit),
	}

	client.SetPathParams(pathParams)

	leadsStatusResp := leadsStatusResp{}

	res, err := client.R().SetResult(&leadsStatusResp).Get(listLeadsStatusPath)

	if err != nil {
		err = fmt.Errorf("list leads status: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("list leads status: %s", res.String())
		logger.Error("Failed to list leads status",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	output := convertLeadsStatusRespToStruct(leadsStatusResp)

	if err := job.Output.WriteData(ctx, output); err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func convertLeadsStatusRespToStruct(resp leadsStatusResp) listLeadsStatusOutput {
	output := listLeadsStatusOutput{
		Leads: []leadStatus{},
	}

	for _, lead := range resp.Data {
		output.Leads = append(output.Leads, leadStatus{
			Email:  lead.Lead.Email,
			Status: lead.Status,
		})
	}

	return output
}

type leadsStatusResp struct {
	TotalLeads string `json:"total_leads"`
	Data       []struct {
		Status string `json:"status"`
		Lead   struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"lead"`
	} `json:"data"`
}
