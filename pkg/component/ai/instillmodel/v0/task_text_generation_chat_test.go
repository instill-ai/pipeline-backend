package instillmodel

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func TestExecuteTextGenerationChat(t *testing.T) {
	c := qt.New(t)

	// Create mock execution
	exec := &execution{
		ComponentExecution: base.ComponentExecution{
			SystemVariables: map[string]any{
				"__PIPELINE_USER_UID":      "test-user",
				"__PIPELINE_REQUESTER_UID": "test-requester",
			},
		},
	}

	// Test case 1: Successful text generation chat
	t.Run("successful text generation chat", func(t *testing.T) {
		// Create mock response
		mockResponse := &modelpb.TriggerModelResponse{
			TaskOutputs: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"data": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"choices": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"message": structpb.NewStructValue(&structpb.Struct{
													Fields: map[string]*structpb.Value{
														"content": structpb.NewStringValue("Hello! How can I help you today?"),
													},
												}),
											},
										}),
									},
								}),
							},
						}),
					},
				},
			},
		}

		mockClient := &mockModelPublicServiceClient{
			triggerResponse: mockResponse,
		}

		// Create test input
		textGenChatInput := TextGenerationChatInput{
			ModelName:     "test-ns/test-model/v1",
			Prompt:        "Hello, how are you?",
			PromptImages:  []string{"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="},
			SystemMessage: stringPtr("You are a helpful assistant."),
			MaxNewTokens:  intPtr(100),
			Temperature:   float32Ptr(0.7),
			Seed:          intPtr(42),
		}

		inputStruct, err := base.ConvertToStructpb(textGenChatInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeTextGenerationChat(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure
		var output TextGenerationChatOutput
		err = base.ConvertFromStructpb(result[0], &output)
		c.Assert(err, qt.IsNil)
		c.Assert(output.Text, qt.Equals, "Hello! How can I help you today?")
	})

	// Test case 2: Chat with history
	t.Run("chat with history", func(t *testing.T) {
		// Create mock response
		mockResponse := &modelpb.TriggerModelResponse{
			TaskOutputs: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"data": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"choices": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"message": structpb.NewStructValue(&structpb.Struct{
													Fields: map[string]*structpb.Value{
														"content": structpb.NewStringValue("Based on our previous conversation, I can help you with that."),
													},
												}),
											},
										}),
									},
								}),
							},
						}),
					},
				},
			},
		}

		mockClient := &mockModelPublicServiceClient{
			triggerResponse: mockResponse,
		}

		// Create test input with chat history
		textGenChatInput := TextGenerationChatInput{
			ModelName: "test-ns/test-model/v1",
			Prompt:    "Can you elaborate on your previous answer?",
			ChatHistory: []ChatMessage{
				{
					Role: "user",
					Content: []MultiModalContent{
						{
							Type: "text",
							Text: "What is machine learning?",
						},
					},
				},
				{
					Role: "assistant",
					Content: []MultiModalContent{
						{
							Type: "text",
							Text: "Machine learning is a subset of artificial intelligence.",
						},
					},
				},
			},
		}

		inputStruct, err := base.ConvertToStructpb(textGenChatInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeTextGenerationChat(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure
		var output TextGenerationChatOutput
		err = base.ConvertFromStructpb(result[0], &output)
		c.Assert(err, qt.IsNil)
		c.Assert(output.Text, qt.Equals, "Based on our previous conversation, I can help you with that.")
	})

	// Test case 3: Empty inputs
	t.Run("empty inputs", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}
		inputs := []*structpb.Struct{}

		result, err := exec.executeTextGenerationChat(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid input.*")
		c.Assert(result, qt.IsNil)
	})

	// Test case 4: Nil gRPC client
	t.Run("nil grpc client", func(t *testing.T) {
		inputStruct, _ := structpb.NewStruct(map[string]any{
			"model-name": "test-ns/test-model/v1",
			"prompt":     "test prompt",
		})
		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeTextGenerationChat(nil, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "uninitialized client")
		c.Assert(result, qt.IsNil)
	})

	// Test case 5: Input conversion error
	t.Run("input conversion error", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}

		// Create invalid input structure
		inputStruct, _ := structpb.NewStruct(map[string]any{
			"invalid-field": "invalid value",
		})
		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeTextGenerationChat(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid output.*for model.*")
		c.Assert(result, qt.IsNil)
	})
}
