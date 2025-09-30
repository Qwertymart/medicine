// internal/handlers/session_manager.go - ИСПРАВЛЕННАЯ ВЕРСИЯ
package handlers

import (
	"fmt"
	"log"
	"sync"
	"time"

	"CTG_monitor/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionManager управляет жизненным циклом сессий КТГ
type SessionManager struct {
	db             *gorm.DB
	activeSessions map[string]*models.CTGSession
	sessionsLock   sync.RWMutex
	dataBuffer     *DataBuffer

	// Callbacks для уведомления о событиях сессий
	onSessionStart func(session *models.CTGSession)
	onSessionStop  func(session *models.CTGSession)
}

// NewSessionManager создает новый менеджер сессий
func NewSessionManager(db *gorm.DB, dataBuffer *DataBuffer) *SessionManager {
	manager := &SessionManager{
		db:             db,
		activeSessions: make(map[string]*models.CTGSession),
		dataBuffer:     dataBuffer,
	}

	log.Println("Session Manager инициализирован")
	return manager
}

// SetCallbacks устанавливает колбэки для событий сессий
func (sm *SessionManager) SetCallbacks(onStart, onStop func(session *models.CTGSession)) {
	sm.onSessionStart = onStart
	sm.onSessionStop = onStop
}

// StartSession создает и запускает новую сессию мониторинга
func (sm *SessionManager) StartSession(cardID uuid.UUID, deviceID string) (*models.CTGSession, error) {
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()

	// Проверяем, нет ли уже активной сессии для этого устройства
	if existing := sm.activeSessions[deviceID]; existing != nil {
		return nil, fmt.Errorf("активная сессия уже существует для устройства %s", deviceID)
	}

	// Создаем новую сессию
	session := &models.CTGSession{
		ID:        uuid.New(),
		CardID:    cardID,
		DeviceID:  deviceID,
		StartTime: time.Now().UTC(),
		FHRData: models.CTGTimeSeries{
			Points:   []models.CTGPoint{},
			Count:    0,
			LastTime: 0,
		},
		UCData: models.CTGTimeSeries{
			Points:   []models.CTGPoint{},
			Count:    0,
			LastTime: 0,
		},
	}

	// Сохраняем в БД
	if err := sm.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("не удалось создать сессию в БД: %w", err)
	}

	// Добавляем в активные сессии
	sm.activeSessions[deviceID] = session

	// Уведомляем о начале сессии
	if sm.onSessionStart != nil {
		sm.onSessionStart(session)
	}

	log.Printf("Запущена сессия %s для устройства %s, карта %s",
		session.ID.String(), deviceID, cardID.String())

	return session, nil
}

// StopSession завершает активную сессию
func (sm *SessionManager) StopSession(sessionID uuid.UUID) (*models.CTGSession, error) {
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()

	// Ищем активную сессию
	var targetDeviceID string
	var targetSession *models.CTGSession

	for deviceID, session := range sm.activeSessions {
		if session.ID == sessionID {
			targetDeviceID = deviceID
			targetSession = session
			break
		}
	}

	if targetSession == nil {
		return nil, fmt.Errorf("активная сессия %s не найдена", sessionID.String())
	}

	// Устанавливаем время завершения
	now := time.Now().UTC()
	targetSession.EndTime = &now

	// Обновляем в БД
	if err := sm.db.Model(targetSession).Update("end_time", now).Error; err != nil {
		return nil, fmt.Errorf("не удалось обновить сессию в БД: %w", err)
	}

	// Удаляем из активных сессий
	delete(sm.activeSessions, targetDeviceID)

	// Очищаем буфер данных для этой сессии
	sm.dataBuffer.RemoveSessionBuffer(sessionID)

	// Уведомляем о завершении сессии
	if sm.onSessionStop != nil {
		sm.onSessionStop(targetSession)
	}

	log.Printf("✅ Завершена сессия %s для устройства %s", sessionID.String(), targetDeviceID)
	return targetSession, nil
}

// GetActiveSession возвращает активную сессию для устройства
func (sm *SessionManager) GetActiveSession(deviceID string) *models.CTGSession {
	sm.sessionsLock.RLock()
	defer sm.sessionsLock.RUnlock()
	return sm.activeSessions[deviceID]
}

// GetAllActiveSessions возвращает все активные сессии
func (sm *SessionManager) GetAllActiveSessions() []*models.CTGSession {
	sm.sessionsLock.RLock()
	defer sm.sessionsLock.RUnlock()

	sessions := make([]*models.CTGSession, 0, len(sm.activeSessions))
	for _, session := range sm.activeSessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetActiveSessionCount возвращает количество активных сессий
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.sessionsLock.RLock()
	defer sm.sessionsLock.RUnlock()
	return len(sm.activeSessions)
}

// GetSession получает сессию из БД по ID
func (sm *SessionManager) GetSession(sessionID uuid.UUID) (*models.CTGSession, error) {
	var session models.CTGSession
	if err := sm.db.First(&session, "id = ?", sessionID).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// GetSessionsByCardID получает все сессии для медицинской карты
func (sm *SessionManager) GetSessionsByCardID(cardID uuid.UUID) ([]*models.CTGSession, error) {
	var sessions []*models.CTGSession
	if err := sm.db.Where("card_id = ?", cardID).
		Order("start_time DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetAllDevices возвращает список всех устройств из БД
func (sm *SessionManager) GetAllDevices() []string {
	var devices []string
	sm.db.Model(&models.CTGSession{}).
		Distinct("device_id").
		Pluck("device_id", &devices)
	return devices
}

// GetSessionStatistics возвращает статистику сессий
func (sm *SessionManager) GetSessionStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	// Активные сессии
	activeSessions := sm.GetAllActiveSessions()
	stats["active_sessions_count"] = len(activeSessions)

	// Статистика по устройствам
	deviceStats := make(map[string]interface{})
	for _, session := range activeSessions {
		duration := time.Since(session.StartTime).Seconds()
		deviceStats[session.DeviceID] = map[string]interface{}{
			"session_id": session.ID.String(),
			"start_time": session.StartTime,
			"duration":   duration,
			"card_id":    session.CardID.String(),
		}
	}
	stats["devices"] = deviceStats

	// Общее количество сессий в БД
	var totalSessions int64
	sm.db.Model(&models.CTGSession{}).Count(&totalSessions)
	stats["total_sessions"] = totalSessions

	return stats
}

// CleanupInactiveSessions проверяет и очищает зависшие сессии
func (sm *SessionManager) CleanupInactiveSessions() {
	sm.sessionsLock.Lock()
	defer sm.sessionsLock.Unlock()

	var sessionsToRemove []string
	threshold := time.Now().Add(-24 * time.Hour)

	for deviceID, session := range sm.activeSessions {
		if session.StartTime.Before(threshold) {
			now := time.Now().UTC()
			session.EndTime = &now
			sm.db.Model(session).Update("end_time", now)

			sessionsToRemove = append(sessionsToRemove, deviceID)
			log.Printf("Принудительно завершена зависшая сессия: %s", session.ID.String())
		}
	}

	// Удаляем зависшие сессии
	for _, deviceID := range sessionsToRemove {
		delete(sm.activeSessions, deviceID)
	}

	if len(sessionsToRemove) > 0 {
		log.Printf("Очищено %d зависших сессий", len(sessionsToRemove))
	}
}
