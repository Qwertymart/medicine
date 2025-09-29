package handlers

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"CTG_monitor/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SessionBuffer буферизует данные ТОЛЬКО для активных сессий
type SessionBuffer struct {
	db          *gorm.DB
	sessionMgr  *SessionManager                // ✅ СВЯЗЫВАЕМ с SessionManager
	buffers     map[uuid.UUID]*BufferedSession // sessionID -> buffer
	buffersMu   sync.RWMutex
	flushTicker *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
}

type BufferedSession struct {
	SessionID uuid.UUID
	DeviceID  string
	FHRBuffer []models.CTGPoint
	UCBuffer  []models.CTGPoint
	LastFlush time.Time
}

var sessionBuffer *SessionBuffer

// InitSessionBuffer инициализирует буфер с привязкой к SessionManager
func InitSessionBuffer(db *gorm.DB, sessionMgr *SessionManager) {
	ctx, cancel := context.WithCancel(context.Background())

	sessionBuffer = &SessionBuffer{
		db:          db,
		sessionMgr:  sessionMgr, // ✅ Привязываем SessionManager
		buffers:     make(map[uuid.UUID]*BufferedSession),
		flushTicker: time.NewTicker(5 * time.Second),
		ctx:         ctx,
		cancel:      cancel,
	}

	go sessionBuffer.autoFlushLoop()
	log.Println("🔄 Session buffer инициализирован с SessionManager")
}

// AddCTGDataPoint добавляет данные ТОЛЬКО для активных сессий
func AddCTGDataPoint(deviceID string, dataType string, value, timeSec float64) {
	if sessionBuffer == nil {
		return
	}

	// ✅ ПРОВЕРЯЕМ: есть ли активная сессия для этого устройства
	activeSession := sessionBuffer.sessionMgr.GetActiveSession(deviceID)
	if activeSession == nil {
		// Нет активной сессии - не сохраняем данные
		return
	}

	sessionBuffer.buffersMu.Lock()
	defer sessionBuffer.buffersMu.Unlock()

	// Получаем или создаем буфер для активной сессии
	buffer := sessionBuffer.getOrCreateBuffer(activeSession.ID, deviceID)

	point := models.CTGPoint{
		T: timeSec,
		V: value,
	}

	// Добавляем в соответствующий буфер
	switch dataType {
	case "fetal_heart_rate":
		buffer.FHRBuffer = append(buffer.FHRBuffer, point)
	case "uterine_contractions":
		buffer.UCBuffer = append(buffer.UCBuffer, point)
	}

	// Проверяем необходимость флаша
	totalPoints := len(buffer.FHRBuffer) + len(buffer.UCBuffer)
	timeSinceFlush := time.Since(buffer.LastFlush)

	if totalPoints >= 50 || timeSinceFlush > 10*time.Second {
		go sessionBuffer.flushBufferAsync(activeSession.ID)
	}
}

// getOrCreateBuffer получает или создает буфер для активной сессии
func (sb *SessionBuffer) getOrCreateBuffer(sessionID uuid.UUID, deviceID string) *BufferedSession {
	if buffer, exists := sb.buffers[sessionID]; exists {
		return buffer
	}

	// Создаем буфер только для существующей сессии
	buffer := &BufferedSession{
		SessionID: sessionID,
		DeviceID:  deviceID,
		FHRBuffer: make([]models.CTGPoint, 0, 200),
		UCBuffer:  make([]models.CTGPoint, 0, 200),
		LastFlush: time.Now(),
	}

	sb.buffers[sessionID] = buffer
	log.Printf("📝 Создан буфер для активной сессии: %s", sessionID)
	return buffer
}

// flushBufferAsync флашит конкретный буфер сессии
func (sb *SessionBuffer) flushBufferAsync(sessionID uuid.UUID) {
	sb.buffersMu.RLock()
	buffer, exists := sb.buffers[sessionID]
	if !exists {
		sb.buffersMu.RUnlock()
		return
	}

	// Копируем данные
	fhrPoints := make([]models.CTGPoint, len(buffer.FHRBuffer))
	copy(fhrPoints, buffer.FHRBuffer)

	ucPoints := make([]models.CTGPoint, len(buffer.UCBuffer))
	copy(ucPoints, buffer.UCBuffer)

	// Очищаем буферы
	buffer.FHRBuffer = buffer.FHRBuffer[:0]
	buffer.UCBuffer = buffer.UCBuffer[:0]
	buffer.LastFlush = time.Now()

	sb.buffersMu.RUnlock()

	if len(fhrPoints) == 0 && len(ucPoints) == 0 {
		return
	}

	// Обновляем сессию в БД
	sb.appendToSession(sessionID, fhrPoints, ucPoints)

	log.Printf("💾 Флаш буфера сессии %s: FHR=%d, UC=%d точек",
		sessionID, len(fhrPoints), len(ucPoints))
}

