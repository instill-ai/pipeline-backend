package handler

import (
	"context"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func extractFromMetadata(ctx context.Context, key string) ([]string, bool) {
	data, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return []string{}, false
	}
	return data[strings.ToLower(key)], true
}

func getOwnerID(ctx context.Context) (uuid.UUID, error) {
	metadatas, ok := extractFromMetadata(ctx, "owner_id")
	if ok {
		if len(metadatas) == 0 {
			return uuid.UUID{}, status.Error(codes.FailedPrecondition, "owner_id not found in your request")
		}

		ownerUUID, err := uuid.FromString(metadatas[0])
		if err != nil {
			return uuid.UUID{}, err
		}

		return ownerUUID, nil
	}

	return uuid.UUID{}, status.Error(codes.FailedPrecondition, "Error when extract metadata")
}
