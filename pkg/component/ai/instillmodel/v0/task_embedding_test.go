package instillmodel

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

// mockModelPublicServiceClient is a mock implementation of ModelPublicServiceClient
type mockModelPublicServiceClient struct {
	modelpb.ModelPublicServiceClient
	triggerResponse *modelpb.TriggerModelResponse
	triggerError    error
}

func (m *mockModelPublicServiceClient) TriggerModel(ctx context.Context, req *modelpb.TriggerModelRequest, opts ...grpc.CallOption) (*modelpb.TriggerModelResponse, error) {
	if m.triggerError != nil {
		return nil, m.triggerError
	}
	return m.triggerResponse, nil
}

func TestExecuteEmbedding(t *testing.T) {
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

	// Test case 1: Successful embedding execution
	t.Run("successful embedding execution", func(t *testing.T) {
		// Create mock response
		embeddingVector := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
		vectorValues := make([]*structpb.Value, len(embeddingVector))
		for i, v := range embeddingVector {
			vectorValues[i] = structpb.NewNumberValue(v)
		}

		mockResponse := &modelpb.TriggerModelResponse{
			TaskOutputs: []*structpb.Struct{
				{
					Fields: map[string]*structpb.Value{
						"data": structpb.NewStructValue(&structpb.Struct{
							Fields: map[string]*structpb.Value{
								"embeddings": structpb.NewListValue(&structpb.ListValue{
									Values: []*structpb.Value{
										structpb.NewStructValue(&structpb.Struct{
											Fields: map[string]*structpb.Value{
												"embedding": structpb.NewListValue(&structpb.ListValue{
													Values: vectorValues,
												}),
												"created": structpb.NewNumberValue(float64(time.Now().Unix())),
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
		embeddingInput := EmbeddingInput{
			Data: EmbeddingInputData{
				Model: "test-ns/test-model/v1",
				Embeddings: []InputEmbedding{
					{
						Type: "text",
						Text: "Hello, world!",
					},
				},
			},
			Parameter: EmbeddingParameter{
				Format:     "float",
				Dimensions: 512,
				InputType:  "query",
				Truncate:   "End",
			},
		}

		inputStruct, err := base.ConvertToStructpb(embeddingInput)
		c.Assert(err, qt.IsNil)

		inputs := []*structpb.Struct{inputStruct}

		// Execute
		result, err := exec.executeEmbedding(mockClient, "test-ns", "test-model", "v1", inputs)

		// Assertions
		c.Assert(err, qt.IsNil)
		c.Assert(result, qt.HasLen, 1)

		// Verify output structure
		var output EmbeddingOutput
		err = base.ConvertFromStructpb(result[0], &output)
		c.Assert(err, qt.IsNil)
		c.Assert(output.Data.Embeddings, qt.HasLen, 1)
		c.Assert(output.Data.Embeddings[0].Index, qt.Equals, 0)
		c.Assert(output.Data.Embeddings[0].Vector, qt.HasLen, 5)
		c.Assert(output.Data.Embeddings[0].Created, qt.Not(qt.Equals), 0)
	})

	// Test case 2: Empty inputs
	t.Run("empty inputs", func(t *testing.T) {
		mockClient := &mockModelPublicServiceClient{}
		inputs := []*structpb.Struct{}

		result, err := exec.executeEmbedding(mockClient, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "invalid input.*")
		c.Assert(result, qt.IsNil)
	})

	// Test case 3: Nil gRPC client
	t.Run("nil grpc client", func(t *testing.T) {
		inputStruct, _ := structpb.NewStruct(map[string]any{
			"data": map[string]any{
				"model": "test-ns/test-model/v1",
				"embeddings": []any{
					map[string]any{
						"type": "text",
						"text": "test",
					},
				},
			},
			"parameter": map[string]any{
				"format": "float",
			},
		})
		inputs := []*structpb.Struct{inputStruct}

		result, err := exec.executeEmbedding(nil, "test-ns", "test-model", "v1", inputs)

		c.Assert(err, qt.ErrorMatches, "uninitialized client")
		c.Assert(result, qt.IsNil)
	})
}
