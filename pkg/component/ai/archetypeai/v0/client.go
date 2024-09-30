package archetypeai

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	host           = "https://api.archetypeai.dev"
	describePath   = "/v0.3/describe"
	summarizePath  = "/v0.3/summarize"
	uploadFilePath = "/v0.3/files"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Archetype AI", getBasePath(setup),
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetAuthToken(getAPIKey(setup))

	return c
}

type errBody struct {
	Error string `json:"error"`
}

func (e errBody) Message() string {
	return e.Error
}
