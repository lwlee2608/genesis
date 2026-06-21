package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	internalhttp "github.com/lwlee2608/project-00/internal/api/http"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var AppVersion = "dev"

func main() {
	if err := InitConfig(); err != nil {
		slog.Error("Failed to init config", "error", err)
		return
	}

	slog.Info("project-00", "version", AppVersion)

	services := &internalhttp.Services{}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	engine.Use(gin.Recovery())
	internalhttp.SetupRoute(engine, services)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Http.Port),
		Handler: engine,
	}

	slog.Info("Starting HTTP server", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server error", "error", err)
	}
}
