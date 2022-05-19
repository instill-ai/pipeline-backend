package external

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/pipeline-backend/configs"
	"github.com/instill-ai/pipeline-backend/internal/logger"

	connectorPB "github.com/instill-ai/protogen-go/connector/v1alpha"
	mgmtPB "github.com/instill-ai/protogen-go/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

// InitUserServiceClient initialises a UserServiceClient instance
func InitUserServiceClient() mgmtPB.UserServiceClient {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if configs.Config.MgmtBackend.HTTPS.Cert != "" && configs.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(configs.Config.MgmtBackend.HTTPS.Cert, configs.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", configs.Config.MgmtBackend.Host, configs.Config.MgmtBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtPB.NewUserServiceClient(clientConn)
}

// InitConnectorServiceClient initialises a ConnectorServiceClient instance
func InitConnectorServiceClient() connectorPB.ConnectorServiceClient {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if configs.Config.ConnectorBackend.HTTPS.Cert != "" && configs.Config.ConnectorBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(configs.Config.ConnectorBackend.HTTPS.Cert, configs.Config.ConnectorBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", configs.Config.ConnectorBackend.Host, configs.Config.ConnectorBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return connectorPB.NewConnectorServiceClient(clientConn)
}

// InitModelServiceClient initialises a ModelServiceClient instance
func InitModelServiceClient() modelPB.ModelServiceClient {
	logger, _ := logger.GetZapLogger()

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if configs.Config.ModelBackend.HTTPS.Cert != "" && configs.Config.ModelBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(configs.Config.ModelBackend.HTTPS.Cert, configs.Config.ModelBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", configs.Config.ModelBackend.Host, configs.Config.ModelBackend.Port), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return modelPB.NewModelServiceClient(clientConn)
}
