package asana

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

type Client struct {
	*httpclient.Client
	APIBaseURL string `json:"api-base-url"`
	logger     *zap.Logger
}

type AuthConfig struct {
	Token   string `json:"token"`
	BaseURL string `json:"base-url"`
}

func newClient(_ context.Context, setup *structpb.Struct, logger *zap.Logger) (*Client, error) {
	var authConfig AuthConfig
	if err := base.ConvertFromStructpb(setup, &authConfig); err != nil {
		return nil, err
	}

	token := strings.TrimSpace(authConfig.Token)
	baseURL := authConfig.BaseURL
	if token == "" {
		return nil, errmsg.AddMessage(
			fmt.Errorf("token not provided"),
			"token not provided",
		)
	}
	if baseURL == "" {
		// Base URL is only left for mock server testing
		// We can use the default base URL for the production
		baseURL = apiBaseURL
	}
	asanaClient := httpclient.New(
		"Asana-Client",
		baseURL,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)
	asanaClient.
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetAuthToken(token)
	client := &Client{
		Client:     asanaClient,
		APIBaseURL: baseURL,
		logger:     logger,
	}
	return client, nil
}

type errBody struct {
	Body struct {
		Msg []string `json:"errorMessages"`
	} `json:"body"`
}

func (e errBody) Message() string {
	return strings.Join(e.Body.Msg, " ")
}