// appendToSession добавляет данные к существующей сессии (ИСПРАВЛЯЕМ JSONB поля)
func (sb *SessionBuffer) appendToSession(sessionID uuid.UUID, fhrPoints, ucPoints []models.CTGPoint) {
	updates := make(map[string]interface{})

	// ✅ ИСПРАВЛЯЕМ: используем правильные имена полей
	if len(fhrPoints) > 0 {
		fhrJSON, _ := json.Marshal(fhrPoints)
		updates["fhr_data"] = gorm.Expr(`
            jsonb_set(
                jsonb_set(
                    jsonb_set(fhr_data, '{fhr_points}', COALESCE(fhr_data->'fhr_points', '[]'::jsonb) || ?::jsonb),
                    '{count}', 
                    (COALESCE((fhr_data->>'count')::int, 0) + ?)::text::jsonb
                ),
                '{last_time}',
                ?::text::jsonb
            )
        `, string(fhrJSON), len(fhrPoints), fhrPoints[len(fhrPoints)-1].T)
	}

	if len(ucPoints) > 0 {
		ucJSON, _ := json.Marshal(ucPoints)
		updates["uc_data"] = gorm.Expr(`
            jsonb_set(
                jsonb_set(
                    jsonb_set(uc_data, '{uc_points}', COALESCE(uc_data->'uc_points', '[]'::jsonb) || ?::jsonb),
                    '{count}', 
                    (COALESCE((uc_data->>'count')::int, 0) + ?)::text::jsonb
                ),
                '{last_time}',
                ?::text::jsonb
            )
        `, string(ucJSON), len(ucPoints), ucPoints[len(ucPoints)-1].T)
	}

	if err := sb.db.Model(&models.CTGSession{}).Where("id = ?", sessionID).Updates(updates).Error; err != nil {
		log.Printf("❌ Ошибка аппенда к сессии: %v", err)
		return
	}

	log.Printf("✅ Аппенд к сессии %s выполнен", sessionID)
}

// ✅ ДОБАВЛЯЕМ: функция для очистки буферов завершенных сессий
func (sb *SessionBuffer) CleanupFinishedSessions() {
	sb.buffersMu.Lock()
	defer sb.buffersMu.Unlock()

	var toRemove []uuid.UUID
	for sessionID := range sb.buffers {
		// Проверяем, активна ли еще сессия
		var session models.CTGSession
		err := sb.db.First(&session, "id = ? AND end_time IS NULL", sessionID).Error
		if err != nil {
			// Сессия завершена или не найдена - удаляем буфер
			toRemove = append(toRemove, sessionID)
		}
	}

	for _, sessionID := range toRemove {
		delete(sb.buffers, sessionID)
		log.Printf("🧹 Удален буфер завершенной сессии: %s", sessionID)
	}
}

// autoFlushLoop с очисткой завершенных сессий
func (sb *SessionBuffer) autoFlushLoop() {
	cleanupTicker := time.NewTicker(60 * time.Second) // Очистка каждую минуту
	defer cleanupTicker.Stop()

	for {
		select {
		case <-sb.flushTicker.C:
			// Флашим все активные буферы
			sb.buffersMu.RLock()
			var sessionsToFlush []uuid.UUID
			for sessionID, buffer := range sb.buffers {
				if time.Since(buffer.LastFlush) > 8*time.Second {
					sessionsToFlush = append(sessionsToFlush, sessionID)
				}
			}
			sb.buffersMu.RUnlock()

			for _, sessionID := range sessionsToFlush {
				go sb.flushBufferAsync(sessionID)
			}

		case <-cleanupTicker.C:
			// Очищаем буферы завершенных сессий
			sb.CleanupFinishedSessions()

		case <-sb.ctx.Done():
			log.Println("🛑 Останавливаем автофлаш сессий")
			sb.finalFlush()
			return
		}
	}
}

// Остальные функции остаются без изменений...
func (sb *SessionBuffer) finalFlush() {
	sb.buffersMu.RLock()
	var sessionIDs []uuid.UUID
	for sessionID := range sb.buffers {
		sessionIDs = append(sessionIDs, sessionID)
	}
	sb.buffersMu.RUnlock()

	log.Printf("🔄 Финальный флаш %d буферов сессий", len(sessionIDs))

	for _, sessionID := range sessionIDs {
		sb.flushBufferAsync(sessionID)
	}

	time.Sleep(2 * time.Second)
	log.Println("✅ Финальный флаш завершен")
}

func CloseSessionBuffer() {
	if sessionBuffer != nil {
		sessionBuffer.cancel()
		sessionBuffer.flushTicker.Stop()
		log.Println("🔒 Session buffer закрыт")
	}
}

func getLastTime(points []models.CTGPoint) float64 {
	if len(points) == 0 {
		return 0.0
	}
	return points[len(points)-1].T
}
