package logger

import (
	"sync"

	"go.uber.org/zap"

	"github.com/instill-ai/pipeline-backend/config"
)

var logger *zap.Logger
var once sync.Once

// GetZapLogger returns an instance of zap logger
func GetZapLogger() (*zap.Logger, error) {
	var err error
	once.Do(func() {
		if config.Config.Server.Debug {
			logger, err = zap.NewDevelopment()
		} else {
			logger, err = zap.NewProduction()
		}
	})

	return logger, err
}
