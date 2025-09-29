package handlers

import (
	"net/http"
	"time"

	_ "CTG_monitor/internal/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title CTG Monitor API
// @version 1.0
// @description API для системы мониторинга КТГ (кардиотокографии)
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @tag.name sessions
// @tag.description Управление сессиями мониторинга

// @tag.name monitoring
// @tag.description Мониторинг состояния сервиса

// RESTAPIServer обрабатывает REST API запросы
type RESTAPIServer struct {
	sessionManager *SessionManager
	grpcStreamer   *GRPCStreamer
	mqttProcessor  *MQTTStreamProcessor
}

// SessionRequest запрос для создания сессии
// @Description Данные для создания новой сессии мониторинга
type SessionRequest struct {
	CardID   string `json:"card_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"` // UUID медицинской карты пациента
	DeviceID string `json:"device_id" binding:"required" example:"CTG-DEVICE-001"`                     // Идентификатор устройства КТГ
}

// SessionResponse ответ с информацией о сессии
// @Description Информация о сессии мониторинга КТГ
type SessionResponse struct {
	SessionID string     `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440001"` // UUID сессии
	CardID    string     `json:"card_id" example:"550e8400-e29b-41d4-a716-446655440000"`    // UUID медицинской карты
	DeviceID  string     `json:"device_id" example:"CTG-DEVICE-001"`                        // Идентификатор устройства
	Status    string     `json:"status" example:"active" enums:"active,stopped"`            // Статус сессии
	StartTime time.Time  `json:"start_time" example:"2023-09-01T10:00:00Z"`                 // Время начала сессии
	EndTime   *time.Time `json:"end_time,omitempty" example:"2023-09-01T11:30:00Z"`         // Время окончания сессии (если завершена)
	Duration  int        `json:"duration" example:"5400"`                                   // Продолжительность в секундах
}

// SessionDataResponse данные КТГ для сессии
// @Description Данные мониторинга КТГ, собранные во время сессии
type SessionDataResponse struct {
	SessionID   string      `json:"session_id" example:"550e8400-e29b-41d4-a716-446655440001"` // UUID сессии
	FHRData     interface{} `json:"fhr_data"`                                                  // Данные частоты сердечных сокращений плода
	UCData      interface{} `json:"uc_data"`                                                   // Данные маточных сокращений
	TotalPoints int         `json:"total_points" example:"1250"`                               // Общее количество точек данных
}

// CardSessionsResponse сессии для медицинской карты
// @Description Список сессий для конкретной медицинской карты
type CardSessionsResponse struct {
	CardID   string            `json:"card_id" example:"550e8400-e29b-41d4-a716-446655440000"` // UUID медицинской карты
	Sessions []SessionResponse `json:"sessions"`                                               // Список сессий
	Count    int               `json:"count" example:"5"`                                      // Количество сессий
}

// DevicesResponse список устройств
// @Description Список всех доступных устройств КТГ
type DevicesResponse struct {
	Devices []string `json:"devices" example:"CTG-DEVICE-001,CTG-DEVICE-002"` // Список идентификаторов устройств
	Count   int      `json:"count" example:"2"`                               // Количество устройств
}

// DeviceStatusResponse статус устройства
// @Description Текущий статус устройства КТГ
type DeviceStatusResponse struct {
	DeviceID  string     `json:"device_id" example:"CTG-DEVICE-001"`                                  // Идентификатор устройства
	Status    string     `json:"status" example:"active" enums:"active,idle"`                         // Статус устройства
	SessionID *string    `json:"session_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"` // UUID активной сессии (если есть)
	StartTime *time.Time `json:"start_time,omitempty" example:"2023-09-01T10:00:00Z"`                 // Время начала активной сессии
	Duration  *int       `json:"duration,omitempty" example:"3600"`                                   // Продолжительность активной сессии в секундах
}

// HealthResponse состояние сервиса
// @Description Информация о состоянии и работоспособности сервиса
type HealthResponse struct {
	Status         string    `json:"status" example:"healthy"`                 // Статус сервиса
	Service        string    `json:"service" example:"CTG Monitor"`            // Название сервиса
	Timestamp      time.Time `json:"timestamp" example:"2023-09-01T10:00:00Z"` // Время проверки
	ActiveSessions int       `json:"active_sessions" example:"3"`              // Количество активных сессий
}

// CleanupResponse результат очистки сессий
// @Description Результат операции очистки зависших сессий
type CleanupResponse struct {
	Message        string `json:"message" example:"Очистка сессий выполнена"` // Сообщение о результате
	ActiveSessions int    `json:"active_sessions" example:"2"`                // Количество активных сессий после очистки
}

// ActiveSessionsResponse список активных сессий
// @Description Список всех активных сессий мониторинга
type ActiveSessionsResponse struct {
	Sessions []SessionResponse `json:"sessions"`          // Список активных сессий
	Count    int               `json:"count" example:"3"` // Количество активных сессий
}

// ErrorResponse стандартный ответ об ошибке
// @Description Стандартная структура ответа об ошибке
type ErrorResponse struct {
	Error   string `json:"error" example:"Неверный формат данных"`     // Описание ошибки
	Details string `json:"details,omitempty" example:"field required"` // Дополнительные детали ошибки
}

// SuccessResponse стандартный ответ об успехе
// @Description Стандартная структура успешного ответа
type SuccessResponse struct {
	Message string      `json:"message" example:"Операция выполнена успешно"` // Сообщение об успехе
	Data    interface{} `json:"data,omitempty"`                               // Дополнительные данные
}

// NewRESTAPIServer создает новый REST API сервер
func NewRESTAPIServer(
	sessionManager *SessionManager,
	grpcStreamer *GRPCStreamer,
	mqttProcessor *MQTTStreamProcessor,
) *RESTAPIServer {
	return &RESTAPIServer{
		sessionManager: sessionManager,
		grpcStreamer:   grpcStreamer,
		mqttProcessor:  mqttProcessor,
	}
}

// SetupRoutes настраивает маршруты REST API
func (api *RESTAPIServer) SetupRoutes() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:80", "*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// API группа
	api_group := r.Group("/api/v1")

	// === УПРАВЛЕНИЕ СЕССИЯМИ ===
	sessions := api_group.Group("/sessions")
	{
		sessions.POST("/start", api.StartSession)
		sessions.POST("/stop/:session_id", api.StopSession)
		//sessions.GET("/active", api.GetActiveSessions)
		//sessions.GET("/:session_id", api.GetSession)
		//sessions.GET("/:session_id/data", api.GetSessionData)
	}

	// === МЕДИЦИНСКИЕ КАРТЫ ===
	//cards := api_group.Group("/cards")
	//{
	//	cards.GET("/:card_id/sessions", api.GetCardSessions)
	//}

	// === УСТРОЙСТВА ===
	//devices := api_group.Group("/devices")
	//{
	//	devices.GET("/", api.GetDevices)
	//	devices.GET("/:device_id/status", api.GetDeviceStatus)
	//}

	// === МОНИТОРИНГ СЕРВИСА ===
	monitoring := api_group.Group("/monitoring")
	{
		monitoring.GET("/health", api.HealthCheck)
		monitoring.POST("/cleanup", api.CleanupSessions)
	}

	return r
}

