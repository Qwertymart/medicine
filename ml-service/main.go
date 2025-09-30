package main

import (
	"log"
	"ml-service/config"
	"ml-service/internal/database"
	"ml-service/internal/handlers"
	"ml-service/internal/services"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "ml-service/docs" // Сгенерированные swagger docs
)

// @title ML Service API
// @version 1.0
// @description API для предиктивного анализа CTG данных
// @termsOfService http://swagger.io/terms/

// @contact.name Itelma Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8052
// @BasePath /api/v1
// @schemes http https

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	dataService := services.NewDataService(db)
	// Передаём dataService и URL
	mlService := services.NewMLService(dataService, cfg.ML.ServiceURL)
	// Передаём только mlService
	handler := handlers.NewMLHandler(mlService)

	r := gin.Default()
	api := r.Group("/api/v1")
	{
		ml := api.Group("/ml")
		{
			ml.GET("/health", handler.Health)
			ml.POST("/features", handler.CalculateFeatures)
			ml.POST("/predict", handler.Predict)
		}
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Printf("Запуск ML сервиса на порту %s", cfg.Port)
	log.Fatal(r.Run(":" + cfg.Port))
}
