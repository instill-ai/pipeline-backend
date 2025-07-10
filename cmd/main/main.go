package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	openfga "github.com/openfga/api/proto/openfga/v1"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/middleware"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/usage"
	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/temporal"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	grpcclientx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
)

const gracefulShutdownWaitPeriod = 15 * time.Second
const gracefulShutdownTimeout = 60 * time.Minute

var (
	// These variables might be overridden at buildtime.
	serviceVersion = "dev"
	serviceName    = "pipeline-backend"

	propagator propagation.TextMapPropagator
)

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logx.Debug = config.Config.Server.Debug
	logger, _ := logx.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	// verbosity 3 will avoid [transport] from emitting
	grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 3)

	// Get gRPC server options and credentials
	grpcServerOpts, creds, tlsConfig := newGrpcOptionAndCreds(logger)

	privateGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(privateGrpcS)

	publicGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(publicGrpcS)

	// Initialize all clients
	pipelinePublicServiceClient, mgmtPublicServiceClient, mgmtPrivateServiceClient,
		artifactPublicServiceClient, artifactPrivateServiceClient, redisClient, db,
		minIOClient, minIOFileGetter, aclClient, temporalClient, closeClients := newClients(ctx, logger)
	defer closeClients()

	// Keep NewArtifactBinaryFetcher as requested
	binaryFetcher := external.NewArtifactBinaryFetcher(artifactPrivateServiceClient, minIOFileGetter)

	compStore := componentstore.Init(componentstore.InitParams{
		Logger:         logger,
		Secrets:        config.Config.Component.Secrets,
		BinaryFetcher:  binaryFetcher,
		TemporalClient: temporalClient,
	})

	pubsub := pubsub.NewRedisPubSub(redisClient)
	ms := memory.NewStore(pubsub, minIOClient.WithLogger(logger))

	repo := repository.NewRepository(db, redisClient)
	service := service.NewService(
		repo,
		redisClient,
		temporalClient,
		&aclClient,
		service.NewConverter(service.ConverterConfig{
			MgmtClient:      mgmtPrivateServiceClient,
			RedisClient:     redisClient,
			ACLClient:       &aclClient,
			Repository:      repo,
			InstillCoreHost: config.Config.Server.InstillCoreHost,
			ComponentStore:  compStore,
		}),
		mgmtPublicServiceClient,
		mgmtPrivateServiceClient,
		minIOClient,
		compStore,
		ms,
		service.NewRetentionHandler(),
		binaryFetcher,
		artifactPublicServiceClient,
		artifactPrivateServiceClient,
	)

	privateHandler := handler.NewPrivateHandler(service)
	pipelinepb.RegisterPipelinePrivateServiceServer(privateGrpcS, privateHandler)

	publicHandler := handler.NewPublicHandler(service)
	pipelinepb.RegisterPipelinePublicServiceServer(publicGrpcS, publicHandler)

	privateServeMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.HTTPResponseModifier),
		runtime.WithErrorHandler(middleware.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(middleware.CustomMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	publicServeMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.HTTPResponseModifier),
		runtime.WithErrorHandler(middleware.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(middleware.CustomMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	// Start usage reporter
	var usg usage.Usage
	if config.Config.Server.Usage.Enabled {
		usageServiceClient, usageServiceClientConn := usage.InitUsageServiceClient(ctx)
		if usageServiceClientConn != nil {
			defer usageServiceClientConn.Close()
			logger.Info("try to start usage reporter")
			go func() {
				for {
					usg = usage.NewUsage(ctx, repo, mgmtPrivateServiceClient, redisClient, usageServiceClient, serviceVersion)
					if usg != nil {
						usg.StartReporter(ctx)
						logger.Info("usage reporter started")
						break
					}
					logger.Warn("retry to start usage reporter after 5 minutes")
					time.Sleep(5 * time.Minute)
				}
			}()
		}
	}

	// Start gRPC server
	var dialOpts []grpc.DialOption
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize))}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize))}
	}

	if err := pipelinepb.RegisterPipelinePrivateServiceHandlerFromEndpoint(ctx, privateServeMux, fmt.Sprintf(":%v", config.Config.Server.PrivatePort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	if err := pipelinepb.RegisterPipelinePublicServiceHandlerFromEndpoint(ctx, publicServeMux, fmt.Sprintf(":%v", config.Config.Server.PublicPort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	streamingHandler := handler.NewStreamingHandler(publicServeMux, pipelinePublicServiceClient, pubsub)
	if err := publicServeMux.HandlePath("POST", "/v1beta/*/{namespaceID=*}/pipelines/{pipelineID=*}/trigger", streamingHandler.HandleTrigger); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1beta/*/{namespaceID=*}/pipelines/{pipelineID=*}/triggerAsync", streamingHandler.HandleTriggerAsync); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1beta/*/{namespaceID=*}/pipelines/{pipelineID=*}/releases/{releaseID=*}/trigger", streamingHandler.HandleTriggerRelease); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1beta/*/{namespaceID=*}/pipelines/{pipelineID=*}/releases/{releaseID=*}/triggerAsync", streamingHandler.HandleTriggerAsyncRelease); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("GET", "/v1beta/*/{namespaceID=*}/pipelines/{pipelineID=*}/image", middleware.HandleProfileImage(service, repo)); err != nil {
		logger.Fatal(err.Error())
	}

	privateHTTPServer := &http.Server{
		Addr:              fmt.Sprintf(":%v", config.Config.Server.PrivatePort),
		Handler:           grpcHandlerFunc(privateGrpcS, privateServeMux),
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Millisecond,
	}

	publicHTTPServer := &http.Server{
		Addr:              fmt.Sprintf(":%v", config.Config.Server.PublicPort),
		Handler:           grpcHandlerFunc(publicGrpcS, publicServeMux),
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Millisecond,
	}

	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 5 seconds.
	quitSig := make(chan os.Signal, 1)
	errSig := make(chan error)
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		go func() {
			if err := privateHTTPServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
		go func() {
			if err := publicHTTPServer.ListenAndServeTLS(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key); err != nil {
				errSig <- err
			}
		}()
	} else {
		go func() {
			if err := privateHTTPServer.ListenAndServe(); err != nil {
				errSig <- err
			}
		}()
		go func() {
			if err := publicHTTPServer.ListenAndServe(); err != nil {
				errSig <- err
			}
		}()
	}

	logger.Info("gRPC server is running.")

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)
	publicHandler.SetReadiness(true)
	select {
	case err := <-errSig:
		logger.Error(fmt.Sprintf("Fatal error: %v\n", err))
	case <-quitSig:
		// When the server receives a SIGTERM, Kubernetes services may still
		// consider it available. To handle this properly, we should:
		// 1. Set the readiness probe to false to signal the server is no longer
		// ready to receive traffic.
		// 2. Sleep for a brief period to allow Kubernetes to stop forwarding
		// requests.
		// 3. Continue processing any in-flight requests.
		// 4. After Kubernetes stops sending new traffic, initiate the server's
		// shutdown process.

		logger.Info("Shutting down server...")
		logger.Info("Stop receiving request...")
		time.Sleep(gracefulShutdownWaitPeriod)
		if config.Config.Server.Usage.Enabled && usg != nil {
			usg.TriggerSingleReporter(ctx)
		}
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer shutdownCancel()

		logger.Info("Shutting down HTTP server...")
		_ = privateHTTPServer.Shutdown(shutdownCtx)
		_ = publicHTTPServer.Shutdown(shutdownCtx)
		logger.Info("Shutting down gRPC server...")
		privateGrpcS.GracefulStop()
		publicGrpcS.GracefulStop()

	}
}

func grpcHandlerFunc(grpcServer *grpc.Server, gwHandler http.Handler) http.Handler {
	return h2c.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			propagator = b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
			r = r.WithContext(ctx)

			if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
				grpcServer.ServeHTTP(w, r)
			} else {
				gwHandler.ServeHTTP(w, r)
			}
		}),
		&http2.Server{},
	)
}

