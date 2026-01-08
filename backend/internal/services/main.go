package services

import (
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/repositories"

	"gorm.io/gorm"
)

type Services struct {
	User         ServiceUserMethods
	Connection   ServiceConnectionMethods
	Integration  ServiceIntegrationMethods
	Sync         ServiceSyncMethods
	GoogleDrive  ServiceGoogleDriveMethods
	QuickBooks   ServiceQuickBooksMethods
}

func NewServices(cfg *config.Config, db *gorm.DB, repos *repositories.Repositories, logger Logger) *Services {
	access := &ServiceAccess{
		Cfg:          cfg,
		Db:           db,
		Logger:       logger,
		Repositories: repos,
	}

	services := &Services{
		User:         NewServiceUser(access),
		Connection:   NewServiceConnection(access),
		Integration:  NewServiceIntegration(access),
		Sync:         NewServiceSync(access),
		GoogleDrive:  NewServiceGoogleDrive(access),
		QuickBooks:   NewServiceQuickBooks(access),
	}

	// Update access to include services for cross-service access
	access.Services = services

	return services
}





