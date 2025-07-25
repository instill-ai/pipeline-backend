package instillartifact

import (
	"crypto/tls"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/pipeline-backend/pkg/component/internal/util"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
)

const maxPayloadSize int = 1024 * 1024 * 32

func initArtifactClient(serverURL string) (pbClient artifactpb.ArtifactPublicServiceClient, connection Connection, err error) {
	var clientDialOpts grpc.DialOption

	if strings.HasPrefix(serverURL, "https://") {
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	serverURL = util.StripProtocolFromURL(serverURL)
	clientConn, err := grpc.NewClient(serverURL, clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxPayloadSize), grpc.MaxCallSendMsgSize(maxPayloadSize)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create client connection: %w", err)
	}

	return artifactpb.NewArtifactPublicServiceClient(clientConn), clientConn, nil
}

func getArtifactServerURL(vars map[string]any) string {
	if v, ok := vars["__ARTIFACT_BACKEND"]; ok {
		return v.(string)
	}
	return ""
}

func getRequestMetadata(vars map[string]any) metadata.MD {
	md := metadata.Pairs(
		"Authorization", util.GetHeaderAuthorization(vars),
		"Instill-User-Uid", util.GetInstillUserUID(vars),
		"Instill-Auth-Type", "user",
	)

	originalHeader := util.GetOriginalHeader(vars)
	for k, v := range originalHeader {
		md.Set(k, v.(string))
	}

	if requester := util.GetInstillRequesterUID(vars); requester != "" {
		md.Set("Instill-Requester-Uid", requester)
	}

	return md
}
