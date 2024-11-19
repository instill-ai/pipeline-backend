package external

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"regexp"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-resty/resty/v2"
	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
)

// InitMgmtPublicServiceClient initialises a MgmtPublicServiceClient instance
func InitMgmtPublicServiceClient(ctx context.Context) (mgmtpb.MgmtPublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.MgmtBackend.HTTPS.Cert != "" && config.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.MgmtBackend.HTTPS.Cert, config.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return mgmtpb.NewMgmtPublicServiceClient(clientConn), clientConn
}

// InitMgmtPrivateServiceClient initialises a MgmtPrivateServiceClient instance
func InitMgmtPrivateServiceClient(ctx context.Context) (mgmtpb.MgmtPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.MgmtBackend.HTTPS.Cert != "" && config.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.MgmtBackend.HTTPS.Cert, config.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PrivatePort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return mgmtpb.NewMgmtPrivateServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initialises a UsageServiceClient instance (no mTLS)
func InitUsageServiceClient(ctx context.Context) (usagepb.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var err error
	if config.Config.Server.Usage.TLSEnabled {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.Server.Usage.Host, config.Config.Server.Usage.Port), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return usagepb.NewUsageServiceClient(clientConn), clientConn
}

// InitArtifactPublicServiceClient initialises a ArtifactPublicServiceClient instance
func InitArtifactPublicServiceClient(ctx context.Context) (artifactpb.ArtifactPublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.ArtifactBackend.HTTPS.Cert != "" && config.Config.ArtifactBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.ArtifactBackend.HTTPS.Cert, config.Config.ArtifactBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.ArtifactBackend.Host, config.Config.ArtifactBackend.PublicPort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return artifactpb.NewArtifactPublicServiceClient(clientConn), clientConn
}

// InitArtifactPrivateServiceClient initialises a ArtifactPrivateServiceClient instance
func InitArtifactPrivateServiceClient(ctx context.Context) (artifactpb.ArtifactPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.ArtifactBackend.HTTPS.Cert != "" && config.Config.ArtifactBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.ArtifactBackend.HTTPS.Cert, config.Config.ArtifactBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf("%v:%v", config.Config.ArtifactBackend.Host, config.Config.ArtifactBackend.PrivatePort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return artifactpb.NewArtifactPrivateServiceClient(clientConn), clientConn
}

type BinaryFetcher interface {
	FetchFromURL(ctx context.Context, url string) (body []byte, contentType string, filename string, err error)
}

type binaryFetcher struct {
	httpClient *resty.Client
}

func NewBinaryFetcher() BinaryFetcher {
	return &binaryFetcher{
		httpClient: resty.New().SetRetryCount(3),
	}
}

func (f *binaryFetcher) FetchFromURL(ctx context.Context, url string) (body []byte, contentType string, filename string, err error) {

	if strings.HasPrefix(url, "data:") {
		return f.convertDataURIToBytes(url)
	}

	var resp *resty.Response
	resp, err = f.httpClient.R().SetContext(ctx).Get(url)
	if err != nil {
		return
	}

	body = resp.Body()
	contentType = strings.Split(mimetype.Detect(body).String(), ";")[0]

	if disposition := resp.Header().Get("Content-Disposition"); disposition == "" {
		if strings.HasPrefix(disposition, "attachment") {
			if _, params, err := mime.ParseMediaType(disposition); err == nil {
				filename = params["filename"]
			}
		}
	}

	return
}

func (f *binaryFetcher) convertDataURIToBytes(url string) (b []byte, contentType string, filename string, err error) {
	slices := strings.Split(url, ",")
	if len(slices) == 1 {
		b, err = base64.StdEncoding.DecodeString(url)
		if err != nil {
			return
		}
		contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	} else {
		mime := strings.Split(slices[0], ":")
		tags := ""
		contentType, tags, _ = strings.Cut(mime[1], ";")
		b, err = base64.StdEncoding.DecodeString(slices[1])
		if err != nil {
			return
		}
		for _, tag := range strings.Split(tags, ";") {
			key, value, _ := strings.Cut(tag, "=")
			if key == "filename" || key == "fileName" || key == "file-name" {
				filename = value
			}
		}
	}
	return b, contentType, filename, nil
}

// Pattern matches: https://{domain}/v1alpha/namespaces/{namespace}/blob-urls/{uid}
var minioURLPattern = regexp.MustCompile(`https?://[^/]+/v1alpha/namespaces/[^/]+/blob-urls/([^/]+)$`)

// ArtifactBinaryFetcher fetches binary data from a URL.
// If that URL comes from an object uploaded on Instill Artifact,
// it uses the blob storage client directly to avoid egress costs.
type artifactBinaryFetcher struct {
	binaryFetcher  *binaryFetcher
	artifactClient artifactpb.ArtifactPrivateServiceClient // By having this injected, main.go is responsible of closing the connection.
	minIOClient    BlobStorage
}

func NewArtifactBinaryFetcher(ac artifactpb.ArtifactPrivateServiceClient, mc BlobStorage) BinaryFetcher {
	return &artifactBinaryFetcher{
		binaryFetcher: &binaryFetcher{
			httpClient: resty.New().SetRetryCount(3),
		},
		artifactClient: ac,
		minIOClient:    mc,
	}
}

func (f *artifactBinaryFetcher) FetchFromURL(ctx context.Context, url string) (b []byte, contentType string, filename string, err error) {
	if strings.HasPrefix(url, "data:") {
		return f.binaryFetcher.convertDataURIToBytes(url)
	}
	if matches := minioURLPattern.FindStringSubmatch(url); matches != nil {
		if len(matches) < 2 {
			err = fmt.Errorf("invalid blob storage url: %s", url)
			return
		}

		return f.fetchFromBlobStorage(ctx, uuid.FromStringOrNil(matches[1]))
	}
	return f.binaryFetcher.FetchFromURL(ctx, url)
}

func (f *artifactBinaryFetcher) fetchFromBlobStorage(ctx context.Context, urlUID uuid.UUID) (b []byte, contentType string, filename string, err error) {

	objectURLRes, err := f.artifactClient.GetObjectURL(ctx, &artifactpb.GetObjectURLRequest{
		Uid: urlUID.String(),
	})
	if err != nil {
		return nil, "", "", err
	}

	objectUID := objectURLRes.ObjectUrl.ObjectUid

	objectRes, err := f.artifactClient.GetObject(ctx, &artifactpb.GetObjectRequest{
		Uid: objectUID,
	})

	if err != nil {
		return nil, "", "", err
	}

	// TODO: we have agreed on to add the bucket name in pb.Object
	// After the contract is updated, we have to replace it
	bucketName := "instill-ai-blob"
	objectPath := *objectRes.Object.Path

	b, _, err = f.minIOClient.GetFile(ctx, bucketName, objectPath)
	if err != nil {
		return nil, "", "", err
	}
	contentType = strings.Split(mimetype.Detect(b).String(), ";")[0]
	return b, contentType, objectRes.Object.Name, nil
}

// BlobStorage is an interface for fetching files from blob storage
// TODO: we should put this interface in x package
type BlobStorage interface {
	// GetFile fetches a file from blob storage
	GetFile(ctx context.Context, bucketName, objectPath string) (data []byte, contentType string, err error)
}
