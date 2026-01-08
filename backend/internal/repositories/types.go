package repositories

import (
	"syncflow-backend/internal/config"

	"gorm.io/gorm"
)

type RepositoryAccess struct {
	Db     *gorm.DB
	Logger Logger
	Cfg    *config.Config
}

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	With(ctx interface{}) Logger
}

type TXProvider = *gorm.DB