func newClients(ctx context.Context, logger *zap.Logger) (
	pipelinepb.PipelinePublicServiceClient,
	mgmtpb.MgmtPublicServiceClient,
	mgmtpb.MgmtPrivateServiceClient,
	artifactpb.ArtifactPublicServiceClient,
	artifactpb.ArtifactPrivateServiceClient,
	*redis.Client,
	*gorm.DB,
	minio.Client,
	*minio.FileGetter,
	acl.ACLClient,
	temporalclient.Client,
	func(),
) {
	closeFuncs := map[string]func() error{}

	// Initialize external service clients
	pipelinePublicServiceClient, pipelinePublicClose, err := grpcclientx.NewPipelinePublicClient(client.ServiceConfig{
		Host:       config.Config.Server.InstanceID,
		PublicPort: config.Config.Server.PublicPort,
		HTTPS: client.HTTPSConfig{
			Cert: config.Config.Server.HTTPS.Cert,
			Key:  config.Config.Server.HTTPS.Key,
		},
	})
	if err != nil {
		logger.Fatal("failed to create pipeline public service client", zap.Error(err))
	}
	closeFuncs["pipelinePublic"] = pipelinePublicClose

	mgmtPublicServiceClient, mgmtPublicClose, err := grpcclientx.NewMgmtPublicClient(config.Config.MgmtBackend)
	if err != nil {
		logger.Fatal("failed to create mgmt public service client", zap.Error(err))
	}
	closeFuncs["mgmtPublic"] = mgmtPublicClose

	mgmtPrivateServiceClient, mgmtPrivateClose, err := grpcclientx.NewMgmtPrivateClient(config.Config.MgmtBackend)
	if err != nil {
		logger.Fatal("failed to create mgmt private service client", zap.Error(err))
	}
	closeFuncs["mgmtPrivate"] = mgmtPrivateClose

	artifactPublicServiceClient, artifactPublicClose, err := grpcclientx.NewArtifactPublicClient(config.Config.ArtifactBackend)
	if err != nil {
		logger.Fatal("failed to create artifact public service client", zap.Error(err))
	}
	closeFuncs["artifactPublic"] = artifactPublicClose

	artifactPrivateServiceClient, artifactPrivateClose, err := grpcclientx.NewArtifactPrivateClient(config.Config.ArtifactBackend)
	if err != nil {
		logger.Fatal("failed to create artifact private service client", zap.Error(err))
	}
	closeFuncs["artifactPrivate"] = artifactPrivateClose

	// Initialize database
	db := database.GetSharedConnection()
	closeFuncs["database"] = func() error {
		database.Close(db)
		return nil
	}

	// Initialize Temporal client
	temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
		Tracer:            otel.Tracer(serviceName + "-temporal"),
		TextMapPropagator: otel.GetTextMapPropagator(),
	})
	if err != nil {
		logger.Fatal("Unable to create temporal tracing interceptor", zap.Error(err))
	}

	temporalClientOptions, err := temporal.ClientOptions(config.Config.Temporal, logger)
	if err != nil {
		logger.Fatal("Unable to build Temporal client options", zap.Error(err))
	}

	temporalClientOptions.Interceptors = []interceptor.ClientInterceptor{temporalTracingInterceptor}
	temporalClient, err := temporalclient.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal("Unable to create Temporal client", zap.Error(err))
	}
	closeFuncs["temporal"] = func() error {
		temporalClient.Close()
		return nil
	}

	// Initialize redis client
	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	closeFuncs["redis"] = redisClient.Close

	// Initialize ACL client
	fgaClient, fgaClientConn := acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Host, config.Config.OpenFGA.Port)
	if fgaClientConn != nil {
		closeFuncs["fga"] = fgaClientConn.Close
	}

	var fgaReplicaClient openfga.OpenFGAServiceClient
	if config.Config.OpenFGA.Replica.Host != "" {
		var fgaReplicaClientConn *grpc.ClientConn
		fgaReplicaClient, fgaReplicaClientConn = acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Replica.Host, config.Config.OpenFGA.Replica.Port)
		if fgaReplicaClientConn != nil {
			closeFuncs["fgaReplica"] = fgaReplicaClientConn.Close
		}
	}

	aclClient := acl.NewACLClient(fgaClient, fgaReplicaClient, redisClient)

	// Initialize MinIO client
	minIOParams := minio.ClientParams{
		Config: config.Config.Minio,
		Logger: logger,
		AppInfo: minio.AppInfo{
			Name:    serviceName,
			Version: serviceVersion,
		},
	}
	minIOFileGetter, err := minio.NewFileGetter(minIOParams)
	if err != nil {
		logger.Fatal("Failed to create MinIO file getter", zap.Error(err))
	}

	retentionHandler := service.NewRetentionHandler()
	minIOParams.ExpiryRules = retentionHandler.ListExpiryRules()
	minIOClient, err := minio.NewMinIOClientAndInitBucket(ctx, minIOParams)
	if err != nil {
		logger.Fatal("failed to create MinIO client", zap.Error(err))
	}

	closer := func() {
		for conn, fn := range closeFuncs {
			if err := fn(); err != nil {
				logger.Error("Failed to close conn", zap.Error(err), zap.String("conn", conn))
			}
		}
	}

	return pipelinePublicServiceClient, mgmtPublicServiceClient, mgmtPrivateServiceClient,
		artifactPublicServiceClient, artifactPrivateServiceClient, redisClient, db,
		minIOClient, minIOFileGetter, aclClient, temporalClient, closer
}

