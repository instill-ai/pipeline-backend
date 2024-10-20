package convert

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Basic contains the basic elements to execute a conversion migration.
type Basic struct {
	DB     *gorm.DB
	Logger *zap.Logger
}
