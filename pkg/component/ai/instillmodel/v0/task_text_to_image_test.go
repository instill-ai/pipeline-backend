package instillmodel

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func TestExecuteTextToImage(t *testing.T) {
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

	// Test case 1: Successful text-to-image generation
	t.Run("successful text-to-image generation", func(t *testing.T) {
		// Create mock response with base64 image data
		mockImageData := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="

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
												"image": structpb.NewStringValue(mockImageData),
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
		textToImageInput := TextToImageInput{
			ModelName:      "test-ns/test-model/v1",
			Prompt:         "A beautiful sunset over the mountains",
			NegativePrompt: stringPtr("blurry, low quality"),
			AspectRatio:    stringPtr("16:9"),
			Samples:        intPtr(1),
			Seed:           intPtr(42),
		}

		inputStruct, err := base.ConvertToStructpb(textToImageInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeTextToImage(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure exists
		_, hasImages := result[0].Fields["images"]
		c.Assert(hasImages, qt.IsTrue)
	})

	// Test case 2: Multiple images generation
	t.Run("multiple images generation", func(t *testing.T) {
		// Create mock response with multiple images
		mockImageData1 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
		mockImageData2 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

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
												"image": structpb.NewStringValue(mockImageData1),
											},
										}),
										structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"image": structpb.NewStringValue(mockImageData2),
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
		textToImageInput := TextToImageInput{
			ModelName: "test-ns/test-model/v1",
			Prompt:    "A cat and a dog playing together",
			Samples:   intPtr(2),
		}

		inputStruct, err := base.ConvertToStructpb(textToImageInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeTextToImage(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure exists
		_, hasImages := result[0].Fields["images"]
		c.Assert(hasImages, qt.IsTrue)
	})

	// Test case 3: Empty inputs
	t.Run("empty inputs", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}
		inputs := []*structpb.Struct{}

		result, err := exec.executeTextToImage(mockClient, "test-ns", "test-model", "v1", inputs)

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

		result, err := exec.executeTextToImage(nil, "test-ns", "test-model", "v1", inputs)

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

		result, err := exec.executeTextToImage(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid output.*for model.*")
		c.Assert(result, qt.IsNil)
	})
}
