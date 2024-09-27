package huggingface

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
)

const (
	modelsPath = "/models/"
)

func newClient(setup *structpb.Struct, logger *zap.Logger) *httpclient.Client {
	c := httpclient.New("Hugging Face", getBaseURL(setup),
		httpclient.WithLogger(logger),
		httpclient.WithEndUserError(new(errBody)),
	)

	c.SetAuthToken(getAPIKey(setup))

	return c
}

type errBody struct {
	// Error can be either a string or a string array.
	Error json.RawMessage `json:"error,omitempty"`
}

func (e errBody) Message() string {
	var errStr string
	if err := json.Unmarshal(e.Error, &errStr); err == nil { // Error is string
		return errStr
	}

	var errSlice []string
	if err := json.Unmarshal(e.Error, &errSlice); err == nil { // Error is string slice
		return fmt.Sprintf("[%s]", strings.Join(errSlice, ", "))
	}

	return ""
}

func post(req *resty.Request, path string) (*resty.Response, error) {
	resp, err := req.Post(path)
	if err != nil {
		err = httpclient.WrapURLError(err)
	}

	return resp, err
}
