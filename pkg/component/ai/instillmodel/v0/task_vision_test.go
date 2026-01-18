package instillmodel

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

func TestExecuteVisionTask(t *testing.T) {
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

	// Test case 1: Successful vision task execution (classification)
	t.Run("successful classification", func(t *testing.T) {
		// Create mock response for classification
		mockResponse := &modelpb.TriggerNamespaceModelResponse{
			TaskOutputs: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"data": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"classes": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStringValue("cat"),
										structpb.NewStringValue("dog"),
									},
								}),
								"scores": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewNumberValue(0.95),
										structpb.NewNumberValue(0.05),
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

		// Create test input with base64 image
		mockImageBase64 := "data:image/jpeg;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=="
		visionInput := VisionInput{
			ModelName:   "test-ns/test-model/v1",
			ImageBase64: mockImageBase64,
		}

		inputStruct, err := base.ConvertToStructpb(visionInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeVisionTask(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure (raw data is returned for vision tasks)
		c.Assert(result[0].Fields["classes"], qt.Not(qt.IsNil))
		c.Assert(result[0].Fields["scores"], qt.Not(qt.IsNil))

		classes := result[0].Fields["classes"].GetListValue()
		c.Assert(classes.Values, qt.HasLen, 2)
		c.Assert(classes.Values[0].GetStringValue(), qt.Equals, "cat")
		c.Assert(classes.Values[1].GetStringValue(), qt.Equals, "dog")
	})

	// Test case 2: Successful detection task
	t.Run("successful detection", func(t *testing.T) {
		// Create mock response for detection
		mockResponse := &modelpb.TriggerNamespaceModelResponse{
			TaskOutputs: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"data": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"objects": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"category": structpb.NewStringValue("person"),
												"score":    structpb.NewNumberValue(0.98),
												"bounding_box": structpb.NewStructValue(&structpb.Struct{
													Fields: map[string]*structpb.Value{
														"left":   structpb.NewNumberValue(100),
														"top":    structpb.NewNumberValue(50),
														"width":  structpb.NewNumberValue(200),
														"height": structpb.NewNumberValue(300),
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
		visionInput := VisionInput{
			ModelName:   "test-ns/test-model/v1",
			ImageBase64: "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
		}

		inputStruct, err := base.ConvertToStructpb(visionInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeVisionTask(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure
		c.Assert(result[0].Fields["objects"], qt.Not(qt.IsNil))
		objects := result[0].Fields["objects"].GetListValue()
		c.Assert(objects.Values, qt.HasLen, 1)

		firstObject := objects.Values[0].GetStructValue()
		c.Assert(firstObject.Fields["category"].GetStringValue(), qt.Equals, "person")
		c.Assert(firstObject.Fields["score"].GetNumberValue(), qt.Equals, 0.98)
	})

	// Test case 3: Empty inputs
	t.Run("empty inputs", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}
		inputs := []*structpb.Struct{}

		result, err := exec.executeVisionTask(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid input.*")
		c.Assert(result, qt.IsNil)
	})

	// Test case 4: Nil gRPC client
	t.Run("nil grpc client", func(t *testing.T) {
		inputStruct, _ := structpb.NewStruct(map[string]any{
			"model-name":   "test-ns/test-model/v1",
			"image-base64": "test-image-data",
		})
		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeVisionTask(nil, "test-ns", "test-model", "v1", inputs)

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

		result, err := exec.executeVisionTask(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid output.*for model.*")
		c.Assert(result, qt.IsNil)
	})

	// Test case 6: Empty task outputs
	t.Run("empty task outputs", func(t *testing.T) {
		mockResponse := &modelpb.TriggerNamespaceModelResponse{
			TaskOutputs: []*structpb.Struct{},
		}

		mockClient := &mockModelPublicServiceClient{
			triggerResponse: mockResponse,
		}

		visionInput := VisionInput{
			ModelName:   "test-ns/test-model/v1",
			ImageBase64: "test-image",
		}

		inputStruct, err := base.ConvertToStructpb(visionInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeVisionTask(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid output.*")
		c.Assert(result, qt.IsNil)
	})
}
