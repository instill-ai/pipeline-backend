package anthropic

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

type anthropicClient struct {
	httpClient *httpclient.Client
}

func newClient(apiKey string, baseURL string, logger *zap.Logger) *anthropicClient {
	c := httpclient.New("Anthropic", baseURL,
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)
	// Anthropic requires an API key to be set in the header "x-api-key" rather than normal "Authorization" header.
	c.Header.Set("X-Api-Key", apiKey)
	c.Header.Set("anthropic-version", "2023-06-01")

	return &anthropicClient{httpClient: c}
}

func (cl *anthropicClient) generateTextChat(request messagesReq) (messagesResp, error) {
	resp := messagesResp{}
	req := cl.httpClient.R().SetResult(&resp).SetBody(request)
	if _, err := req.Post(messagesPath); err != nil {
		return resp, err
	}
	return resp, nil
}

type errBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e errBody) Message() string {
	return e.Error.Message
}

func getBasePath(setup *structpb.Struct) string {
	v, ok := setup.GetFields()["base-path"]
	if !ok {
		return host
	}
	return v.GetStringValue()
}

func getAPIKey(setup *structpb.Struct) string {
	return setup.GetFields()[cfgAPIKey].GetStringValue()
}
