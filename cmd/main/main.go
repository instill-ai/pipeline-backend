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

	grpcmiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	openfga "github.com/openfga/api/proto/openfga/v1"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.temporal.io/sdk/client"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/acl"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	customotel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	"github.com/instill-ai/pipeline-backend/pkg/middleware"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/usage"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"
)

var propagator propagation.TextMapPropagator

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

	clientConn, err := grpc.Dial(fmt.Sprintf(":%v", config.Config.Server.PublicPort), clientDialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.MaxPayloadSize), grpc.MaxCallSendMsgSize(constant.MaxPayloadSize)))
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
				if match, _ := regexp.MatchString("vdp.pipeline.v1beta.PipelinePublicService/.*ness$", fullMethodName); match {
					return false
				}
				// stop logging successful private function calls
				if match, _ := regexp.MatchString("vdp.pipeline.v1beta.PipelinePrivateService/.*Admin$", fullMethodName); match {
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
	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

	repository := repository.NewRepository(db, redisClient)

	service := service.NewService(
		repository,
		redisClient,
		temporalClient,
		&aclClient,
		service.NewConverter(mgmtPrivateServiceClient, redisClient, &aclClient, repository),
	)

	privateGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(privateGrpcS)

	publicGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(publicGrpcS)

	pb.RegisterPipelinePrivateServiceServer(
		privateGrpcS,
		handler.NewPrivateHandler(ctx, service),
	)

	pb.RegisterPipelinePublicServiceServer(
		publicGrpcS,
		handler.NewPublicHandler(ctx, service),
	)

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
					usg = usage.NewUsage(ctx, repository, mgmtPrivateServiceClient, redisClient, usageServiceClient)
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

	if err := publicServeMux.HandlePath("POST", "/v1beta/{name=users/*/pipelines/*}/trigger", middleware.AppendCustomHeaderMiddleware(publicServeMux, pipelinePublicServiceClient, handler.HandleTrigger)); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1beta/{name=users/*/pipelines/*}/triggerAsync", middleware.AppendCustomHeaderMiddleware(publicServeMux, pipelinePublicServiceClient, handler.HandleTriggerAsync)); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1beta/{name=users/*/pipelines/*/releases/*}/trigger", middleware.AppendCustomHeaderMiddleware(publicServeMux, pipelinePublicServiceClient, handler.HandleTriggerRelease)); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1beta/{name=users/*/pipelines/*/releases/*}/triggerAsync", middleware.AppendCustomHeaderMiddleware(publicServeMux, pipelinePublicServiceClient, handler.HandleTriggerAsyncRelease)); err != nil {
		logger.Fatal(err.Error())
	}
	privateHTTPServer := &http.Server{
		Addr:      fmt.Sprintf(":%v", config.Config.Server.PrivatePort),
		Handler:   grpcHandlerFunc(privateGrpcS, privateServeMux),
		TLSConfig: tlsConfig,
	}

	publicHTTPServer := &http.Server{
		Addr:      fmt.Sprintf(":%v", config.Config.Server.PublicPort),
		Handler:   grpcHandlerFunc(publicGrpcS, publicServeMux),
		TLSConfig: tlsConfig,
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

	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quitSig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errSig:
		logger.Error(fmt.Sprintf("Fatal error: %v\n", err))
	case <-quitSig:
		if config.Config.Server.Usage.Enabled && usg != nil {
			usg.TriggerSingleReporter(ctx)
		}
		logger.Info("Shutting down server...")
		privateGrpcS.GracefulStop()
		publicGrpcS.GracefulStop()
	}
}
