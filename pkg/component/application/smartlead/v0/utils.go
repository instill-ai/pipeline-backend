package smartlead

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

var (
	getCampaignPath    = "campaigns?api_key={apiKey}"
	getSenderEmailPath = "email-accounts/?api_key={apiKey}"
)

func getCampaignID(client *httpclient.Client, logger *zap.Logger, campaignName string) (string, error) {

	campaignResp := []campaignResp{}
	req := client.R().SetResult(&campaignResp)

	res, err := req.Get(getCampaignPath)

	if err != nil {
		err = fmt.Errorf("get campaign: %w", err)
		return "", err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("get campaign: %s", res.String())
		logger.Error("Failed to get campaign",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		return "", err
	}

	var campaignID string

	for _, c := range campaignResp {
		if c.Name == campaignName {
			campaignID = fmt.Sprintf("%d", c.ID)
			break
		}
	}

	if campaignID == "" {
		err = fmt.Errorf("campaign not found: %s", campaignName)
		return "", err
	}

	return campaignID, nil
}

type campaignResp struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getSenderEmailID(client *httpclient.Client, logger *zap.Logger, senderEmail string) (string, error) {

	senderEmailResp := []senderEmailResp{}
	req := client.R().SetResult(&senderEmailResp)

	res, err := req.Get(getSenderEmailPath)

	if err != nil {
		err = fmt.Errorf("get sender email: %w", err)
		return "", err
	}

	if res.StatusCode() != 200 {
		err = fmt.Errorf("get sender email: %s", res.String())
		logger.Error("Failed to get sender email",
			zap.String("response", res.String()),
			zap.Int("status", res.StatusCode()),
		)
		return "", err
	}

	var senderEmailID string

	for _, c := range senderEmailResp {
		if c.FromEmail == senderEmail {
			senderEmailID = fmt.Sprintf("%d", c.ID)
			break
		}
	}

	if senderEmailID == "" {
		err = fmt.Errorf("sender email not found: %s", senderEmail)
		return "", err
	}

	return senderEmailID, nil
}

type senderEmailResp struct {
	ID        int    `json:"id"`
	FromEmail string `json:"from_email"`
}
