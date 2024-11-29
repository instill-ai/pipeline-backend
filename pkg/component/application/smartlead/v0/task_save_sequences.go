package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	saveSequencesPath = "campaigns/{campaignID}/sequences?api_key={apiKey}"
)

func (e *execution) saveSequences(ctx context.Context, job *base.Job) error {
	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct saveSequencesInput

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

	requestIn := buildSaveSequenceReq(inputStruct)

	var respStruct saveSequencesResp
	req := client.R().SetBody(requestIn).SetResult(&respStruct)

	res, err := req.Post(saveSequencesPath)

	if err != nil {
		err = fmt.Errorf("save sequences: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("save sequences: %s", res.String())
		logger.Error("Failed to save sequences",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	outputStruct := saveSequencesOutput{
		Result: respStruct.Data,
	}

	err = job.Output.WriteData(ctx, outputStruct)

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildSaveSequenceReq(input saveSequencesInput) saveSequencesReq {
	var sequences []sequenceReq
	for i, seq := range input.Sequences {
		var seqDelayDetails sequenceDelayDetails
		if seq.SequenceDelayDays != nil {
			seqDelayDetails = sequenceDelayDetails{
				DelayInDays: *seq.SequenceDelayDays,
			}
		} else {
			seqDelayDetails = sequenceDelayDetails{
				DelayInDays: 1,
			}
		}
		var seqNumber int
		if seq.SeqNumber != nil {
			seqNumber = *seq.SeqNumber
		} else {
			seqNumber = i + 1
		}
		sequences = append(sequences, sequenceReq{
			SeqNumber:       &seqNumber,
			SeqDelayDetails: seqDelayDetails,
			Subject:         seq.Subject,
			EmailBody:       seq.EmailBody,
		})
	}
	return saveSequencesReq{
		Sequences: sequences,
	}
}

type saveSequencesReq struct {
	Sequences []sequenceReq `json:"sequences"`
}

type sequenceReq struct {
	SeqNumber       *int                 `json:"seq_number,omitempty"`
	SeqDelayDetails sequenceDelayDetails `json:"seq_delay_details"`
	Subject         string               `json:"subject"`
	EmailBody       string               `json:"email_body"`
}

type sequenceDelayDetails struct {
	// It is delay_in_days in save sequence request.
	DelayInDays int `json:"delay_in_days"`
}

type saveSequencesResp struct {
	Ok   bool   `json:"ok"`
	Data string `json:"data"`
}
