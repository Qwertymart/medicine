package handlers


import (
   "net/http"
   "time"


   "ml-service/internal/services"
   "github.com/gin-gonic/gin"
)


// MLHandler обрабатывает HTTP запросы для ML
type MLHandler struct {
   mlService *services.MLService
}


// NewMLHandler создает новый обработчик ML запросов
func NewMLHandler(mlService *services.MLService) *MLHandler {
   return &MLHandler{mlService: mlService}
}


// PredictRequest структура запроса на предсказание
type PredictRequest struct {
   CardID     string `json:"card_id" binding:"required"`
   TargetTime int    `json:"target_time" binding:"required"`
}


// FeaturesRequest структура запроса на вычисление фичей
type FeaturesRequest struct {
   CardID     string `json:"card_id" binding:"required"`
   TargetTime int    `json:"target_time" binding:"required"`
}


// Predict обрабатывает запрос на ML предсказание
func (h *MLHandler) Predict(c *gin.Context) {
   var req PredictRequest
   if err := c.ShouldBindJSON(&req); err != nil {
       c.JSON(http.StatusBadRequest, gin.H{
           "error":   "invalid request",
           "details": err.Error(),
       })
       return
   }


   response, err := h.mlService.ProcessMLRequest(req.CardID, req.TargetTime)
   if err != nil {
       c.JSON(http.StatusInternalServerError, gin.H{
           "error":   "ml service error",
           "details": err.Error(),
       })
       return
   }
   c.JSON(http.StatusOK, response)
}


// CalculateFeatures обрабатывает запрос на вычисление фичей
func (h *MLHandler) CalculateFeatures(c *gin.Context) {
   var req FeaturesRequest
   if err := c.ShouldBindJSON(&req); err != nil {
       c.JSON(http.StatusBadRequest, gin.H{
           "error":   "invalid request",
           "details": err.Error(),
       })
       return
   }


   features, err := h.mlService.CalculateFeatures(req.CardID, req.TargetTime)
   if err != nil {
       c.JSON(http.StatusInternalServerError, gin.H{
           "error":   "feature calculation error",
           "details": err.Error(),
       })
       return
   }
   c.JSON(http.StatusOK, features)
}


// Health проверяет состояние сервиса
func (h *MLHandler) Health(c *gin.Context) {
   c.JSON(http.StatusOK, gin.H{
       "status":    "healthy",
       "timestamp": time.Now().UTC(),
   })
}
