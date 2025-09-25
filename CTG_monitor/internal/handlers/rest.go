package handlers

import (
	"net/http"
	"time"

	_ "CTG_monitor/internal/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RESTAPIServer обрабатывает REST API запросы
type RESTAPIServer struct {
	sessionManager *SessionManager
	grpcStreamer   *GRPCStreamer
	mqttProcessor  *MQTTStreamProcessor
}

// SessionRequest запрос для создания сессии
type SessionRequest struct {
	CardID   string `json:"card_id" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
}

// SessionResponse ответ с информацией о сессии
type SessionResponse struct {
	SessionID string     `json:"session_id"`
	CardID    string     `json:"card_id"`
	DeviceID  string     `json:"device_id"`
	Status    string     `json:"status"` // "active", "stopped"
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Duration  int        `json:"duration"` // секунды
}

// ServiceStatusResponse статус сервиса
type ServiceStatusResponse struct {
	Service        string                 `json:"service"`
	Status         string                 `json:"status"`
	Timestamp      time.Time              `json:"timestamp"`
	ActiveSessions int                    `json:"active_sessions"`
	StreamClients  int                    `json:"stream_clients"`
	BatchClients   int                    `json:"batch_clients"`
	Statistics     map[string]interface{} `json:"statistics"`
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

	// API группа
	api_group := r.Group("/api/v1")

	// === УПРАВЛЕНИЕ СЕССИЯМИ ===
	sessions := api_group.Group("/sessions")
	{
		sessions.POST("/start", api.StartSession)
		sessions.POST("/stop/:session_id", api.StopSession)
		sessions.GET("/active", api.GetActiveSessions)
		sessions.GET("/:session_id", api.GetSession)
		sessions.GET("/:session_id/data", api.GetSessionData)
	}

	// === МЕДИЦИНСКИЕ КАРТЫ ===
	cards := api_group.Group("/cards")
	{
		cards.GET("/:card_id/sessions", api.GetCardSessions)
	}

	// === УСТРОЙСТВА ===
	devices := api_group.Group("/devices")
	{
		devices.GET("/", api.GetDevices)
		devices.GET("/:device_id/status", api.GetDeviceStatus)
	}

	// === МОНИТОРИНГ СЕРВИСА ===
	monitoring := api_group.Group("/monitoring")
	{
		monitoring.GET("/status", api.GetServiceStatus)
		monitoring.GET("/health", api.HealthCheck)
		monitoring.POST("/cleanup", api.CleanupSessions)
	}

	return r
}

// StartSession запускает новую сессию мониторинга
func (api *RESTAPIServer) StartSession(c *gin.Context) {
	var req SessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Неверный формат данных",
			"details": err.Error(),
		})
		return
	}

	// Валидация UUID карты
	cardID, err := uuid.Parse(req.CardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный ID медицинской карты",
		})
		return
	}

	// Проверка активной сессии
	if activeSession := api.sessionManager.GetActiveSession(req.DeviceID); activeSession != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":             "Сессия для устройства уже активна",
			"active_session_id": activeSession.ID.String(),
		})
		return
	}

	// Создание сессии
	session, err := api.sessionManager.StartSession(cardID, req.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Не удалось создать сессию",
			"details": err.Error(),
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Сессия успешно запущена",
		"session": response,
	})
}

// StopSession завершает активную сессию
func (api *RESTAPIServer) StopSession(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный ID сессии",
		})
		return
	}

	session, err := api.sessionManager.StopSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Сессия не найдена или уже завершена",
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Сессия успешно завершена",
		"session": response,
	})
}

// GetActiveSessions возвращает список активных сессий
func (api *RESTAPIServer) GetActiveSessions(c *gin.Context) {
	sessions := api.sessionManager.GetAllActiveSessions()
	var responseList []SessionResponse

	for _, session := range sessions {
		duration := int(time.Since(session.StartTime).Seconds())
		responseList = append(responseList, SessionResponse{
			SessionID: session.ID.String(),
			CardID:    session.CardID.String(),
			DeviceID:  session.DeviceID,
			Status:    "active",
			StartTime: session.StartTime,
			Duration:  duration,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"sessions": responseList,
		"count":    len(responseList),
	})
}

// GetSession возвращает информацию о сессии
func (api *RESTAPIServer) GetSession(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный ID сессии",
		})
		return
	}

	session, err := api.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Сессия не найдена",
		})
		return
	}

	duration := 0
	status := "active"
	if session.EndTime != nil {
		duration = int(session.EndTime.Sub(session.StartTime).Seconds())
		status = "stopped"
	} else {
		duration = int(time.Since(session.StartTime).Seconds())
	}

	response := SessionResponse{
		SessionID: session.ID.String(),
		CardID:    session.CardID.String(),
		DeviceID:  session.DeviceID,
		Status:    status,
		StartTime: session.StartTime,
		EndTime:   session.EndTime,
		Duration:  duration,
	}

	c.JSON(http.StatusOK, response)
}

// GetSessionData возвращает данные КТГ для сессии
func (api *RESTAPIServer) GetSessionData(c *gin.Context) {
	sessionIDStr := c.Param("session_id")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный ID сессии",
		})
		return
	}

	session, err := api.sessionManager.GetSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Сессия не найдена",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session_id":   session.ID.String(),
		"fhr_data":     session.FHRData,
		"uc_data":      session.UCData,
		"total_points": session.FHRData.Count + session.UCData.Count,
	})
}

// GetCardSessions возвращает сессии для медицинской карты
func (api *RESTAPIServer) GetCardSessions(c *gin.Context) {
	cardIDStr := c.Param("card_id")
	cardID, err := uuid.Parse(cardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверный ID медицинской карты",
		})
		return
	}

	sessions, err := api.sessionManager.GetSessionsByCardID(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка получения сессий",
		})
		return
	}

	var responseList []SessionResponse
	for _, session := range sessions {
		duration := 0
		status := "active"
		if session.EndTime != nil {
			duration = int(session.EndTime.Sub(session.StartTime).Seconds())
			status = "stopped"
		} else {
			duration = int(time.Since(session.StartTime).Seconds())
		}

		responseList = append(responseList, SessionResponse{
			SessionID: session.ID.String(),
			CardID:    session.CardID.String(),
			DeviceID:  session.DeviceID,
			Status:    status,
			StartTime: session.StartTime,
			EndTime:   session.EndTime,
			Duration:  duration,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"card_id":  cardID.String(),
		"sessions": responseList,
		"count":    len(responseList),
	})
}

// GetDevices возвращает список устройств
func (api *RESTAPIServer) GetDevices(c *gin.Context) {
	devices := api.sessionManager.GetAllDevices()
	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

// GetDeviceStatus возвращает статус устройства
func (api *RESTAPIServer) GetDeviceStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	activeSession := api.sessionManager.GetActiveSession(deviceID)

	if activeSession != nil {
		c.JSON(http.StatusOK, gin.H{
			"device_id":  deviceID,
			"status":     "active",
			"session_id": activeSession.ID.String(),
			"start_time": activeSession.StartTime,
			"duration":   int(time.Since(activeSession.StartTime).Seconds()),
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"device_id":  deviceID,
			"status":     "idle",
			"session_id": nil,
		})
	}
}

// GetServiceStatus возвращает статус сервиса
func (api *RESTAPIServer) GetServiceStatus(c *gin.Context) {
	streamClients, batchClients := api.grpcStreamer.GetSubscriberCount()
	statistics := api.sessionManager.GetSessionStatistics()

	response := ServiceStatusResponse{
		Service:        "CTG Monitor",
		Status:         "healthy",
		Timestamp:      time.Now().UTC(),
		ActiveSessions: api.sessionManager.GetActiveSessionCount(),
		StreamClients:  streamClients,
		BatchClients:   batchClients,
		Statistics:     statistics,
	}

	c.JSON(http.StatusOK, response)
}

// HealthCheck проверка здоровья сервиса
func (api *RESTAPIServer) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"service":         "CTG Monitor",
		"timestamp":       time.Now().UTC(),
		"active_sessions": api.sessionManager.GetActiveSessionCount(),
	})
}

// CleanupSessions очистка зависших сессий
func (api *RESTAPIServer) CleanupSessions(c *gin.Context) {
	api.sessionManager.CleanupInactiveSessions()
	c.JSON(http.StatusOK, gin.H{
		"message":         "Очистка сессий выполнена",
		"active_sessions": api.sessionManager.GetActiveSessionCount(),
	})
}
