package huggingface

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util/httpclient"
	"github.com/instill-ai/x/errmsg"
)

const (
	apiKey = "123"
	model  = "openai/whisper-tiny"

	testInput = "testing generation"

	fillMaskResp            = `[{"score": 0.234, "token": 3, "sequence": "one", "token-str": "three"}]`
	classificationResp      = `[{"score": 0.123, "label": "backpack hip-hop"}, {"score": 0.894, "label": "lo-fi jazz"}]`
	tokenClassificationResp = `[{"entity-group":"foo", "score": 0.234, "start": 0, "end": 5, "word": "bar"}]`
	objDetectionResp        = `
[
  {
	"score": 0.123,
	"label": "backpack hip-hop",
	"box": {
	  "xmin": 0,
	  "xmax": 1,
	  "ymin": 0,
	  "ymax": 1
	}
  }
]`

	errorResp  = ` { "error": "Invalid request" }`
	errorsResp = ` { "error": ["Temporarily unavailable", "Too many requests"] }`
)

var (
	bRaw     = []byte("aaa")
	bEncoded = base64.StdEncoding.EncodeToString(bRaw)

	inputsBody = []byte(`{"inputs": "testing generation"}`)
)

type taskParams struct {
	task        string
	input       any
	contentType string // content type received in Hugging Face
	wantBody    []byte // expected request body in Hugging Face
	okResp      string // successful response from Hugging Face
	wantResp    string // successful response from component
}

func wrapArrayInObject(array, key string) string {
	return fmt.Sprintf(`{"%s": %s}`, key, array)
}

var coveredTasks = []taskParams{
	{
		task:        textGenerationTask,
		input:       TextGenerationRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      `[{"generated_text": "text response"}]`,
		wantResp:    `{"generated-text": "text response"}`,
	},
	{
		task:        textToImageTask,
		input:       TextToImageRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      string(bRaw),
		wantResp:    fmt.Sprintf(`{"image": "data:image/jpeg;base64,%s"}`, bEncoded),
	},
	{
		task:        fillMaskTask,
		input:       FillMaskRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      fillMaskResp,
		wantResp:    wrapArrayInObject(fillMaskResp, "results"),
	},
	{
		task:        summarizationTask,
		input:       SummarizationRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      `[{"summary_text": "summary"}]`,
		wantResp:    `{"summary-text": "summary"}`,
	},
	{
		task:        textClassificationTask,
		input:       TextClassificationRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      "[" + classificationResp + "]",
		wantResp:    wrapArrayInObject(classificationResp, "results"),
	},
	{
		task:        tokenClassificationTask,
		input:       TokenClassificationRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      tokenClassificationResp,
		wantResp:    wrapArrayInObject(tokenClassificationResp, "results"),
	},
	{
		task:        translationTask,
		input:       TranslationRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      `[{"translation_text": "translated"}]`,
		wantResp:    `{"translation-text": "translated"}`,
	},
	{
		task:        zeroShotClassificationTask,
		input:       ZeroShotRequest{Inputs: testInput},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    inputsBody,
		okResp:      `{"sequence": "seq"}`,
		wantResp:    `{"sequence": "seq"}`,
	},
	{
		task:        questionAnsweringTask,
		input:       QuestionAnsweringRequest{Inputs: QuestionAnsweringInputs{Question: "isn't it?"}},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    []byte(`{"inputs": {"question": "is it?"}}`),
		okResp:      `{"answer": "it is"}`,
		wantResp:    `{"answer": "it is"}`,
	},
	{
		task:        tableQuestionAnsweringTask,
		input:       TableQuestionAnsweringRequest{Inputs: TableQuestionAnsweringInputs{Query: "yes?"}},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    []byte(`{"inputs": {"query": "yes?"}}`),
		okResp:      `{"answer": "yes"}`,
		wantResp:    `{"answer": "yes"}`,
	},
	{
		task:        sentenceSimilarityTask,
		input:       SentenceSimilarityRequest{Inputs: SentenceSimilarityInputs{}},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    []byte(`{"inputs": {}}`),
		okResp:      `[0.23]`,
		wantResp:    wrapArrayInObject(`[0.23]`, "scores"),
	},
	{
		task:        conversationalTask,
		input:       ConversationalRequest{Inputs: ConversationalInputs{}},
		contentType: httpclient.MIMETypeJSON,
		wantBody:    []byte(`{"inputs": {}}`),
		okResp:      `{"generated_text": "gen"}`,
		wantResp:    `{"generated-text": "gen"}`,
	},
	{
		task:        imageClassificationTask,
		input:       ImageRequest{Image: bEncoded},
		contentType: "text/plain.*",
		wantBody:    bRaw,
		okResp:      classificationResp,
		wantResp:    wrapArrayInObject(classificationResp, "classes"),
	},
	{
		task:        imageSegmentationTask,
		input:       ImageRequest{Image: bEncoded},
		contentType: "text/plain.*",
		wantBody:    bRaw,
		okResp:      `[{"score": 0.123, "label": "backpack hip-hop", "mask": "YBcsSdfg"}]`,
		wantResp:    `{"segments": [{"score": 0.123, "label": "backpack hip-hop", "mask": "data:image/png;base64,YBcsSdfg"}]}`,
	},
	{
		task:        objectDetectionTask,
		input:       ImageRequest{Image: bEncoded},
		contentType: "text/plain.*",
		wantBody:    bRaw,
		okResp:      objDetectionResp,
		wantResp:    wrapArrayInObject(objDetectionResp, "objects"),
	},
	{
		task:        imageToTextTask,
		input:       ImageRequest{Image: bEncoded},
		contentType: "text/plain.*",
		wantBody:    bRaw,
		okResp:      `[{"generated_text": "Me robaron mi runa mula"}]`,
		wantResp:    `{"text": "Me robaron mi runa mula"}`,
	},
	{
		task:        speechRecognitionTask,
		input:       AudioRequest{Audio: bEncoded},
		contentType: "text/plain.*",
		wantBody:    bRaw,
		okResp:      `{"text": "Me robaron mi runa mula"}`,
		wantResp:    `{"text": "Me robaron mi runa mula"}`,
	},
	{
		task:        audioClassificationTask,
		input:       AudioRequest{Audio: bEncoded},
		contentType: "text/plain.*",
		wantBody:    bRaw,
		okResp:      classificationResp,
		wantResp:    wrapArrayInObject(classificationResp, "classes"),
	},
}

