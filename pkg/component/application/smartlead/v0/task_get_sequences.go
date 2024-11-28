package smartlead

import (
	"context"
	"fmt"
	"log"

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

	var sequenceResp []sequenceResp

	req := client.R().SetResult(&sequenceResp)

	res, err := req.Get(getSequencesPath)

	log.Println("Sending request to get sequences", req.Body)
	log.Println("sequenceResp", sequenceResp)

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
			SeqID:             fmt.Sprintf("%d", s.ID),
			SeqNumber:         s.SeqNumber,
			Subject:           s.Subject,
			EmailBody:         s.EmailBody,
			SequenceDelayDays: s.SeqDelayDetails.DelayInDays,
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
	ID              int         `json:"id"`
	SeqNumber       int         `json:"seq_number"`
	Subject         string      `json:"subject"`
	EmailBody       string      `json:"email_body"`
	SeqDelayDetails delayInDays `json:"seq_delay_details"`
}

type delayInDays struct {
	// It is different from Smartlead API documentation.
	DelayInDays int `json:"delayInDays"`
}
