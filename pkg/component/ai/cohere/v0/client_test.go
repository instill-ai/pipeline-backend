package cohere

import (
	"context"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap"

	cohereSDK "github.com/cohere-ai/cohere-go/v2"
	qt "github.com/frankban/quicktest"

	"github.com/cohere-ai/cohere-go/v2/core"
)

func newMockClient() *cohereClient {
	return &cohereClient{sdkClient: &MockSDKClient{}, logger: zap.NewNop(), lock: sync.Mutex{}}
}

type MockSDKClient struct {
}

func (cl *MockSDKClient) Chat(ctx context.Context, request *cohereSDK.ChatRequest, opts ...core.RequestOption) (*cohereSDK.NonStreamedChatResponse, error) {
	uid := "944a80f0-c485-4fda-a5e8-2dd68890a5b7"
	return &cohereSDK.NonStreamedChatResponse{
		Text:         strings.ToUpper(request.Message),
		GenerationId: &uid,
	}, nil
}

func (cl *MockSDKClient) Embed(ctx context.Context, request *cohereSDK.EmbedRequest, opts ...core.RequestOption) (*cohereSDK.EmbedResponse, error) {
	emb := make([][]float64, 1)
	emb[0] = make([]float64, len(request.Texts[0]))
	return &cohereSDK.EmbedResponse{
		EmbeddingsFloats: &cohereSDK.EmbedFloatsResponse{Embeddings: emb},
	}, nil
}

func (cl *MockSDKClient) Rerank(ctx context.Context, request *cohereSDK.RerankRequest, opts ...core.RequestOption) (*cohereSDK.RerankResponse, error) {
	docCnt := len(request.Documents)
	res := make([]*cohereSDK.RerankResponseResultsItem, docCnt)
	for i, doc := range request.Documents {
		// reverse the provided documents
		res[(docCnt-1)-i] = &cohereSDK.RerankResponseResultsItem{
			Document: &cohereSDK.RerankResponseResultsItemDocument{Text: doc.String},
		}
	}
	return &cohereSDK.RerankResponse{
		Results: res,
	}, nil
}

func TestClient(t *testing.T) {
	c := qt.New(t)

	clt := newMockClient()

	commandTc := struct {
		request cohereSDK.ChatRequest
		want    string
	}{
		request: cohereSDK.ChatRequest{Message: "Hello World"},
		want:    "HELLO WORLD",
	}
	c.Run("ok - task command", func(c *qt.C) {
		resp, err := clt.generateTextChat(commandTc.request)
		c.Check(err, qt.IsNil)
		c.Check(resp.Text, qt.Equals, commandTc.want)

	})

	embedTc := struct {
		request cohereSDK.EmbedRequest
		want    [][]float64
	}{
		request: cohereSDK.EmbedRequest{Texts: []string{"abcde"}},
		want:    [][]float64{{0, 0, 0, 0, 0}},
	}
	c.Run("ok - task embed", func(c *qt.C) {
		resp, err := clt.generateEmbedding(embedTc.request)
		c.Check(err, qt.IsNil)
		c.Check(len(resp.EmbeddingsFloats.Embeddings[0]), qt.Equals, len(embedTc.want[0]))

	})

	rerankTc := struct {
		request cohereSDK.RerankRequest
		want    []string
	}{

		request: cohereSDK.RerankRequest{Documents: []*cohereSDK.RerankRequestDocumentsItem{{String: "a"}, {String: "b"}, {String: "c"}, {String: "d"}}},

		want: []string{"d", "c", "b", "a"},
	}
	c.Run("ok - task rerank", func(c *qt.C) {
		resp, err := clt.generateRerank(rerankTc.request)
		c.Check(err, qt.IsNil)
		c.Check(len(resp.Results), qt.Equals, len(rerankTc.want))
		for i, r := range resp.Results {
			c.Check(r.Document.Text, qt.Equals, rerankTc.want[i])
		}
	})
}
