package instillartifact

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactPB "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func (e *execution) query(input *structpb.Struct) (*structpb.Struct, error) {

	inputStruct := QueryInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient, connection := e.client, e.connection

	defer connection.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	queryRes, err := artifactClient.QuestionAnswering(ctx, &artifactPB.QuestionAnsweringRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		Question:    inputStruct.Question,
		TopK:        inputStruct.TopK,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to question answering: %w", err)
	}

	output := QueryOutput{
		Answer: queryRes.Answer,
		Chunks: []SimilarityChunk{},
	}

	for _, chunkPB := range queryRes.SimilarChunks {
		output.Chunks = append(output.Chunks, SimilarityChunk{
			ChunkUID:        chunkPB.ChunkUid,
			SimilarityScore: chunkPB.SimilarityScore,
			TextContent:     chunkPB.TextContent,
			SourceFileName:  chunkPB.SourceFile,
		})
	}

	return base.ConvertToStructpb(output)
}
