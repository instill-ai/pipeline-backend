package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	getSequencesPath = "campaigns/{campaignID}/sequences?api_key={apiKey}"
)

func (e *execution) getSequences(ctx context.Context, job *base.Job) error {
	var inputStruct getSequencesInput

	if err := job.Input.ReadData(ctx, &inputStruct); err != nil {
		err = fmt.Errorf("reading input data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}
	logger := e.GetLogger()

	client := newClient(e.GetSetup(), logger, nil)

	campaignResp := []campaignResp{}
	req := client.R().SetResult(&campaignResp)

	res, err := req.Get(getCampaignPath)

	if err != nil {
		err = fmt.Errorf("get campaign: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("get campaign: %s", res.String())
		logger.Error("Failed to get campaign",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	var campaignID string

	for _, c := range campaignResp {
		if c.Name == inputStruct.CampaignName {
			campaignID = fmt.Sprintf("%d", c.ID)
			break
		}
	}

	if campaignID == "" {
		err = fmt.Errorf("campaign not found: %s", inputStruct.CampaignName)
		job.Error.Error(ctx, err)
		return err
	}

	pathParams := map[string]string{
		"campaignID": campaignID,
	}

	client.SetPathParams(pathParams)

	var sequenceResp []sequenceResp

	req = client.R().SetResult(&sequenceResp)

	res, err = req.Get(getSequencesPath)

	if err != nil {
		err = fmt.Errorf("get sequences: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("get sequences: %s", res.String())
		logger.Error("Failed to get sequences",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	var sequences []sequence
	for _, s := range sequenceResp {
		sequences = append(sequences, sequence{
			SeqID:     fmt.Sprintf("%d", s.ID),
			SeqNumber: s.SeqNumber,
			Subject:   s.Subject,
			EmailBody: s.EmailBody,
		})
	}

	outputStruct := getSequencesOutput{
		Sequences: sequences,
	}

	err = job.Output.WriteData(ctx, outputStruct)

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

type sequenceResp struct {
	ID        int    `json:"id"`
	SeqNumber int    `json:"seq-number"`
	Subject   string `json:"subject"`
	EmailBody string `json:"email_body"`
}
