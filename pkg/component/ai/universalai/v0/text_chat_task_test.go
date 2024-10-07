package universalai

import (
	"context"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

func TestExecuteTextChat(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name       string
		modelName  string
		fakeAPIKey string
		wantErrMsg string
	}{
		// This test case validate that the model is not supported.
		{
			name:       "Unsupported Model",
			modelName:  "testModel",
			fakeAPIKey: "testAPIKey",
			wantErrMsg: "unsupported vendor for model: testModel",
		},
		// This test case validate that the request is sent from UniversalAI to OpenAI.
		// The other cases should be supported in openai/v1/text_chat_task_test.go.
		{
			name:       "OpenAI Model",
			modelName:  "gpt-4",
			fakeAPIKey: "",
			wantErrMsg: "send request to openai error with error code: 401",
		},
	}

	component := Init(base.Component{})
	component.instillAPIKey = map[string]string{
		"openai": "testAPIKey",
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(t *qt.C) {
			setup, err := structpb.NewStruct(map[string]any{
				"model":   tc.modelName,
				"api-key": tc.fakeAPIKey,
			})

			c.Assert(err, qt.IsNil)

			x := base.ComponentExecution{
				Component: component,
				Setup:     setup,
				Task:      "TASK_CHAT",
			}

			e, err := component.CreateExecution(x)

			c.Assert(err, qt.IsNil)

			ctx := context.Background()
			job := &base.Job{}

			input, err := structpb.NewStruct(map[string]interface{}{
				"data": map[string]interface{}{
					"messages": []interface{}{
						map[string]interface{}{
							"role": "user",
							"name": "John",
							"content": []interface{}{
								map[string]interface{}{
									"type": "text",
									"text": "Hello, how can I help you?",
								},
							},
						},
						map[string]interface{}{
							"role": "assistant",
							"content": []interface{}{
								map[string]interface{}{
									"type": "text",
									"text": "I'm here to assist you.",
								},
							},
						},
					},
				},
				"parameter": map[string]interface{}{
					"max-tokens":  100,
					"temperature": 0.7,
					"top-p":       0.9,
					"stream":      false,
				},
			})

			c.Assert(err, qt.IsNil)

			execution := e.(*execution)

			_, err = execution.executeTextChat(input, job, ctx)

			c.Assert(err.Error(), qt.Contains, tc.wantErrMsg)

		})
	}

}
