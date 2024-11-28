package smartlead

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

var (
	addSenderEmailPath = "campaigns/{campaignID}/email-accounts?api_key={apiKey}"
)

func (e *execution) addSenderEmail(ctx context.Context, job *base.Job) error {

	logger := e.GetLogger()
	client := newClient(e.GetSetup(), logger, nil)

	var inputStruct addSenderEmailInput

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

	senderEmailID, err := getSenderEmailID(client, logger, inputStruct.SenderEmail)

	if err != nil {
		err = fmt.Errorf("getting sender email id: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	addSenderEmailReq := buildSenderEmailReq(senderEmailID)

	res, err := client.R().SetBody(addSenderEmailReq).Post(addSenderEmailPath)

	if err != nil {
		err = fmt.Errorf("add sender email: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("add sender email: %s", res.String())
		logger.Error("Failed to add sender email",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		job.Error.Error(ctx, err)
		return err
	}

	err = job.Output.WriteData(ctx, addSenderEmailOutput{
		Result: "success",
	})

	if err != nil {
		err = fmt.Errorf("writing output data: %w", err)
		job.Error.Error(ctx, err)
		return err
	}

	return nil
}

func buildSenderEmailReq(senderEmailID string) addSenderEmailReq {
	return addSenderEmailReq{
		EmailAccountIDs: []string{senderEmailID},
	}
}

type addSenderEmailReq struct {
	EmailAccountIDs []string `json:"email_account_ids"`
}
