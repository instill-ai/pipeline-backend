package stabilityai

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Stability AI", getBasePath(setup),
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetAuthToken(getAPIKey(setup))

	return c
}

type errBody struct {
	Msg string `json:"message"`
}

func (e errBody) Message() string {
	return e.Msg
}

// getBasePath returns Stability AI's API URL. This configuration param allows
// us to override the API the component will point to. It isn't meant to be
// exposed to users. Rather, it can serve to test the logic against a fake
// server.
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
