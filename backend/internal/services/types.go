package services

import (
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/repositories"

	"gorm.io/gorm"
)

type ServiceAccess struct {
	Cfg          *config.Config
	Db           *gorm.DB
	Logger       Logger
	Repositories *repositories.Repositories
	Services     *Services
}

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	With(ctx interface{}) Logger
}





