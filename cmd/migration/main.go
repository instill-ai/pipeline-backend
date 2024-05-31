package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000013"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000015"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/legacy"
)

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
		panic(err)
	}

	defer db.Close()

	// Open() may just validate its arguments without creating a connection to the database.
	// To verify that the data source name is valid, call Ping().
	if err = db.Ping(); err != nil {
		panic(err)
	}

	var rows *sql.Rows
	rows, err = db.Query(fmt.Sprintf("SELECT datname FROM pg_catalog.pg_database WHERE lower(datname) = lower('%s');", databaseConfig.Name))

	if err != nil {
		panic(err)
	}

	dbExist := false
	defer rows.Close()
	for rows.Next() {
		var databaseName string
		if err := rows.Scan(&databaseName); err != nil {
			panic(err)
		}

		if databaseConfig.Name == databaseName {
			dbExist = true
			fmt.Printf("Database %s exist\n", databaseName)
		}
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	if !dbExist {
		fmt.Printf("Create database %s\n", databaseConfig.Name)
		if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s;", databaseConfig.Name)); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	migrateFolder, _ := os.Getwd()

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		panic(err)
	}

	databaseConfig := config.Config.Database

	if err := checkExist(databaseConfig); err != nil {
		panic(err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?%s",
		databaseConfig.Username,
		databaseConfig.Password,
		databaseConfig.Host,
		databaseConfig.Port,
		databaseConfig.Name,
		"sslmode=disable",
	)

	m, err := migrate.New(fmt.Sprintf("file:///%s/pkg/db/migration", migrateFolder), dsn)

	if err != nil {
		panic(err)
	}

	curVersion, dirty, err := m.Version()

	if err != nil && curVersion != 0 {
		panic(err)
	}

	ExpectedVersion := databaseConfig.Version

	fmt.Printf("Expected migration version is %d\n", ExpectedVersion)
	fmt.Printf("The current schema version is %d, and dirty flag is %t\n", curVersion, dirty)
	if dirty {
		panic("The database has dirty flag, please fix it")
	}

	step := curVersion
	for {
		if ExpectedVersion <= step {
			fmt.Printf("Migration to version %d complete\n", ExpectedVersion)
			break
		}

		switch step {
		case 5:
			if err := legacy.MigratePipelineRecipeUp000006(); err != nil {
				panic(err)
			}
		case 6:
			if err := legacy.MigratePipelineRecipeUp000007(); err != nil {
				panic(err)
			}
		case 11:
			if err := legacy.MigratePipelineRecipeUp000012(); err != nil {
				panic(err)
			}
		}

		fmt.Printf("Step up to version %d\n", step+1)
		if err := m.Steps(1); err != nil {
			panic(err)
		}

		if step, _, err = m.Version(); err != nil {
			panic(err)
		}
		switch step {
		case 13:
			if err := convert000013.Migrate(); err != nil {
				panic(err)
			}
		case 15:
			if err := convert000015.Migrate(); err != nil {
				panic(err)
			}
		}

	}
}
