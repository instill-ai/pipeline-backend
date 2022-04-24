package db

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	configs "github.com/instill-ai/pipeline-backend/configs"
)

var db *gorm.DB
var once sync.Once

func GetConnection() *gorm.DB {
	once.Do(func() {
		databaseConfig := configs.Config.Database
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=%s",
			databaseConfig.Host,
			databaseConfig.Username,
			databaseConfig.Password,
			databaseConfig.Name,
			databaseConfig.Port,
			databaseConfig.TimeZone,
		)
		var err error
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true, // disables implicit prepared statement usage
		}), &gorm.Config{
			QueryFields: true, // QueryFields mode will select by all fields’ name for current model
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		})

		if err != nil {
			panic("Could not open database connection")
		}

		sqlDB, _ := db.DB()

		// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
		sqlDB.SetMaxIdleConns(databaseConfig.Pool.IdleConnections)
		// SetMaxOpenConns sets the maximum number of open connections to the database.
		sqlDB.SetMaxOpenConns(databaseConfig.Pool.MaxConnections)
		// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
		sqlDB.SetConnMaxLifetime(time.Minute * databaseConfig.Pool.ConnLifeTime)
	})
	return db
}

func Close(db *gorm.DB) {
	// https://github.com/go-gorm/gorm/issues/3216
	//
	// This only works with a single master connection, but when dealing with replicas using DBResolver,
	// it does not close everything since DB.DB() only returns the master connection.
	if db != nil {
		sqlDB, _ := db.DB()

		sqlDB.Close()
	}
}
