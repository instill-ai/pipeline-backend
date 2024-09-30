package cohere

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	cohereSDK "github.com/cohere-ai/cohere-go/v2"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type EmbeddingInput struct {
	Text          string `json:"text"`
	ModelName     string `json:"model-name"`
	InputType     string `json:"input-type"`
	EmbeddingType string `json:"embedding-type"`
}

type EmbeddingFloatOutput struct {
	Usage     embedUsage `json:"usage"`
	Embedding []float64  `json:"embedding"`
}

type EmbeddingIntOutput struct {
	Usage     embedUsage `json:"usage"`
	Embedding []int      `json:"embedding"`
}

type embedUsage struct {
	Tokens int `json:"tokens"`
}

func (e *execution) taskEmbedding(in *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := EmbeddingInput{}
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("error generating input struct: %v", err)
	}

	if IsEmbeddingOutputInt(inputStruct.EmbeddingType) {
		tokenCount, embedding, err := processWithIntOutput(e, inputStruct)
		if err != nil {
			return nil, err
		}

		outputStruct := EmbeddingIntOutput{
			Usage: embedUsage{
				Tokens: tokenCount,
			},
			Embedding: embedding,
		}
		output, err := base.ConvertToStructpb(outputStruct)
		if err != nil {
			return nil, err
		}
		return output, nil
	}

	tokenCount, embedding, err := processWithFloatOutput(e, inputStruct)
	if err != nil {
		return nil, err
	}
	outputStruct := EmbeddingFloatOutput{
		Usage: embedUsage{
			Tokens: tokenCount,
		},
		Embedding: embedding,
	}
	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil

}

func IsEmbeddingOutputInt(embeddingType string) bool {
	return embeddingType == "int8" || embeddingType == "uint8" || embeddingType == "binary" || embeddingType == "ubinary"
}

func processWithIntOutput(e *execution, inputStruct EmbeddingInput) (tokenCount int, embedding []int, err error) {
	req := cohereSDK.EmbedRequest{
		Texts:          []string{inputStruct.Text},
		Model:          &inputStruct.ModelName,
		InputType:      (*cohereSDK.EmbedInputType)(&inputStruct.InputType),
		EmbeddingTypes: []cohereSDK.EmbeddingType{cohereSDK.EmbeddingType(inputStruct.EmbeddingType)},
	}
	resp, err := e.client.generateEmbedding(req)

	if err != nil {
		return 0, nil, err
	}

	embeddingResult, err := getIntEmbedding(resp, inputStruct.EmbeddingType)
	if err != nil {
		return 0, nil, err
	}
	return getBillingTokens(resp, inputStruct.EmbeddingType), embeddingResult, nil
}

func processWithFloatOutput(e *execution, inputStruct EmbeddingInput) (tokenCount int, embedding []float64, err error) {
	embeddingTypeArray := []cohereSDK.EmbeddingType{}
	if inputStruct.EmbeddingType == "float" {
		embeddingTypeArray = append(embeddingTypeArray, cohereSDK.EmbeddingTypeFloat)
	}
	req := cohereSDK.EmbedRequest{
		Texts:          []string{inputStruct.Text},
		Model:          &inputStruct.ModelName,
		InputType:      (*cohereSDK.EmbedInputType)(&inputStruct.InputType),
		EmbeddingTypes: embeddingTypeArray,
	}
	resp, err := e.client.generateEmbedding(req)

	if err != nil {
		return 0, nil, err
	}

	embeddingResult := getFloatEmbedding(resp, inputStruct.EmbeddingType)

	return getBillingTokens(resp, inputStruct.EmbeddingType), embeddingResult, nil

}

func getIntEmbedding(resp cohereSDK.EmbedResponse, embeddingType string) ([]int, error) {
	switch embeddingType {
	case "int8":
		return resp.EmbeddingsByType.Embeddings.Int8[0], nil
	case "uint8":
		return resp.EmbeddingsByType.Embeddings.Uint8[0], nil
	case "binary":
		return resp.EmbeddingsByType.Embeddings.Binary[0], nil
	case "ubinary":
		return resp.EmbeddingsByType.Embeddings.Ubinary[0], nil
	}
	return nil, fmt.Errorf("invalid embedding type: %s", embeddingType)
}

func getFloatEmbedding(resp cohereSDK.EmbedResponse, embeddingType string) []float64 {
	if embeddingType == "float" {
		return resp.EmbeddingsByType.Embeddings.Float[0]
	} else {
		return resp.EmbeddingsFloats.Embeddings[0]
	}
}

func getBillingTokens(resp cohereSDK.EmbedResponse, embeddingType string) int {
	if IsEmbeddingOutputInt(embeddingType) || embeddingType == "float" {
		return int(*resp.EmbeddingsByType.Meta.BilledUnits.InputTokens)
	} else {
		return int(*resp.EmbeddingsFloats.Meta.BilledUnits.InputTokens)
	}
}
