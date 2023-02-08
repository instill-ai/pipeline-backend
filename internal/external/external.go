package external

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/logger"

	connectorPB "github.com/instill-ai/protogen-go/vdp/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
	usagePB "github.com/instill-ai/protogen-go/vdp/usage/v1alpha"
)

// InitConnectorServiceClient initialises a ConnectorServiceClient instance
func InitConnectorServiceClient() (connectorPB.ConnectorServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

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

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.ConnectorBackend.Host, config.Config.ConnectorBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return connectorPB.NewConnectorServiceClient(clientConn), clientConn
}

// InitModelServiceClient initialises a ModelServiceClient instance
func InitModelServiceClient() (modelPB.ModelServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

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

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.ModelBackend.Host, config.Config.ModelBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return modelPB.NewModelServiceClient(clientConn), clientConn
}

// InitUserServiceClient initialises a UserServiceClient instance
func InitMgmtAdminServiceClient() (mgmtPB.MgmtAdminServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

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

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.AdminHost, config.Config.MgmtBackend.AdminPort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtPB.NewMgmtAdminServiceClient(clientConn), clientConn
}

// InitUsageServiceClient initialises a UsageServiceClient instance (no mTLS)
func InitUsageServiceClient() (usagePB.UsageServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger()

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
		logger.Fatal(err.Error())
	}

	return usagePB.NewUsageServiceClient(clientConn), clientConn

}
