package middlewares

import (
	"syncflow-backend/internal/config"

	"github.com/gin-gonic/gin"
)

type Middlewares struct {
	CORS   gin.HandlerFunc
	Logger gin.HandlerFunc
}

func NewMiddlewares(cfg *config.Config, logger interface{}) *Middlewares {
	return &Middlewares{
		CORS:   CORS(),
		Logger: Logger(),
	}
}

