package controllers

import (
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/services"
)

type Controllers struct {
	User         ControllerUserMethods
	Connection   ControllerConnectionMethods
	Integration  ControllerIntegrationMethods
	Sync         ControllerSyncMethods
	OAuth        OAuthMethods
	Company      ControllerCompanyMethods
	App          ControllerAppMethods
	Pipeline     *PipelineHandler
	DataObject   *DataObjectHandler
	GoogleSheets *GoogleSheetsHandler
}

func NewControllers(cfg *config.Config, logger Logger, services *services.Services) *Controllers {
	access := &ControllerAccess{
		Cfg:      cfg,
		Logger:   logger,
		Services: services,
	}

	return &Controllers{
		User:         NewControllerUser(access),
		Connection:   NewControllerConnection(access),
		Integration:  NewControllerIntegration(access),
		Sync:         NewControllerSync(access),
		OAuth:        NewOAuthHandler(),
		Company:      NewCompanyHandlerWithAccess(access),
		App:          NewAppHandlerWithAccess(access),
		Pipeline:     NewPipelineHandlerWithAccess(access),
		DataObject:   NewDataObjectHandlerWithAccess(access),
		GoogleSheets: NewGoogleSheetsHandler(),
	}
}
