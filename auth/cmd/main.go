// main.go
package main

import (
	"auth/internal/config"
	"auth/internal/database"
	_ "auth/internal/models"
	"context"
	"errors"
	_ "fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var (
	cfg *config.Config
	db  *gorm.DB
)

func main() {
	config.InitLogger()
	slog.Info("Starting application", "version", "1.0.0")

	cfg = config.Load()
	slog.Info("Configuration loaded successfully",
		"server_port", cfg.Server.Port,
		"gin_mode", cfg.Server.Mode,
		"db_host", cfg.Database.Host,
	)
	db = database.Connect(cfg.Database)

	gin.SetMode(cfg.Server.Mode)

	router := setupRouter()

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("Starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Failed to start server", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("Server started successfully", "port", cfg.Server.Port)

	waitForShutdown(server)
}

func waitForShutdown(server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("Shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	slog.Info("Server gracefully stopped")
}
