package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	openfga "github.com/openfga/api/proto/openfga/v1"
	temporalclient "go.temporal.io/sdk/client"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/middleware"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/usage"
	"github.com/instill-ai/x/temporal"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	artifactpb "github.com/instill-ai/protogen-go/artifact/artifact/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	usagepb "github.com/instill-ai/protogen-go/core/usage/v1beta"
	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	clientx "github.com/instill-ai/x/client"
	clientgrpcx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
	miniox "github.com/instill-ai/x/minio"
	otelx "github.com/instill-ai/x/otel"
	servergrpcx "github.com/instill-ai/x/server/grpc"
	gatewayx "github.com/instill-ai/x/server/grpc/gateway"
)

const gracefulShutdownWaitPeriod = 15 * time.Second
const gracefulShutdownTimeout = 60 * time.Minute

var (
	// These variables might be overridden at buildtime.
	serviceName    = "pipeline-backend"
	serviceVersion = "dev"
)

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup all OpenTelemetry components
	cleanup := otelx.SetupWithCleanup(ctx,
		otelx.WithServiceName(serviceName),
		otelx.WithServiceVersion(serviceVersion),
		otelx.WithHost(config.Config.OTELCollector.Host),
		otelx.WithPort(config.Config.OTELCollector.Port),
		otelx.WithCollectorEnable(config.Config.OTELCollector.Enable),
	)
	defer cleanup()

	logx.Debug = config.Config.Server.Debug
	logger, _ := logx.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	// Set gRPC logging based on debug mode
	if config.Config.Server.Debug {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 0) // All logs
	} else {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 3) // verbosity 3 will avoid [transport] from emitting
	}

	// Get gRPC server options and credentials
	grpcServerOpts, err := servergrpcx.NewServerOptionsAndCreds(
		servergrpcx.WithServiceName(serviceName),
		servergrpcx.WithServiceVersion(serviceVersion),
		servergrpcx.WithServiceConfig(clientx.HTTPSConfig{
			Cert: config.Config.Server.HTTPS.Cert,
			Key:  config.Config.Server.HTTPS.Key,
		}),
		servergrpcx.WithSetOTELServerHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create gRPC server options and credentials", zap.Error(err))
	}

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
		runtime.WithForwardResponseOption(gatewayx.HTTPResponseModifier),
		runtime.WithErrorHandler(gatewayx.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(gatewayx.CustomHeaderMatcher),
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
		runtime.WithForwardResponseOption(gatewayx.HTTPResponseModifier),
		runtime.WithErrorHandler(gatewayx.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(gatewayx.CustomHeaderMatcher),
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
		usageServiceClient, usageServiceClientClose, err := clientgrpcx.NewClient[usagepb.UsageServiceClient](
			clientgrpcx.WithServiceConfig(clientx.ServiceConfig{
				Host:       config.Config.Server.Usage.Host,
				PublicPort: config.Config.Server.Usage.Port,
			}),
			clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
		)
		if err != nil {
			logger.Error("failed to create usage service client", zap.Error(err))
		}
		defer func() {
			if err := usageServiceClientClose(); err != nil {
				logger.Error("failed to close usage service client", zap.Error(err))
			}
		}()
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

	dialOpts, err := clientgrpcx.NewClientOptionsAndCreds(
		clientgrpcx.WithServiceConfig(clientx.ServiceConfig{
			HTTPS: clientx.HTTPSConfig{
				Cert: config.Config.Server.HTTPS.Cert,
				Key:  config.Config.Server.HTTPS.Key,
			},
		}),
		clientgrpcx.WithSetOTELClientHandler(false),
	)
	if err != nil {
		logger.Fatal("failed to create client options and credentials", zap.Error(err))
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
		ReadHeaderTimeout: 10 * time.Millisecond,
	}

	publicHTTPServer := &http.Server{
		Addr:              fmt.Sprintf(":%v", config.Config.Server.PublicPort),
		Handler:           grpcHandlerFunc(publicGrpcS, publicServeMux),
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
	miniox.Client,
	*miniox.FileGetter,
	acl.ACLClient,
	temporalclient.Client,
	func(),
) {
	closeFuncs := map[string]func() error{}

	// Initialize external service clients
	pipelinePublicServiceClient, pipelinePublicClose, err := clientgrpcx.NewClient[pipelinepb.PipelinePublicServiceClient](
		clientgrpcx.WithServiceConfig(clientx.ServiceConfig{
			Host:       config.Config.Server.InstanceID,
			PublicPort: config.Config.Server.PublicPort,
			HTTPS: clientx.HTTPSConfig{
				Cert: config.Config.Server.HTTPS.Cert,
				Key:  config.Config.Server.HTTPS.Key,
			},
		}),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create pipeline public service client", zap.Error(err))
	}
	closeFuncs["pipelinePublic"] = pipelinePublicClose

	mgmtPublicServiceClient, mgmtPublicClose, err := clientgrpcx.NewClient[mgmtpb.MgmtPublicServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.MgmtBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create mgmt public service client", zap.Error(err))
	}
	closeFuncs["mgmtPublic"] = mgmtPublicClose

	mgmtPrivateServiceClient, mgmtPrivateClose, err := clientgrpcx.NewClient[mgmtpb.MgmtPrivateServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.MgmtBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create mgmt private service client", zap.Error(err))
	}
	closeFuncs["mgmtPrivate"] = mgmtPrivateClose

	artifactPublicServiceClient, artifactPublicClose, err := clientgrpcx.NewClient[artifactpb.ArtifactPublicServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.ArtifactBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create artifact public service client", zap.Error(err))
	}
	closeFuncs["artifactPublic"] = artifactPublicClose

	artifactPrivateServiceClient, artifactPrivateClose, err := clientgrpcx.NewClient[artifactpb.ArtifactPrivateServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.ArtifactBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
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
	temporalClientOptions, err := temporal.ClientOptions(config.Config.Temporal, logger)
	if err != nil {
		logger.Fatal("Unable to build Temporal client options", zap.Error(err))
	}

	// Only add interceptor if tracing is enabled
	if config.Config.OTELCollector.Enable {
		temporalTracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{
			Tracer:            otel.Tracer(serviceName),
			TextMapPropagator: otel.GetTextMapPropagator(),
		})
		if err != nil {
			logger.Fatal("Unable to create temporal tracing interceptor", zap.Error(err))
		}
		temporalClientOptions.Interceptors = []interceptor.ClientInterceptor{temporalTracingInterceptor}
	}

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
	minIOParams := miniox.ClientParams{
		Config: config.Config.Minio,
		Logger: logger,
		AppInfo: miniox.AppInfo{
			Name:    serviceName,
			Version: serviceVersion,
		},
	}
	minIOFileGetter, err := miniox.NewFileGetter(minIOParams)
	if err != nil {
		logger.Fatal("Failed to create MinIO file getter", zap.Error(err))
	}

	retentionHandler := service.NewRetentionHandler()
	minIOParams.ExpiryRules = retentionHandler.ListExpiryRules()
	minIOClient, err := miniox.NewMinIOClientAndInitBucket(ctx, minIOParams)
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
