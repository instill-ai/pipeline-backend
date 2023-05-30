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

	"github.com/go-redis/redis/v9"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/client"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/handler"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	"github.com/instill-ai/pipeline-backend/pkg/middleware"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/usage"
	"github.com/instill-ai/x/temporal"
	"github.com/instill-ai/x/zapadapter"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func grpcHandlerFunc(grpcServer *grpc.Server, gwHandler http.Handler, CORSOrigins []string) http.Handler {
	return h2c.NewHandler(
		cors.New(cors.Options{
			AllowedOrigins:   CORSOrigins,
			AllowCredentials: true,
			Debug:            false,
			AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "HEAD"},
		}).Handler(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if len(r.Header["X-B3-Traceid"]) > 0 {
					traceID, _ := trace.TraceIDFromHex(r.Header["X-B3-Traceid"][0])
					spanID, _ := trace.SpanIDFromHex(r.Header["X-B3-Spanid"][0])
					var traceFlags trace.TraceFlags
					if r.Header["X-B3-Sampled"][0] == "1" {
						traceFlags = trace.FlagsSampled
					}

					spanContext := trace.NewSpanContext(trace.SpanContextConfig{
						TraceID:    traceID,
						SpanID:     spanID,
						TraceFlags: traceFlags,
					})

					ctx := trace.ContextWithSpanContext(r.Context(), spanContext)
					r = r.WithContext(ctx)
				}
				if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(w, r)
				} else {
					gwHandler.ServeHTTP(w, r)
				}
			})),
		&http2.Server{},
	)
}

func main() {

	if err := config.Init(); err != nil {
		log.Fatal(err.Error())
	}

	// setup tracing and metrics
	ctx, cancel := context.WithCancel(context.Background())

	if tp, err := custom_otel.SetupTracing(ctx, "pipeline-backend"); err != nil {
		panic(err)
	} else {
		defer tp.Shutdown(ctx)
	}

	if mp, err := custom_otel.SetupMetrics(ctx, "pipeline-backend"); err != nil {
		panic(err)
	} else {
		defer mp.Shutdown(ctx)
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

	db := database.GetConnection()
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
	opts := []grpc_zap.Option{
		grpc_zap.WithDecider(func(fullMethodName string, err error) bool {
			// will not log gRPC calls if it was a call to liveness or readiness and no error was raised
			if err == nil {
				if match, _ := regexp.MatchString("vdp.pipeline.v1alpha.PipelinePublicService/.*ness$", fullMethodName); match {
					return false
				}
			}
			// by default everything will be logged
			return true
		}),
	}

	grpcServerOpts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			middleware.StreamAppendMetadataInterceptor,
			grpc_zap.StreamServerInterceptor(logger, opts...),
			grpc_recovery.StreamServerInterceptor(middleware.RecoveryInterceptorOpt()),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			middleware.UnaryAppendMetadataInterceptor,
			grpc_zap.UnaryServerInterceptor(logger, opts...),
			grpc_recovery.UnaryServerInterceptor(middleware.RecoveryInterceptorOpt()),
		)),
	}

	grpc_zap.ReplaceGrpcLoggerV2(logger)

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

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

	connectorPublicServiceClient, connectorPublicServiceClientConn := external.InitConnectorPublicServiceClient(ctx)
	if connectorPublicServiceClientConn != nil {
		defer connectorPublicServiceClientConn.Close()
	}

	connectorPrivateServiceClient, connectorPrivateServiceClientConn := external.InitConnectorPrivateServiceClient(ctx)
	if connectorPrivateServiceClientConn != nil {
		defer connectorPrivateServiceClientConn.Close()
	}

	modelPublicServiceClient, modelPublicServiceClientConn := external.InitModelPublicServiceClient(ctx)
	if modelPublicServiceClientConn != nil {
		defer modelPublicServiceClientConn.Close()
	}

	modelPrivateServiceClient, modelPrivateServiceClientConn := external.InitModelPrivateServiceClient(ctx)
	if modelPrivateServiceClientConn != nil {
		defer modelPrivateServiceClientConn.Close()
	}

	controllerServiceClient, controllerServiceClientConn := external.InitControllerPrivateServiceClient(ctx)
	if controllerServiceClientConn != nil {
		defer controllerServiceClientConn.Close()
	}

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	repository := repository.NewRepository(db)

	service := service.NewService(
		repository,
		mgmtPrivateServiceClient,
		connectorPublicServiceClient,
		connectorPrivateServiceClient,
		modelPublicServiceClient,
		modelPrivateServiceClient,
		controllerServiceClient,
		redisClient,
		temporalClient,
	)

	privateGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(privateGrpcS)

	publicGrpcS := grpc.NewServer(grpcServerOpts...)
	reflection.Register(publicGrpcS)

	pipelinePB.RegisterPipelinePrivateServiceServer(
		privateGrpcS,
		handler.NewPrivateHandler(ctx, service),
	)

	pipelinePB.RegisterPipelinePublicServiceServer(
		publicGrpcS,
		handler.NewPublicHandler(ctx, service),
	)

	privateServeMux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(middleware.HTTPResponseModifier),
		runtime.WithErrorHandler(middleware.ErrorHandler),
		runtime.WithIncomingHeaderMatcher(middleware.CustomMatcher),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames:   true,
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
				UseProtoNames:   true,
				EmitUnpopulated: true,
				UseEnumNumbers:  false,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: true,
			},
		}),
	)

	// Register custom route for POST multipart form data
	if err := publicServeMux.HandlePath("POST", "/v1alpha/pipelines/{id}/triggerSyncMultipart", middleware.AppendCustomHeaderMiddleware(service, handler.HandleTriggerSyncPipelineBinaryFileUpload)); err != nil {
		logger.Fatal(err.Error())
	}
	if err := publicServeMux.HandlePath("POST", "/v1alpha/pipelines/{id}/triggerAsyncMultipart", middleware.AppendCustomHeaderMiddleware(service, handler.HandleTriggerAsyncPipelineBinaryFileUpload)); err != nil {
		logger.Fatal(err.Error())
	}

	// Start usage reporter
	var usg usage.Usage
	if !config.Config.Server.DisableUsage {
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
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(creds)}
	} else {
		dialOpts = []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}

	if err := pipelinePB.RegisterPipelinePrivateServiceHandlerFromEndpoint(ctx, privateServeMux, fmt.Sprintf(":%v", config.Config.Server.PrivatePort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	if err := pipelinePB.RegisterPipelinePublicServiceHandlerFromEndpoint(ctx, publicServeMux, fmt.Sprintf(":%v", config.Config.Server.PublicPort), dialOpts); err != nil {
		logger.Fatal(err.Error())
	}

	privateHTTPServer := &http.Server{
		Addr:      fmt.Sprintf(":%v", config.Config.Server.PrivatePort),
		Handler:   grpcHandlerFunc(privateGrpcS, privateServeMux, config.Config.Server.CORSOrigins),
		TLSConfig: tlsConfig,
	}

	publicHTTPServer := &http.Server{
		Addr:      fmt.Sprintf(":%v", config.Config.Server.PublicPort),
		Handler:   grpcHandlerFunc(publicGrpcS, publicServeMux, config.Config.Server.CORSOrigins),
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
		if !config.Config.Server.DisableUsage && usg != nil {
			usg.TriggerSingleReporter(ctx)
		}
		logger.Info("Shutting down server...")
		privateGrpcS.GracefulStop()
		publicGrpcS.GracefulStop()
	}
}
