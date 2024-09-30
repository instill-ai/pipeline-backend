package milvus

import (
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Milvus", getURL(setup),
		httpclient.WithLogger(logger),
	)

	c.SetHeader("Authorization", "Bearer "+getUsername(setup)+":"+getPassword(setup))
	c.SetHeader("Content-Type", "application/json")

	return c
}

func getURL(setup *structpb.Struct) string {
	return setup.GetFields()["url"].GetStringValue()
}

func getUsername(setup *structpb.Struct) string {
	return setup.GetFields()["username"].GetStringValue()
}

func getPassword(setup *structpb.Struct) string {
	return setup.GetFields()["password"].GetStringValue()
}
