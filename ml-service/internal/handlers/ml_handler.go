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


// Predict выполняет полный ML pipeline
// @Summary Предиктивный анализ CTG данных
// @Description Вычисляет фичи и выполняет ML предсказание состояния плода
// @Tags ml
// @Accept json
// @Produce json
// @Param request body models.MLRequest true "Запрос на предсказание"
// @Success 200 {object} models.MLResponse "Результат предсказания"
// @Failure 400 {object} models.ErrorResponse "Неверный запрос"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /ml/predict [post]
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

// CalculateFeatures вычисляет фичи для указанного пациента и времени
// @Summary Вычисление фичей CTG данных
// @Description Рассчитывает статистические фичи на основе CTG данных пациента
// @Tags ml
// @Accept json
// @Produce json
// @Param request body models.MLRequest true "Запрос на вычисление фичей"
// @Success 200 {object} models.FeaturesResponse "Вычисленные фичи"
// @Failure 400 {object} models.ErrorResponse "Неверный запрос"
// @Failure 500 {object} models.ErrorResponse "Внутренняя ошибка сервера"
// @Router /ml/features [post]
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
// @Summary Проверка состояния ML сервиса
// @Description Возвращает статус работы ML сервиса
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Сервис работает"
// @Router /ml/health [get]
func (h *MLHandler) Health(c *gin.Context) {
   c.JSON(http.StatusOK, gin.H{
       "status":    "healthy",
       "timestamp": time.Now().UTC(),
   })
}
