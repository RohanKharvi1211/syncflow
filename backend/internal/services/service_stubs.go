package services

// Stub service implementations

type ServiceUserMethods interface{}
type ServiceConnectionMethods interface{}
type ServiceIntegrationMethods interface{}
type ServiceSyncMethods interface{}
type ServiceGoogleDriveMethods interface{}
type ServiceQuickBooksMethods interface{}

func NewServiceUser(access *ServiceAccess) ServiceUserMethods {
	return &struct{}{}
}

func NewServiceConnection(access *ServiceAccess) ServiceConnectionMethods {
	return &struct{}{}
}

func NewServiceIntegration(access *ServiceAccess) ServiceIntegrationMethods {
	return &struct{}{}
}

func NewServiceSync(access *ServiceAccess) ServiceSyncMethods {
	return &struct{}{}
}

func NewServiceGoogleDrive(access *ServiceAccess) ServiceGoogleDriveMethods {
	return &struct{}{}
}

func NewServiceQuickBooks(access *ServiceAccess) ServiceQuickBooksMethods {
	return &struct{}{}
}





