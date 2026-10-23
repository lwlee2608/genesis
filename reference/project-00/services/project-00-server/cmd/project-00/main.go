package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	internalhttp "github.com/lwlee2608/project-00/internal/api/http"
	"github.com/lwlee2608/project-00/internal/db"
	"github.com/lwlee2608/project-00/internal/db/sqlc"
)

var AppVersion = "dev"

func main() {
	if err := InitConfig(); err != nil {
		slog.Error("Failed to init config", "error", err)
		os.Exit(1)
	}

	slog.Info("project-00", "version", AppVersion)

	if config.DB.URL == "" {
		slog.Error("db.url is required")
		os.Exit(1)
	}

	if err := db.RunMigrations(config.DB.URL, config.DB.Schema); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	dbPool, err := db.InitDB(context.Background(), config.DB)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	services := &internalhttp.Services{
		Queries: sqlc.New(dbPool),
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "fc00::/7"})
	internalhttp.SetupRoute(engine, services)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Http.Port),
		Handler: engine,
	}

	slog.Info("Starting HTTP server", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}
