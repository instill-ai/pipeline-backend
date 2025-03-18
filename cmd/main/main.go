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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofrs/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/middleware"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/usage"
	"github.com/instill-ai/x/minio"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	pipelineworker "github.com/instill-ai/pipeline-backend/pkg/worker"
	pb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
)

const gracefulShutdownWaitPeriod = 15 * time.Second
const gracefulShutdownTimeout = 60 * time.Minute

var propagator propagation.TextMapPropagator

// These variables might be overridden at buildtime.
var version = "dev"
var serviceName = "pipeline-backend"

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

// InitPipelinePublicServiceClient initialises a PipelineServiceClient instance
func InitPipelinePublicServiceClient(ctx context.Context) (pb.PipelinePublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.NewClient(fmt.Sprintf(":%v", config.Config.Server.PublicPort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return pb.NewPipelinePublicServiceClient(clientConn), clientConn
}

func main() {
	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	// setup tracing and metrics
	ctx, cancel := context.WithCancel(context.Background())

	if tp, err := customotel.SetupTracing(ctx, "pipeline-backend"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = tp.Shutdown(ctx)
		}()
	}

	ctx, span := otel.Tracer("main-tracer").Start(ctx,
		"main",
	)
	defer cancel()

	logger, _ := logger.GetZapLogger(ctx)
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	// verbosity 3 will avoid [transport] from emitting
	grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 3)

	db := database.GetSharedConnection()
	defer database.Close(db)

	var temporalClientOptions client.Options
	var err error
	if config.Config.Temporal.Ca != "" && config.Config.Temporal.Cert != "" && config.Config.Temporal.Key != "" {
		if temporalClientOptions, err = temporal.GetTLSClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger),
			config.Config.Temporal.Ca,
			config.Config.Temporal.Cert,
			config.Config.Temporal.Key,
			config.Config.Temporal.ServerName,
			true,
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	} else {
		if temporalClientOptions, err = temporal.GetClientOption(
			config.Config.Temporal.HostPort,
			config.Config.Temporal.Namespace,
			zapadapter.NewZapAdapter(logger)); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to get Temporal client options: %s", err))
		}
	}

	temporalClient, err := client.Dial(temporalClientOptions)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to create client: %s", err))
	}
	defer temporalClient.Close()

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

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	fgaClient, fgaClientConn := acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Host, config.Config.OpenFGA.Port)
	if fgaClientConn != nil {
		defer fgaClientConn.Close()
	}
	var fgaReplicaClient openfga.OpenFGAServiceClient
	var fgaReplicaClientConn *grpc.ClientConn
	if config.Config.OpenFGA.Replica.Host != "" {

		fgaReplicaClient, fgaReplicaClientConn = acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Replica.Host, config.Config.OpenFGA.Replica.Port)
		if fgaReplicaClientConn != nil {
			defer fgaReplicaClientConn.Close()
		}
	}

	aclClient := acl.NewACLClient(fgaClient, fgaReplicaClient, redisClient)

	// Create tls based credential.
	var creds credentials.TransportCredentials
	var tlsConfig *tls.Config
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

	pipelinePublicServiceClient, pipelinePublicServiceClientConn := InitPipelinePublicServiceClient(ctx)
	if pipelinePublicServiceClientConn != nil {
		defer pipelinePublicServiceClientConn.Close()
	}
	mgmtPublicServiceClient, mgmtPublicServiceClientConn := external.InitMgmtPublicServiceClient(ctx)
	if mgmtPublicServiceClientConn != nil {
		defer mgmtPublicServiceClientConn.Close()
	}
	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

	// Initialize MinIO client
	minIOParams := minio.ClientParams{
		Config: config.Config.Minio,
		Logger: logger,
		AppInfo: minio.AppInfo{
			Name:    serviceName,
			Version: version,
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

	artifactPublicServiceClient, artifactPublicServiceClientConn := external.InitArtifactPublicServiceClient(ctx)
	if artifactPublicServiceClientConn != nil {
		defer artifactPublicServiceClientConn.Close()
	}

	artifactPrivateServiceClient, artifactPrivateServiceClientConn := external.InitArtifactPrivateServiceClient(ctx)
	if artifactPrivateServiceClientConn != nil {
		defer artifactPrivateServiceClientConn.Close()
	}

	binaryFetcher := external.NewArtifactBinaryFetcher(artifactPrivateServiceClient, minIOFileGetter)

	compStore := componentstore.Init(componentstore.InitParams{
		Logger:         logger,
		Secrets:        config.Config.Component.Secrets,
		BinaryFetcher:  binaryFetcher,
		TemporalClient: temporalClient,
	})
	workerUID, _ := uuid.NewV4()

	pubsub := pubsub.NewRedisPubSub(redisClient)
	ms := memory.NewStore(pubsub)

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
		workerUID,
		retentionHandler,
		binaryFetcher,
		artifactPublicServiceClient,
		artifactPrivateServiceClient,
	)

	privateGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(privateGrpcS)

	publicGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(publicGrpcS)

	privateHandler := handler.NewPrivateHandler(service, logger)
	pb.RegisterPipelinePrivateServiceServer(privateGrpcS, privateHandler)

	publicHandler := handler.NewPublicHandler(service, logger)
	pb.RegisterPipelinePublicServiceServer(publicGrpcS, publicHandler)

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
		usageServiceClient, usageServiceClientConn := external.InitUsageServiceClient(ctx)
		if usageServiceClientConn != nil {
			defer usageServiceClientConn.Close()
			logger.Info("try to start usage reporter")
			go func() {
				for {
					usg = usage.NewUsage(ctx, repo, mgmtPrivateServiceClient, redisClient, usageServiceClient)
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

	if err := pb.RegisterPipelinePrivateServiceHandlerFromEndpoint(ctx, privateServeMux, fmt.Sprintf(":%v", config.Config.Server.PrivatePort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	if err := pb.RegisterPipelinePublicServiceHandlerFromEndpoint(ctx, publicServeMux, fmt.Sprintf(":%v", config.Config.Server.PublicPort), dialOpts); err != nil {
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

	span.End()
	logger.Info("gRPC server is running.")

	// for only local temporal cluster
	if config.Config.Temporal.Ca == "" && config.Config.Temporal.Cert == "" && config.Config.Temporal.Key == "" {
		initTemporalNamespace(ctx, temporalClient)
	}

	timeseries := repository.MustNewInfluxDB(ctx)
	defer timeseries.Close()

	cw := pipelineworker.NewWorker(
		pipelineworker.WorkerConfig{
			Repository:                   repo,
			RedisClient:                  redisClient,
			InfluxDBWriteClient:          timeseries.WriteAPI(),
			Component:                    compStore,
			MinioClient:                  minIOClient,
			MemoryStore:                  ms,
			WorkerUID:                    workerUID,
			ArtifactPublicServiceClient:  artifactPublicServiceClient,
			ArtifactPrivateServiceClient: artifactPrivateServiceClient,
			BinaryFetcher:                binaryFetcher,
			PipelinePublicServiceClient:  pipelinePublicServiceClient,
		},
	)

	w := worker.New(temporalClient, pipelineworker.TaskQueue, worker.Options{
		WorkflowPanicPolicy:                    worker.BlockWorkflow,
		WorkerStopTimeout:                      gracefulShutdownTimeout,
		MaxConcurrentWorkflowTaskExecutionSize: 100,
	})
	lw := worker.New(temporalClient, workerUID.String(), worker.Options{
		WorkflowPanicPolicy:                worker.BlockWorkflow,
		WorkerStopTimeout:                  gracefulShutdownTimeout,
		MaxConcurrentActivityExecutionSize: 100,
	})
	mw := worker.New(temporalClient, fmt.Sprintf("%s-minio", workerUID.String()), worker.Options{
		WorkflowPanicPolicy:                worker.BlockWorkflow,
		WorkerStopTimeout:                  gracefulShutdownTimeout,
		MaxConcurrentActivityExecutionSize: 50,
	})

	w.RegisterWorkflow(cw.TriggerPipelineWorkflow)
	w.RegisterWorkflow(cw.SchedulePipelineWorkflow)

	lw.RegisterActivity(cw.ComponentActivity)
	lw.RegisterActivity(cw.OutputActivity)
	lw.RegisterActivity(cw.PreIteratorActivity)
	lw.RegisterActivity(cw.PostIteratorActivity)
	lw.RegisterActivity(cw.InitComponentsActivity)
	lw.RegisterActivity(cw.SendStartedEventActivity)
	lw.RegisterActivity(cw.PostTriggerActivity)
	lw.RegisterActivity(cw.ClosePipelineActivity)
	lw.RegisterActivity(cw.IncreasePipelineTriggerCountActivity)
	lw.RegisterActivity(cw.UpdatePipelineRunActivity)
	lw.RegisterActivity(cw.UpsertComponentRunActivity)

	mw.RegisterActivity(cw.UploadOutputsToMinIOActivity)
	mw.RegisterActivity(cw.UploadRecipeToMinIOActivity)
	mw.RegisterActivity(cw.UploadComponentInputsActivity)
	mw.RegisterActivity(cw.UploadComponentOutputsActivity)

	err = w.Start()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start worker: %s", err))
	}
	err = lw.Start()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start local worker: %s", err))
	}
	err = mw.Start()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to start minio worker: %s", err))
	}
	logger.Info("worker is running.")

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

		logger.Info("Shutting down worker...")
		w.Stop()
		lw.Stop()
		mw.Stop()

		logger.Info("Shutting down HTTP server...")
		_ = privateHTTPServer.Shutdown(shutdownCtx)
		_ = publicHTTPServer.Shutdown(shutdownCtx)
		logger.Info("Shutting down gRPC server...")
		privateGrpcS.GracefulStop()
		publicGrpcS.GracefulStop()

	}
}

func initTemporalNamespace(ctx context.Context, client client.Client) {
	logger, _ := logger.GetZapLogger(ctx)

	resp, err := client.WorkflowService().ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{})
	if err != nil {
		logger.Fatal(fmt.Sprintf("Unable to list namespaces: %s", err))
	}

	found := false
	for _, n := range resp.GetNamespaces() {
		if n.NamespaceInfo.Name == config.Config.Temporal.Namespace {
			found = true
		}
	}

	if !found {
		if _, err := client.WorkflowService().RegisterNamespace(ctx,
			&workflowservice.RegisterNamespaceRequest{
				Namespace: config.Config.Temporal.Namespace,
				WorkflowExecutionRetentionPeriod: func() *durationpb.Duration {
					// Check if the string ends with "d" for day.
					s := config.Config.Temporal.Retention
					if strings.HasSuffix(s, "d") {
						// Parse the number of days.
						days, err := strconv.Atoi(s[:len(s)-1])
						if err != nil {
							logger.Fatal(fmt.Sprintf("Unable to parse retention period in day: %s", err))
						}
						// Convert days to hours and then to a duration.
						t := time.Hour * 24 * time.Duration(days)
						return durationpb.New(t)
					}
					logger.Fatal(fmt.Sprintf("Unable to parse retention period in day: %s", err))
					return nil
				}(),
			},
		); err != nil {
			logger.Fatal(fmt.Sprintf("Unable to register namespace: %s", err))
		}
	}
}