func TestComponent_ExecuteSpeechRecognition(t *testing.T) {
	c := qt.New(t)

	for _, params := range coveredTasks {
		testTask(c, params)
	}
}

func testTask(c *qt.C, p taskParams) {
	bc := base.Component{}
	cmp := Init(bc)
	ctx := context.Background()

	c.Run("nok - HTTP client error - "+p.task, func(c *qt.C) {
		c.Parallel()

		setup, err := structpb.NewStruct(map[string]any{
			"base-url": "http://no-such.host",
		})
		c.Assert(err, qt.IsNil)

		exec, err := cmp.CreateExecution(base.ComponentExecution{
			Component: cmp,
			Setup:     setup,
			Task:      p.task,
		})
		c.Assert(err, qt.IsNil)

		pbIn, err := base.ConvertToStructpb(p.input)
		c.Assert(err, qt.IsNil)
		pbIn.Fields["model"] = structpb.NewStringValue(model)

		ir, ow, eh, job := mock.GenerateMockJob(c)
		ir.ReadMock.Return(pbIn, nil)
		ow.WriteMock.Optional().Return(nil)
		eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
			c.Check(err, qt.ErrorMatches, ".*no such host")
			c.Check(errmsg.Message(err), qt.Matches, "Failed to call .*check that the component configuration is correct.")
		})

		err = exec.Execute(ctx, []*base.Job{job})
		c.Check(err, qt.IsNil)

	})

	testcases := []struct {
		name           string
		customEndpoint bool
		httpStatus     int
		httpBody       string
		wantErr        string
	}{
		{
			name:       "ok",
			httpStatus: http.StatusOK,
			httpBody:   p.okResp,
		},
		{
			name:       "nok - API error",
			httpStatus: http.StatusBadRequest,
			httpBody:   errorResp,
			wantErr:    "Hugging Face responded with a 400 status code. Invalid request",
		},
		{
			name:           "nok - API errors",
			customEndpoint: true,
			httpStatus:     http.StatusTooManyRequests,
			httpBody:       errorsResp,
			wantErr:        "Hugging Face responded with a 429 status code. [Temporarily unavailable, Too many requests]",
		},
	}

	for _, tc := range testcases {
		tc := tc
		c.Run(tc.name+" _ "+p.task, func(c *qt.C) {
			c.Parallel()

			wantPath := modelsPath + model
			if tc.customEndpoint {
				wantPath = "/"
			}

			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.Check(r.Method, qt.Equals, http.MethodPost)
				c.Check(r.URL.Path, qt.Matches, wantPath)

				c.Check(r.Header.Get("Authorization"), qt.Equals, "Bearer "+apiKey)

				ct := r.Header.Get("Content-Type")
				c.Check(ct, qt.Matches, p.contentType)

				c.Assert(r.Body, qt.IsNotNil)
				defer r.Body.Close()

				body, err := io.ReadAll(r.Body)
				c.Assert(err, qt.IsNil)
				if ct == httpclient.MIMETypeJSON {
					// If we have a case where we don't pass the input request,
					// we can check if p.wantBody is not empty and then do
					// c.Check(body, qt.JSONEquals, json.RawMessage(p.wantBody)

					c.Check(body, qt.JSONEquals, p.input)
				} else {
					c.Check(body, qt.ContentEquals, p.wantBody)
				}

				w.Header().Set("Content-Type", httpclient.MIMETypeJSON)
				w.WriteHeader(tc.httpStatus)
				fmt.Fprint(w, tc.httpBody)
			})

			srv := httptest.NewServer(h)
			c.Cleanup(srv.Close)

			setup, _ := structpb.NewStruct(map[string]any{
				"api-key":            apiKey,
				"base-url":           srv.URL,
				"is-custom-endpoint": tc.customEndpoint,
			})

			exec, err := cmp.CreateExecution(base.ComponentExecution{
				Component: cmp,
				Setup:     setup,
				Task:      p.task,
			})
			c.Assert(err, qt.IsNil)

			pbIn, err := base.ConvertToStructpb(p.input)
			c.Assert(err, qt.IsNil)
			pbIn.Fields["model"] = structpb.NewStringValue(model)

			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadMock.Return(pbIn, nil)
			ow.WriteMock.Optional().Set(func(ctx context.Context, output *structpb.Struct) (err error) {
				c.Check(p.wantResp, qt.JSONEquals, output.AsMap())
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				if tc.wantErr != "" {
					c.Check(err, qt.IsNotNil)
					c.Check(errmsg.Message(err), qt.Equals, tc.wantErr)
				}
			})

			err = exec.Execute(ctx, []*base.Job{job})
			c.Check(err, qt.IsNil)

		})
	}
}
