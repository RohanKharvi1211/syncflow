package repositories

import (
	"syncflow-backend/internal/config"

	"gorm.io/gorm"
)

type Repositories struct {
	User         RepositoryUserMethods
	Connection   RepositoryConnectionMethods
	Integration  RepositoryIntegrationMethods
	SyncJob      RepositorySyncJobMethods
	SyncRecord   RepositorySyncRecordMethods
}

func NewRepositories(db *gorm.DB, logger Logger, cfg *config.Config) *Repositories {
	access := &RepositoryAccess{
		Db:     db,
		Logger: logger,
		Cfg:    cfg,
	}

	return &Repositories{
		User:         NewRepositoryUser(access),
		Connection:   NewRepositoryConnection(access),
		Integration:  NewRepositoryIntegration(access),
		SyncJob:      NewRepositorySyncJob(access),
		SyncRecord:   NewRepositorySyncRecord(access),
	}
}





