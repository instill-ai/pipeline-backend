package instillartifact

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

func (e *execution) query(input *structpb.Struct) (*structpb.Struct, error) {
	inputStruct := QueryInput{}
	err := base.ConvertFromStructpb(input, &inputStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to convert input to struct: %w", err)
	}

	artifactClient := e.client

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(e.SystemVariables))

	// Validate file UID param. Empty UIDs will be filtered out, other invalid
	// strings will cause an error.
	fileUIDs := make([]string, 0, len(inputStruct.FileUIDs))
	for _, uid := range inputStruct.FileUIDs {
		if uid == "" {
			continue
		}

		if _, err := uuid.FromString(uid); err != nil {
			return nil, fmt.Errorf("invalid file UID %s: %w", uid, err)
		}

		fileUIDs = append(fileUIDs, uid)
	}

	queryRes, err := artifactClient.QuestionAnswering(ctx, &artifactpb.QuestionAnsweringRequest{
		NamespaceId: inputStruct.Namespace,
		CatalogId:   inputStruct.CatalogID,
		Question:    inputStruct.Question,
		TopK:        inputStruct.TopK,
		FileUids:    fileUIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to question answering: %w", err)
	}

	output := QueryOutput{
		Answer: queryRes.Answer,
		Chunks: make([]SimilarityChunk, 0, len(queryRes.GetSimilarChunks())),
	}

	for _, chunkPB := range queryRes.GetSimilarChunks() {
		output.Chunks = append(output.Chunks, SimilarityChunk{
			ChunkUID:        chunkPB.GetChunkUid(),
			SimilarityScore: chunkPB.GetSimilarityScore(),
			TextContent:     chunkPB.GetTextContent(),
			SourceFileName:  chunkPB.GetSourceFile(),
			SourceFileUID:   chunkPB.GetChunkMetadata().GetOriginalFileUid(),
		})
	}

	return base.ConvertToStructpb(output)
}
