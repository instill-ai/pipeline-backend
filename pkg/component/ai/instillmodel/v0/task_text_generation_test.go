package instillmodel

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func TestExecuteTextGeneration(t *testing.T) {
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

	// Test case 1: Successful text generation
	t.Run("successful text generation", func(t *testing.T) {
		// Create mock response
		mockResponse := &modelpb.TriggerNamespaceModelResponse{
			TaskOutputs: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"data": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"choices": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"content": structpb.NewStringValue("Generated text response"),
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
		textGenInput := TextGenerationInput{
			ModelName:     "test-ns/test-model/v1",
			Prompt:        "Hello, how are you?",
			SystemMessage: stringPtr("You are a helpful assistant."),
			MaxNewTokens:  intPtr(100),
			Temperature:   float32Ptr(0.7),
			Seed:          intPtr(42),
		}

		inputStruct, err := base.ConvertToStructpb(textGenInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeTextGeneration(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure
		var output TextGenerationOutput
		err = base.ConvertFromStructpb(result[0], &output)
		c.Assert(err, qt.IsNil)
		c.Assert(output.Text, qt.Equals, "Generated text response")
	})

	// Test case 2: Empty inputs
	t.Run("empty inputs", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}
		inputs := []*structpb.Struct{}

		result, err := exec.executeTextGeneration(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid input.*")
		c.Assert(result, qt.IsNil)
	})

	// Test case 3: Nil gRPC client
	t.Run("nil grpc client", func(t *testing.T) {
		inputStruct, _ := structpb.NewStruct(map[string]any{
			"model-name": "test-ns/test-model/v1",
			"prompt":     "test prompt",
		})
		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeTextGeneration(nil, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "uninitialized client")
		c.Assert(result, qt.IsNil)
	})

	// Test case 4: Input conversion error
	t.Run("input conversion error", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}

		// Create invalid input structure
		inputStruct, _ := structpb.NewStruct(map[string]any{
			"invalid-field": "invalid value",
		})
		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeTextGeneration(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid output.*for model.*")
		c.Assert(result, qt.IsNil)
	})
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func float32Ptr(f float32) *float32 {
	return &f
}
