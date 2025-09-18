// main.go
package main

import (
	"auth/internal/config"
	"auth/internal/database"
	"auth/internal/handlers"
	"auth/internal/middleware"
	_ "auth/internal/models"
	"auth/internal/repository"
	"auth/internal/service"
	"context"
	"errors"
	_ "fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

var (
	cfg *config.Config
	db  *gorm.DB
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, using system environment variables")
	}
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

func setupRouter() *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(loggingMiddleware())
	router.Use(gin.Recovery())

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Статические файлы
	router.Static("/static", "./static")
	router.StaticFile("/", "./static/index.html")
	router.StaticFile("/login.html", "./static/login.html")
	router.StaticFile("/register.html", "./static/register.html")
	router.StaticFile("/profile.html", "./static/profile.html")

	// Инициализация сервисов
	userRepo := repository.NewUserRepository(db)
	jwtService := service.NewJWTService()
	userService := service.NewUserService(userRepo, jwtService)
	jwtMiddleware := middleware.NewJWTMiddleware(jwtService, userService)
	authHandlers := handlers.NewAuthHandlers(userService, jwtService)

	// Auth endpoints
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandlers.Register)
		auth.POST("/login", authHandlers.Login)
		auth.POST("/refresh", authHandlers.RefreshToken)
		auth.POST("/logout", authHandlers.Logout)
	}

	// Защищенные endpoints
	protected := router.Group("/api/v1")
	protected.Use(jwtMiddleware.RequireAuth())
	{
		protected.GET("/auth/me", authHandlers.GetProfile)
	}

	return router
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Определяем уровень лога по статус коду
		logLevel := slog.LevelInfo
		if statusCode >= 400 && statusCode < 500 {
			logLevel = slog.LevelWarn
		} else if statusCode >= 500 {
			logLevel = slog.LevelError
		}

		slog.Log(context.Background(), logLevel, "HTTP request completed",
			"method", method,
			"path", path,
			"status", statusCode,
			"latency_ms", latency.Milliseconds(),
			"ip", clientIP,
			"user_agent", c.Request.UserAgent(),
		)
	}
}
