package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	scheduleSettingPath = "campaigns/{campaignID}/schedule?api_key={apiKey}"
	generalSettingPath  = "campaigns/{campaignID}/settings?api_key={apiKey}"
)

func (e *execution) setupCampaign(ctx context.Context, job *base.Job) error {

	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct setupCampaignInput

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

	scheduleReq := buildScheduleReq(inputStruct)

	res, err := client.R().SetBody(scheduleReq).Post(scheduleSettingPath)

	if err != nil {
		err = fmt.Errorf("setup schedule campaign: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("setup schedule campaign: %s", res.String())
		logger.Error("Failed to setup schedule campaign",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	generalReq := buildGeneralSettingReq(inputStruct)

	res, err = client.R().SetBody(generalReq).Post(generalSettingPath)

	if err != nil {
		err = fmt.Errorf("setup general campaign: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("setup general campaign: %s", res.String())
		logger.Error("Failed to setup general campaign",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	outputStruct := setupCampaignOutput{
		Result: "success",
	}

	err = job.Output.WriteData(ctx, outputStruct)

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildScheduleReq(input setupCampaignInput) setupScheduleCampaignReq {
	return setupScheduleCampaignReq{
		Timezone:          input.Timezone,
		DaysOfTheWeek:     input.DaysOfTheWeek,
		StartHour:         input.StartHour,
		EndHour:           input.EndHour,
		MinTimeBtwEmails:  input.MinTimeBtwEmails,
		MaxNewLeadsPerDay: input.MaxNewLeadsPerDay,
		ScheduleStartTime: input.ScheduleStartTime,
	}
}

type setupScheduleCampaignReq struct {
	Timezone          string   `json:"timezone,omitempty"`
	DaysOfTheWeek     []string `json:"days_of_the_week,omitempty"`
	StartHour         string   `json:"start_hour,omitempty"`
	EndHour           string   `json:"end_hour,omitempty"`
	MinTimeBtwEmails  int      `json:"min_time_btw_emails,omitempty"`
	MaxNewLeadsPerDay int      `json:"max_new_leads_per_day,omitempty"`
	ScheduleStartTime string   `json:"schedule_start_time,omitempty"`
}

func buildGeneralSettingReq(input setupCampaignInput) setupGeneralCampaignReq {
	return setupGeneralCampaignReq{
		TrackSettings:               input.TrackSettings,
		StopLeadSettings:            input.StopLeadSettings,
		SendAsPlainText:             input.SendAsPlainText,
		FollowUpPercentage:          input.FollowUpPercentage,
		AddUnsubscribeTag:           input.AddUnsubscribeTag,
		IgnoreSsMailboxSendingLimit: input.IgnoreSsMailboxSendingLimit,
	}
}

type setupGeneralCampaignReq struct {
	TrackSettings               []string `json:"track_settings,omitempty"`
	StopLeadSettings            string   `json:"stop_lead_settings,omitempty"`
	SendAsPlainText             bool     `json:"send_as_plain_text,omitempty"`
	FollowUpPercentage          int      `json:"follow_up_percentage,omitempty"`
	AddUnsubscribeTag           bool     `json:"add_unsubscribe_tag,omitempty"`
	IgnoreSsMailboxSendingLimit bool     `json:"ignore_ss_mailbox_sending_limit,omitempty"`
}
