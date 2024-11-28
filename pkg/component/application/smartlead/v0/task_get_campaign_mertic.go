package smartlead

import (
	"context"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	getAnalyticsPath = "campaigns/{campaignID}/analytics?api_key={apiKey}"
)

func (e *execution) getCampaignMetric(ctx context.Context, job *base.Job) error {
	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct getCampaignMetricInput

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

	getCampaignMetricResp := getCampaignMetricResp{}

	res, err := client.R().SetResult(&getCampaignMetricResp).Get(getAnalyticsPath)

	if err != nil {
		err = fmt.Errorf("get campaign metric: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("get campaign metric: %s", res.String())
		logger.Error("Failed to get campaign metric",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	outputStruct := convertRespToStruct(getCampaignMetricResp)

	if err := job.Output.WriteData(ctx, outputStruct); err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func convertRespToStruct(resp getCampaignMetricResp) getCampaignMetricOutput {
	return getCampaignMetricOutput{
		SentCount:        toInt(resp.SentCount),
		UniqueSentCount:  toInt(resp.UniqueSentCount),
		OpenCount:        toInt(resp.OpenCount),
		UniqueOpenCount:  toInt(resp.UniqueOpenCount),
		ClickCount:       toInt(resp.ClickCount),
		UniqueClickCount: toInt(resp.UniqueClickCount),
		ReplyCount:       toInt(resp.ReplyCount),
		TotalCount:       toInt(resp.TotalCount),
		BounceCount:      toInt(resp.BounceCount),
	}
}

func toInt(value string) int {
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return intVal
}

type getCampaignMetricResp struct {
	SentCount        string `json:"sent_count"`
	UniqueSentCount  string `json:"unique_sent_count"`
	OpenCount        string `json:"open_count"`
	UniqueOpenCount  string `json:"unique_open_count"`
	ClickCount       string `json:"click_count"`
	UniqueClickCount string `json:"unique_click_count"`
	ReplyCount       string `json:"reply_count"`
	TotalCount       string `json:"total_count"`
	BounceCount      string `json:"bounce_count"`
}
