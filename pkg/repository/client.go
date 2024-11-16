package repository

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/external"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	miniox "github.com/instill-ai/x/minio"
)

// BlobStorage is an interface for fetching files from blob storage
type BlobStorage interface {
	// GetFile fetches a file from blob storage
	GetFile(ctx context.Context, bucketName, objectPath string) (data []byte, contentType string, err error)
}

// Clients is a struct that holds all the clients
type Clients struct {
	// ArtifactPublicServiceClient can fetch metadata from artifact service
	ArtifactPrivateServiceClient artifactpb.ArtifactPrivateServiceClient
	// GRPCConn is the connection to the artifact private service
	GRPCConn *grpc.ClientConn
	// BlobStorageClient can fetch files from blob storage
	BlobStorageClient BlobStorage
}

// Close closes all connections held by the clients
func (c *Clients) Close() error {
	if c.GRPCConn != nil {
		return c.GRPCConn.Close()
	}
	return nil
}

// GetClients returns an instance of Clients
func GetClients(ctx context.Context, logger *zap.Logger) (*Clients, error) {
	clients := Clients{}
	artifactPrivateServiceClient, privConn := external.InitArtifactPrivateServiceClient(ctx)
	if artifactPrivateServiceClient == nil {
		return nil, fmt.Errorf("artifact private service client is nil")
	}
	clients.GRPCConn = privConn
	clients.ArtifactPrivateServiceClient = artifactPrivateServiceClient

	minioClient, err := miniox.NewMinioClient(ctx, &config.Config.Minio, logger)
	if err != nil {
		return nil, err
	}
	clients.BlobStorageClient = minioClient

	return &clients, nil
}
