package instillmodel

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
)

const maxPayloadSize int = 1024 * 1024 * 32

// initModelPublicServiceClient initializes a ModelPublicServiceClient instance
func initModelPublicServiceClient(serverURL string) (modelpb.ModelPublicServiceClient, *grpc.ClientConn) {
	var clientDialOpts grpc.DialOption

	if strings.HasPrefix(serverURL, "https://") {
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	serverURL = util.StripProtocolFromURL(serverURL)
	clientConn, err := grpc.NewClient(serverURL, clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxPayloadSize), grpc.MaxCallSendMsgSize(maxPayloadSize)))
	if err != nil {
		return nil, nil
	}

	return modelpb.NewModelPublicServiceClient(clientConn), clientConn
}

func trigger(gRPCClient modelpb.ModelPublicServiceClient, vars map[string]any, nsID string, modelID string, version string, taskInputs []*structpb.Struct) ([]*structpb.Struct, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, getRequestMetadata(vars))

	res, err := gRPCClient.TriggerModel(ctx, &modelpb.TriggerModelRequest{
		Name:       fmt.Sprintf("namespaces/%s/models/%s/versions/%s", nsID, modelID, version),
		TaskInputs: taskInputs,
	})
	if err != nil || res == nil {
		return nil, err
	}
	return res.TaskOutputs, nil
}