// StartSession запускает новую сессию мониторинга
// @Summary Запуск новой сессии мониторинга КТГ
// @Description Создает новую сессию мониторинга КТГ для указанной медицинской карты и устройства
// @Tags sessions
// @Accept json
// @Produce json
// @Param request body SessionRequest true "Данные для создания сессии"
// @Success 200 {object} SuccessResponse{data=SessionResponse} "Сессия успешно запущена"
// @Failure 400 {object} ErrorResponse "Неверный формат данных"
// @Failure 409 {object} ErrorResponse "Сессия для устройства уже активна"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /sessions/start [post]
func (api *RESTAPIServer) StartSession(c *gin.Context) {
	var req SessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Неверный формат данных",
			Details: err.Error(),
		})
		return
	}

	// Валидация UUID карты
	cardID, err := uuid.Parse(req.CardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Неверный ID медицинской карты",
		})
		return
	}

	// Проверка активной сессии
	if activeSession := api.sessionManager.GetActiveSession(req.DeviceID); activeSession != nil {
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   "Сессия для устройства уже активна",
			Details: "active_session_id: " + activeSession.ID.String(),
		})
		return
	}

	// Создание сессии
	session, err := api.sessionManager.StartSession(cardID, req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Не удалось создать сессию",
			Details: err.Error(),
		})
		return
	}

	response := SessionResponse{
		SessionID: session.ID.String(),
		CardID:    session.CardID.String(),
		DeviceID:  session.DeviceID,
		Status:    "active",
		StartTime: session.StartTime,
		Duration:  0,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Сессия успешно запущена",
		Data:    response,
	})
}

// StopSession завершает активную сессию
// @Summary Завершение активной сессии мониторинга
// @Description Завершает указанную активную сессию мониторинга КТГ
// @Tags sessions
// @Produce json
// @Param session_id path string true "UUID сессии" format(uuid)
// @Success 200 {object} SuccessResponse{data=SessionResponse} "Сессия успешно завершена"
// @Failure 400 {object} ErrorResponse "Неверный ID сессии"
// @Failure 404 {object} ErrorResponse "Сессия не найдена"
// @Router /sessions/stop/{session_id} [post]
func (api *RESTAPIServer) StopSession(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Неверный ID сессии",
		})
		return
	}

	session, err := api.sessionManager.StopSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "Сессия не найдена или уже завершена",
		})
		return
	}

	duration := 0
	if session.EndTime != nil {
		duration = int(session.EndTime.Sub(session.StartTime).Seconds())
	}

	response := SessionResponse{
		SessionID: session.ID.String(),
		CardID:    session.CardID.String(),
		DeviceID:  session.DeviceID,
		Status:    "stopped",
		StartTime: session.StartTime,
		EndTime:   session.EndTime,
		Duration:  duration,
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Message: "Сессия успешно завершена",
		Data:    response,
	})
	go SendSessionToMedicalRecords(sessionID)
}

// HealthCheck проверка здоровья сервиса
// @Summary Проверка состояния сервиса
// @Description Возвращает информацию о текущем состоянии и работоспособности сервиса мониторинга КТГ
// @Tags monitoring
// @Produce json
// @Success 200 {object} HealthResponse "Состояние сервиса"
// @Router /monitoring/health [get]
func (api *RESTAPIServer) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:         "healthy",
		Service:        "CTG Monitor",
		Timestamp:      time.Now().UTC(),
		ActiveSessions: api.sessionManager.GetActiveSessionCount(),
	})
}

// CleanupSessions очистка зависших сессий
// @Summary Очистка зависших сессий
// @Description Выполняет очистку зависших и неактивных сессий в системе
// @Tags monitoring
// @Produce json
// @Success 200 {object} CleanupResponse "Результат очистки"
// @Router /monitoring/cleanup [post]
func (api *RESTAPIServer) CleanupSessions(c *gin.Context) {
	api.sessionManager.CleanupInactiveSessions()
	c.JSON(http.StatusOK, CleanupResponse{
		Message:        "Очистка сессий выполнена",
		ActiveSessions: api.sessionManager.GetActiveSessionCount(),
	})
}
