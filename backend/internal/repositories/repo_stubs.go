package repositories

import (
	"syncflow-backend/internal/models"

	"github.com/google/uuid"
)

// Stub repository implementations to allow server to start

type RepositoryUserMethods interface {
	GetByID(id uuid.UUID) (*models.User, error)
	Create(user *models.User) error
}

type RepositoryConnectionMethods interface{}
type RepositoryIntegrationMethods interface{}
type RepositorySyncJobMethods interface{}
type RepositorySyncRecordMethods interface{}

type repoUser struct {
	access *RepositoryAccess
}

func NewRepositoryUser(access *RepositoryAccess) RepositoryUserMethods {
	return &repoUser{access: access}
}

func (r *repoUser) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.access.Db.First(&user, id).Error
	return &user, err
}

func (r *repoUser) Create(user *models.User) error {
	return r.access.Db.Create(user).Error
}

func NewRepositoryConnection(access *RepositoryAccess) RepositoryConnectionMethods {
	return &struct{}{}
}

func NewRepositoryIntegration(access *RepositoryAccess) RepositoryIntegrationMethods {
	return &struct{}{}
}

func NewRepositorySyncJob(access *RepositoryAccess) RepositorySyncJobMethods {
	return &struct{}{}
}

func NewRepositorySyncRecord(access *RepositoryAccess) RepositorySyncRecordMethods {
	return &struct{}{}
}

