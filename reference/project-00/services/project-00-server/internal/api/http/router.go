package http

import (
	"github.com/gin-gonic/gin"
	"github.com/lwlee2608/project-00/internal/api/http/handler"
	"github.com/lwlee2608/project-00/internal/api/http/middleware"
	"github.com/lwlee2608/project-00/internal/db/sqlc"
)

type Config struct {
	Port uint
}

type Services struct {
	Queries *sqlc.Queries
}

func SetupRoute(engine *gin.Engine, srvs *Services) {
	engine.Use(middleware.RequestLogger())
	engine.Use(middleware.ErrorHandler())

	healthHandler := handler.NewHealthHandler()

	engine.GET("/health", healthHandler.Check)

	apis := engine.Group("/api/v1")
	{
		_ = apis
		// Add your API routes here
	}
}
