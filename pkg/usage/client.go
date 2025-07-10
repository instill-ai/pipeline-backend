package usage

import (
	"context"
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/pipeline-backend/config"

	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
	logx "github.com/instill-ai/x/log"
)

// InitUsageServiceClient initializes a UsageServiceClient instance (no mTLS)
func InitUsageServiceClient(ctx context.Context) (usagepb.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logx.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var err error
	if config.Config.Server.Usage.TLSEnabled {
		tlsConfig := &tls.Config{}
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(
		fmt.Sprintf("%v:%v", config.Config.Server.Usage.Host, config.Config.Server.Usage.Port),
		clientDialOpts,
	)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return usagepb.NewUsageServiceClient(clientConn), clientConn
}
