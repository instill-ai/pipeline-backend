package http

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) (*httpclient.Client, error) {
	c := httpclient.New("HTTP API", "",
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
