package main


import (
   "log"
   "net/http"
  
   "ml-service/config"
   "ml-service/internal/database"
   "ml-service/internal/handlers"
   "ml-service/internal/services"
  
   "github.com/gin-gonic/gin"
)


func main() {
   // Загрузка конфигурации
   cfg := config.Load()
  
   // Подключение к БД
   db, err := database.Connect(cfg)
   if err != nil {
       log.Fatalf("Ошибка подключения к БД: %v", err)
   }
  
   // Инициализация сервисов
   dataService := services.NewDataService(db)
   mlService := services.NewMLService(dataService, cfg.ML.ServiceURL)
  
   // Инициализация обработчиков
   mlHandler := handlers.NewMLHandler(mlService)
  
   // Настройка роутера
   router := gin.Default()
  
   // Middleware
   router.Use(gin.Logger())
   router.Use(gin.Recovery())
  
   // API endpoints
   api := router.Group("/api/v1")
   {
       api.POST("/ml/features", mlHandler.CalculateFeatures)
       api.POST("/ml/predict", mlHandler.Predict)
       api.GET("/ml/health", mlHandler.Health)
   }
  
   log.Printf("Запуск ML сервиса на порту %s", cfg.Port)
   log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
