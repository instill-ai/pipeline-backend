package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	addLeadsPath = "campaigns/{campaignID}/leads?api_key={apiKey}"
)

func (e *execution) addLeads(ctx context.Context, job *base.Job) error {

	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct addLeadsInput

	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	campaignID, err := getCampaignID(client, logger, inputStruct.CampaignName)

	if err != nil {
		err = fmt.Errorf("getting campaign ID: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	pathParams := map[string]string{
		"campaignID": campaignID,
	}

	client.SetPathParams(pathParams)

	requestIn := buildAddLeadsRequest(inputStruct)

	var response addLeadsResp

	req := client.R().SetBody(requestIn).SetResult(&response)

	res, err := req.Post(addLeadsPath)

	if err != nil {
		err = fmt.Errorf("add leads: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("add leads: %s", res.String())
		logger.Error("Failed to add leads",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	outputStruct := addLeadsOutput{
		UploadCount:            response.UploadCount,
		TotalLeads:             response.TotalLeads,
		AlreadyAddedToCampaign: response.AlreadyAddedToCampaign,
		InvalidEmailCount:      response.InvalidEmailCount,
		Error:                  response.Error,
	}

	err = job.Output.WriteData(ctx, outputStruct)

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildAddLeadsRequest(input addLeadsInput) addLeadsReq {
	var leads []leadReq
	for _, l := range input.Leads {
		customFields := make(map[string]string)

		for _, cf := range l.CustomFields {
			customFields[cf.Key] = cf.Value
		}

		leads = append(leads, leadReq{
			FirstName:    l.FirstName,
			LastName:     l.LastName,
			Email:        l.Email,
			CompanyName:  l.Company,
			Location:     l.Location,
			CustomFields: customFields,
		})
	}

	return addLeadsReq{
		Leads: leads,
		Settings: settingsReq{
			IgnoreGlobalBlockList:               input.Settings.IgnoreGlobalBlockList,
			IgnoreUnsubscribeList:               input.Settings.IgnoreUnsubscribeList,
			IgnoreCommunityBounceList:           input.Settings.IgnoreCommunityBounceList,
			IgnoreDuplicateLeadsInOtherCampaign: input.Settings.IgnoreDuplicateLeadsInOtherCampaign,
		},
	}
}

type addLeadsReq struct {
	Leads    []leadReq   `json:"lead_list"`
	Settings settingsReq `json:"settings"`
}

type leadReq struct {
	FirstName    *string           `json:"first_name,omitempty"`
	LastName     *string           `json:"last_name,omitempty"`
	Email        string            `json:"email"`
	CompanyName  *string           `json:"company_name,omitempty"`
	Location     *string           `json:"location,omitempty"`
	CustomFields map[string]string `json:"custom_fields"`
}

type settingsReq struct {
	IgnoreGlobalBlockList               bool `json:"ignore_global_block_list"`
	IgnoreUnsubscribeList               bool `json:"ignore_unsubscribe_list"`
	IgnoreCommunityBounceList           bool `json:"ignore_community_bounce_list"`
	IgnoreDuplicateLeadsInOtherCampaign bool `json:"ignore_duplicate_leads_in_other_campaign"`
}

type addLeadsResp struct {
	UploadCount            int    `json:"upload_count"`
	TotalLeads             int    `json:"total_leads"`
	AlreadyAddedToCampaign int    `json:"already_added_to_campaign"`
	InvalidEmailCount      int    `json:"invalid_email_count"`
	Error                  string `json:"error"`
}
