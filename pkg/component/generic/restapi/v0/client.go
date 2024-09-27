package restapi

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

type TaskInput struct {
	EndpointURL string      `json:"endpoint-url"`
	Body        interface{} `json:"body,omitempty"`
}

type TaskOutput struct {
	StatusCode int                 `json:"status-code"`
	Body       interface{}         `json:"body"`
	Header     map[string][]string `json:"header"`
}

func newClient(setup *structpb.Struct, logger *zap.Logger) (*httpclient.Client, error) {
	c := httpclient.New("REST API", "",
		httpclient.WithLogger(logger),
	)

	auth, err := getAuthentication(setup)
	if err != nil {
		return nil, err
	}

	if err := auth.setAuthInClient(c); err != nil {
		return nil, err
	}

	return c, nil
}
