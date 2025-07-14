package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/legacy"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	clientgrpcx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
)

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	databaseConfig := config.Config.Database
	if err := checkExist(databaseConfig, logger); err != nil {
		logger.Fatal("Checking database existence", zap.Error(err))
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?%s",
		databaseConfig.Username,
		databaseConfig.Password,
		databaseConfig.Host,
		databaseConfig.Port,
		databaseConfig.Name,
		"sslmode=disable",
	)

	codeMigrator, cleanup := initCodeMigrator(ctx, logger)
	defer cleanup()

	if err := runMigration(dsn, migration.TargetSchemaVersion, codeMigrator.Migrate, logger); err != nil {
		logger.Fatal("Running migration", zap.Error(err))
	}
}

func checkExist(databaseConfig config.DatabaseConfig, logger *zap.Logger) error {
	db, err := sql.Open(
		"postgres",
		fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%d sslmode=disable TimeZone=%s",
			databaseConfig.Host,
			databaseConfig.Username,
			databaseConfig.Password,
			databaseConfig.Port,
			databaseConfig.TimeZone,
		),
	)

	if err != nil {
		return fmt.Errorf("opening database connection: %w", err)
	}

	defer db.Close()

	// Open() may just validate its arguments without creating a connection to
	// the database. To verify that the data source name is valid, call Ping().
	if err = db.Ping(); err != nil {
		return fmt.Errorf("pinging database: %w", err)
	}

	var rows *sql.Rows
	rows, err = db.Query(fmt.Sprintf("SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s');", databaseConfig.Name))
	if err != nil {
		return fmt.Errorf("executing database name query: %w", err)
	}

	dbExist := false
	defer rows.Close()
	for rows.Next() {
		var databaseName string
		if err := rows.Scan(&databaseName); err != nil {
			return fmt.Errorf("scanning database name from row: %w", err)
		}

		if databaseConfig.Name == databaseName {
			dbExist = true
			logger.Info("Database exists", zap.String("name", databaseName))
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("scanning rows: %w", err)
	}

	if !dbExist {
		logger.Info("Create database", zap.String("name", databaseConfig.Name))
		if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", databaseConfig.Name)); err != nil {
			return fmt.Errorf("creating database: %w", err)
		}
	}

	return nil
}

func runMigration(
	dsn string,
	expectedVersion uint,
	execCode func(version uint) error,
	logger *zap.Logger,
) error {
	migrateFolder, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("accessing base path: %w", err)
	}

	m, err := migrate.New(fmt.Sprintf("file:///%s/pkg/db/migration", migrateFolder), dsn)
	if err != nil {
		return fmt.Errorf("creating migration: %w", err)
	}

	curVersion, dirty, err := m.Version()
	if err != nil && curVersion != 0 {
		return fmt.Errorf("getting current version: %w", err)
	}

	logger.Info("Running migration",
		zap.Uint("expectedVersion", expectedVersion),
		zap.Uint("currentVersion", curVersion),
		zap.Bool("dirty", dirty),
	)

	if dirty {
		return fmt.Errorf("database is dirty, please fix it")
	}

	step := curVersion
	for {
		if expectedVersion <= step {
			logger.Info("Migration completed", zap.Uint("expectedVersion", expectedVersion))
			break
		}

		switch step {
		case 5:
			if err := legacy.MigratePipelineRecipeUp000006(); err != nil {
				return fmt.Errorf("running legacy step 6: %w", err)
			}
		case 6:
			if err := legacy.MigratePipelineRecipeUp000007(); err != nil {
				return fmt.Errorf("running legacy step 7: %w", err)
			}
		case 11:
			if err := legacy.MigratePipelineRecipeUp000012(); err != nil {
				return fmt.Errorf("running legacy step 12: %w", err)
			}
		}

		logger.Info("Step up", zap.Uint("step", step+1))
		if err := m.Steps(1); err != nil {
			return fmt.Errorf("stepping up: %w", err)
		}

		if step, _, err = m.Version(); err != nil {
			return fmt.Errorf("getting new version: %w", err)
		}

		if err := execCode(step); err != nil {
			return fmt.Errorf("running associated code: %w", err)
		}
	}

	return nil
}

func initCodeMigrator(ctx context.Context, logger *zap.Logger) (cm *migration.CodeMigrator, cleanup func()) {
	cleanups := make([]func(), 0)

	rh := service.NewRetentionHandler()
	mgmtPrivateServiceClient, mgmtPrivateClose, err := clientgrpcx.NewClient[mgmtpb.MgmtPrivateServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.MgmtBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create mgmt private service client", zap.Error(err))
	}
	cleanups = append(cleanups, func() { _ = mgmtPrivateClose() })

	db := database.GetConnection().WithContext(ctx)
	cleanups = append(cleanups, func() { database.Close(db) })
	codeMigrator := &migration.CodeMigrator{
		Logger:                   logger,
		DB:                       db,
		RetentionHandler:         rh,
		MGMTPrivateServiceClient: mgmtPrivateServiceClient,
	}

	return codeMigrator, func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}
}
