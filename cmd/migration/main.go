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

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/legacy"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

var log *zap.Logger

func checkExist(databaseConfig config.DatabaseConfig) error {
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
			log.With(zap.String("name", databaseName)).Info("Database exists")
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("scanning rows: %w", err)
	}

	if !dbExist {
		log.With(zap.String("name", databaseConfig.Name)).Info("Create database")
		if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", databaseConfig.Name)); err != nil {
			return fmt.Errorf("creating database: %w", err)
		}
	}

	return nil
}

func main() {
	ctx := context.Background()
	log, _ = logger.GetZapLogger(ctx)

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.With(zap.Error(err)).Fatal("Loading configuration")
	}

	databaseConfig := config.Config.Database
	if err := checkExist(databaseConfig); err != nil {
		log.With(zap.Error(err)).Fatal("Checking database existence")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?%s",
		databaseConfig.Username,
		databaseConfig.Password,
		databaseConfig.Host,
		databaseConfig.Port,
		databaseConfig.Name,
		"sslmode=disable",
	)

	codeMigrator, cleanup := initCodeMigrator(ctx)
	defer cleanup()

	if err := runMigration(dsn, database.TargetSchemaVersion, codeMigrator.Migrate); err != nil {
		log.With(zap.Error(err)).Fatal("Running migration")
	}
}

func runMigration(
	dsn string,
	expectedVersion uint,
	execCode func(version uint) error,
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

	log.With(
		zap.Uint("expectedVersion", expectedVersion),
		zap.Uint("currentVersion", curVersion),
		zap.Bool("dirty", dirty),
	).Info("Running migration")

	if dirty {
		return fmt.Errorf("database is dirty, please fix it")
	}

	step := curVersion
	for {
		if expectedVersion <= step {
			log.With(zap.Uint("expectedVersion", expectedVersion)).Info("Migration completed")
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

		log.With(zap.Uint("step", step+1)).Info("Step up")
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

func initCodeMigrator(ctx context.Context) (cm *migration.CodeMigrator, cleanup func()) {
	l, _ := logger.GetZapLogger(ctx)
	cleanups := make([]func(), 0)

	rh := service.NewRetentionHandler()
	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		cleanups = append(cleanups, func() { _ = mgmtPrivateServiceClientConn.Close() })
	}

	db := database.GetConnection().WithContext(ctx)
	cleanups = append(cleanups, func() { database.Close(db) })
	codeMigrator := &migration.CodeMigrator{
		Logger:                   l,
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
