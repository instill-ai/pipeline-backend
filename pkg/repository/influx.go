package repository

import (
	"context"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/log"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"

	client "github.com/influxdata/influxdb-client-go/v2"

	"github.com/instill-ai/pipeline-backend/config"

	logx "github.com/instill-ai/x/log"
)

// InfluxDB reads and writes time series data from InfluxDB.
type InfluxDB struct {
	client client.Client
	api    api.WriteAPI
}

// MustNewInfluxDB returns an initialized InfluxDB repository.
func MustNewInfluxDB(ctx context.Context) *InfluxDB {
	logger, _ := logx.GetZapLogger(ctx)

	opts := client.DefaultOptions()
	if config.Config.Server.Debug {
		opts = opts.SetLogLevel(log.DebugLevel)
	}

	flush := uint(config.Config.InfluxDB.FlushInterval.Milliseconds())
	opts = opts.SetFlushInterval(flush)

	var creds credentials.TransportCredentials
	var err error

	if config.Config.InfluxDB.HTTPS.Cert != "" && config.Config.InfluxDB.HTTPS.Key != "" {
		// TODO support TLS
		creds, err = credentials.NewServerTLSFromFile(config.Config.InfluxDB.HTTPS.Cert, config.Config.InfluxDB.HTTPS.Key)
		if err != nil {
			logger.With(zap.Error(err)).Fatal("Couldn't initialize InfluxDB client")
		}

		logger = logger.With(zap.String("influxServer", creds.Info().ServerName))
	}

	db := new(InfluxDB)
	db.client = client.NewClientWithOptions(
		config.Config.InfluxDB.URL,
		config.Config.InfluxDB.Token,
		opts,
	)

	bucket, org := config.Config.InfluxDB.Org, config.Config.InfluxDB.Bucket
	db.api = db.client.WriteAPI(bucket, org)
	logger = logger.With(zap.String("bucket", bucket)).
		With(zap.String("org", org))

	errChan := db.api.Errors()
	go func() {
		for err := range errChan {
			logger.With(zap.Error(err)).Error("Failed to write to InfluxDB bucket")
		}
	}()

	logger.Info("InfluxDB client initialized")
	if _, err := db.client.Ping(ctx); err != nil {
		logger.With(zap.Error(err)).Warn("Failed to ping InfluxDB")
	}

	return db
}

// Close  cleans up the InfluxDB connections.
func (i *InfluxDB) Close() {
	i.client.Close()
}

// WriteAPI return the InfluxDB client's Write API.
// TODO this is a shortcut to avoid refactoring client packages (e.g. worker)
// but we should use a TimeSeriesRepository interface in them.
func (i *InfluxDB) WriteAPI() api.WriteAPI {
	return i.api
}
