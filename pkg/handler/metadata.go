package handler

import (
	"context"
	"strings"

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

func getOwner(ctx context.Context) (string, error) {
	metadatas, ok := extractFromMetadata(ctx, "owner")
	if ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.FailedPrecondition, "owner not found in your request")
		}
		return metadatas[0], nil
	}
	return "", status.Error(codes.FailedPrecondition, "Error when extract metadata")
}
