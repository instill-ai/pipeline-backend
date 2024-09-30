package openai

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("OpenAI", getBasePath(setup),
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetAuthToken(getAPIKey(setup))

	org := getOrg(setup)
	if org != "" {
		c.SetHeader("OpenAI-Organization", org)
	}

	return c
}

type errBody struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e errBody) Message() string {
	return e.Error.Message
}

// getBasePath returns OpenAI's API URL. This configuration param allows us to
// override the API the connector will point to. It isn't meant to be exposed
// to users. Rather, it can serve to test the logic against a fake server.
// TODO instead of having the API value hardcoded in the codebase, it should be
// read from a setup file or environment variable.
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

func getOrg(setup *structpb.Struct) string {
	val, ok := setup.GetFields()[cfgOrganization]
	if !ok {
		return ""
	}
	return val.GetStringValue()
}
