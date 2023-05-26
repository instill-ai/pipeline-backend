package external

import (
	"context"
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	controllerPB "github.com/instill-ai/protogen-go/vdp/controller/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// InitConnectorPublicServiceClient initialises a ConnectorPublicServiceClient instance
func InitConnectorPublicServiceClient(ctx context.Context) (connectorPB.ConnectorPublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.ConnectorBackend.HTTPS.Cert != "" && config.Config.ConnectorBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.ConnectorBackend.HTTPS.Cert, config.Config.ConnectorBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.ConnectorBackend.Host, config.Config.ConnectorBackend.PublicPort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return connectorPB.NewConnectorPublicServiceClient(clientConn), clientConn
}

// InitConnectorPrivateServiceClient initialises a ConnectorPrivateServiceClient instance
func InitConnectorPrivateServiceClient(ctx context.Context) (connectorPB.ConnectorPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.ConnectorBackend.HTTPS.Cert != "" && config.Config.ConnectorBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.ConnectorBackend.HTTPS.Cert, config.Config.ConnectorBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.ConnectorBackend.Host, config.Config.ConnectorBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return connectorPB.NewConnectorPrivateServiceClient(clientConn), clientConn
}

// InitModelPublicServiceClient initialises a ModelPublicServiceClient instance
func InitModelPublicServiceClient(ctx context.Context) (modelPB.ModelPublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.ModelBackend.HTTPS.Cert != "" && config.Config.ModelBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.ModelBackend.HTTPS.Cert, config.Config.ModelBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return modelPB.NewModelPublicServiceClient(clientConn), clientConn
}

// InitModelPrivateServiceClient initialises a ModelPrivateServiceClient instance
func InitModelPrivateServiceClient(ctx context.Context) (modelPB.ModelPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.ModelBackend.HTTPS.Cert != "" && config.Config.ModelBackend.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.ModelBackend.HTTPS.Cert, config.Config.ModelBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.ModelBackend.Host, config.Config.ModelBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return modelPB.NewModelPrivateServiceClient(clientConn), clientConn
}

// InitMgmtPrivateServiceClient initialises a MgmtPrivateServiceClient instance
func InitMgmtPrivateServiceClient(ctx context.Context) (mgmtPB.MgmtPrivateServiceClient, *grpc.ClientConn) {
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

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return mgmtPB.NewMgmtPrivateServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initialises a UsageServiceClient instance (no mTLS)
func InitUsageServiceClient(ctx context.Context) (usagePB.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var err error
	if config.Config.UsageServer.TLSEnabled {
		tlsConfig := &tls.Config{}
		clientDialOpts = grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.UsageServer.Host, config.Config.UsageServer.Port), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return usagePB.NewUsageServiceClient(clientConn), clientConn
}

// InitControllerPrivateServiceClient initialises a ControllerPrivateServiceClient instance
func InitControllerPrivateServiceClient(ctx context.Context) (controllerPB.ControllerPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.Controller.HTTPS.Cert != "" && config.Config.Controller.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.Controller.HTTPS.Cert, config.Config.Controller.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.Controller.Host, config.Config.Controller.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return controllerPB.NewControllerPrivateServiceClient(clientConn), clientConn
}