func newGrpcOptionAndCreds(logger *zap.Logger) ([]grpc.ServerOption, credentials.TransportCredentials, *tls.Config) {
	// Shared options for the logger, with a custom gRPC code to log level function.
	opts := []grpczap.Option{
		grpczap.WithDecider(func(fullMethodName string, err error) bool {
			// will not log gRPC calls if it was a call to liveness or readiness and no error was raised
			if err == nil {
				if match, _ := regexp.MatchString("pipeline.pipeline.v1beta.PipelinePublicService/.*ness$", fullMethodName); match {
					return false
				}
				// stop logging successful private function calls
				if match, _ := regexp.MatchString("pipeline.pipeline.v1beta.PipelinePrivateService/.*Admin$", fullMethodName); match {
					return false
				}
			}
			// by default everything will be logged
			return true
		}),
	}

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpcmiddleware.ChainStreamServer(
			middleware.StreamAppendMetadataInterceptor,
			grpczap.StreamServerInterceptor(logger, opts...),
			grpcrecovery.StreamServerInterceptor(middleware.RecoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpcmiddleware.ChainUnaryServer(
			middleware.UnaryAppendMetadataInterceptor,
			grpczap.UnaryServerInterceptor(logger, opts...),
			grpcrecovery.UnaryServerInterceptor(middleware.RecoveryInterceptorOpt()),
		)),
	}

	// Create tls based credential.
	var creds credentials.TransportCredentials
	var tlsConfig *tls.Config
	var err error
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		tlsConfig = &tls.Config{
			ClientAuth: tls.RequireAndVerifyClientCert,
			MinVersion: tls.VersionTLS12,
		}
		creds, err = credentials.NewServerTLSFromFile(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key)
		if err != nil {
			logger.Fatal(fmt.Sprintf("failed to create credentials: %v", err))
		}
		grpcServerOpts = append(grpcServerOpts, grpc.Creds(creds))
	}

	grpcServerOpts = append(grpcServerOpts, grpc.MaxRecvMsgSize(constant.MaxPayloadSize))
	grpcServerOpts = append(grpcServerOpts, grpc.MaxSendMsgSize(constant.MaxPayloadSize))
	return grpcServerOpts, creds, tlsConfig
}
