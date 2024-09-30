package openai

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"
)

const (
	transcriptionsPath = "/v1/audio/transcriptions"
)

type AudioTranscriptionInput struct {
	Audio       string   `json:"audio"`
	Model       string   `json:"model"`
	Prompt      *string  `json:"prompt,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	Language    *string  `json:"language,omitempty"`
}

type AudioTranscriptionReq struct {
	File           []byte   `json:"file"`
	Model          string   `json:"model"`
	Prompt         *string  `json:"prompt,omitempty"`
	Language       *string  `json:"language,omitempty"`
	Temperature    *float64 `json:"temperature,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
}

type AudioTranscriptionResp struct {
	Text     string  `json:"text"`
	Duration float32 `json:"duration"`
}

func getBytes(req AudioTranscriptionReq) (*bytes.Reader, string, error) {
	data := &bytes.Buffer{}
	writer := multipart.NewWriter(data)
	err := util.WriteFile(writer, "file", req.File)
	if err != nil {
		return nil, "", err
	}
	util.WriteField(writer, "model", req.Model)
	util.WriteField(writer, "response_format", req.ResponseFormat)
	if req.Prompt != nil {
		util.WriteField(writer, "prompt", *req.Prompt)
	}
	if req.Language != nil {
		util.WriteField(writer, "language", *req.Language)
	}
	if req.Temperature != nil {
		util.WriteField(writer, "temperature", fmt.Sprintf("%f", *req.Temperature))
	}
	writer.Close()
	return bytes.NewReader(data.Bytes()), writer.FormDataContentType(), nil
}
