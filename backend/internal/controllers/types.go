package controllers

import (
	"syncflow-backend/internal/config"
	"syncflow-backend/internal/services"

	"github.com/gin-gonic/gin"
)

type ControllerAccess struct {
	Cfg      *config.Config
	Logger   Logger
	Services *services.Services
}

type Logger interface {
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	With(ctx interface{}) Logger
	WithRequest(c *gin.Context, r interface{}) interface{}
}





