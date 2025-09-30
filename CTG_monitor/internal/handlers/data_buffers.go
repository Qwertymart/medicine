// internal/handlers/data_buffer.go - ЗАМЕНИТЬ session_buffer.go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"CTG_monitor/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DataBuffer управляет буферизацией данных для записи в БД
type DataBuffer struct {
	db             *gorm.DB
	sessionBuffers map[uuid.UUID]*SessionDataBuffer
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// SessionDataBuffer буфер для одной сессии
type SessionDataBuffer struct {
	SessionID uuid.UUID
	FHRBuffer []models.CTGPoint
	UCBuffer  []models.CTGPoint
	LastFlush time.Time
	mu        sync.Mutex
}

// NewDataBuffer создает новый буфер данных
func NewDataBuffer(db *gorm.DB) *DataBuffer {
	ctx, cancel := context.WithCancel(context.Background())

	buffer := &DataBuffer{
		db:             db,
		sessionBuffers: make(map[uuid.UUID]*SessionDataBuffer),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Запуск автофлаша
	buffer.wg.Add(1)
	go buffer.autoFlushWorker()

	log.Println("Data Buffer инициализирован")
	return buffer
}

// AddDataPoint добавляет точку данных в буфер
func (db *DataBuffer) AddDataPoint(sessionID uuid.UUID, dataType string, value, timeSec float64) {
	db.mu.RLock()
	sessionBuffer, exists := db.sessionBuffers[sessionID]
	db.mu.RUnlock()

	if !exists {
		db.mu.Lock()
		if sessionBuffer, exists = db.sessionBuffers[sessionID]; !exists {
			sessionBuffer = &SessionDataBuffer{
				SessionID: sessionID,
				FHRBuffer: make([]models.CTGPoint, 0, 500),
				UCBuffer:  make([]models.CTGPoint, 0, 500),
				LastFlush: time.Now(),
			}
			db.sessionBuffers[sessionID] = sessionBuffer
		}
		db.mu.Unlock()
	}

	sessionBuffer.mu.Lock()
	defer sessionBuffer.mu.Unlock()

	point := models.CTGPoint{
		T: timeSec,
		V: value,
	}

	switch dataType {
	case "fetal_heart_rate":
		sessionBuffer.FHRBuffer = append(sessionBuffer.FHRBuffer, point)
	case "uterine_contractions":
		sessionBuffer.UCBuffer = append(sessionBuffer.UCBuffer, point)
	}

	totalPoints := len(sessionBuffer.FHRBuffer) + len(sessionBuffer.UCBuffer)
	timeSinceFlush := time.Since(sessionBuffer.LastFlush)

	if totalPoints >= 100 || timeSinceFlush > 30*time.Second {
		go db.flushSessionAsync(sessionID)
	}
}

// FlushAll флашит все буферы
func (db *DataBuffer) FlushAll() {
	db.mu.RLock()
	var sessionIDs []uuid.UUID
	for sessionID := range db.sessionBuffers {
		sessionIDs = append(sessionIDs, sessionID)
	}
	db.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		db.flushSessionAsync(sessionID)
	}
}

// flushSessionAsync асинхронно флашит буфер сессии
func (db *DataBuffer) flushSessionAsync(sessionID uuid.UUID) {
	db.mu.RLock()
	sessionBuffer, exists := db.sessionBuffers[sessionID]
	db.mu.RUnlock()

	if !exists {
		return
	}

	sessionBuffer.mu.Lock()

	// Копируем данные для флаша
	fhrPoints := make([]models.CTGPoint, len(sessionBuffer.FHRBuffer))
	copy(fhrPoints, sessionBuffer.FHRBuffer)
	ucPoints := make([]models.CTGPoint, len(sessionBuffer.UCBuffer))
	copy(ucPoints, sessionBuffer.UCBuffer)

	// Очищаем буферы
	sessionBuffer.FHRBuffer = sessionBuffer.FHRBuffer[:0]
	sessionBuffer.UCBuffer = sessionBuffer.UCBuffer[:0]
	sessionBuffer.LastFlush = time.Now()

	sessionBuffer.mu.Unlock()

	if len(fhrPoints) == 0 && len(ucPoints) == 0 {
		return
	}

	// Записываем в БД
	if err := db.writeToDatabase(sessionID, fhrPoints, ucPoints); err != nil {
		log.Printf("❌ Ошибка записи в БД для сессии %s: %v", sessionID, err)
	} else {
		log.Printf("💾 Записано в БД: сессия %s, FHR=%d, UC=%d точек",
			sessionID, len(fhrPoints), len(ucPoints))
	}
}

// writeToDatabase записывает данные в БД пакетно
func (db *DataBuffer) writeToDatabase(sessionID uuid.UUID, fhrPoints, ucPoints []models.CTGPoint) error {
	updates := make(map[string]interface{})

	if len(fhrPoints) > 0 {
		fhrJSON, _ := json.Marshal(fhrPoints)
		// формируем строковое представление последнего времени
		lastTimeStr := strconv.FormatFloat(fhrPoints[len(fhrPoints)-1].T, 'f', -1, 64)

		updates["fhr_data"] = gorm.Expr(
			`jsonb_set(
       jsonb_set(
         jsonb_set(fhr_data,
           '{points}', COALESCE(fhr_data->'points','[]'::jsonb)||?::jsonb),
         '{count}', (COALESCE((fhr_data->>'count')::int,0)+?)::text::jsonb),
       '{last_time}', ?::text::jsonb)`,
			string(fhrJSON),
			len(fhrPoints),
			lastTimeStr,
		)
	}

	if len(ucPoints) > 0 {
		ucJSON, _ := json.Marshal(ucPoints)
		lastTimeUC := strconv.FormatFloat(ucPoints[len(ucPoints)-1].T, 'f', -1, 64)

		updates["uc_data"] = gorm.Expr(
			`jsonb_set(
       jsonb_set(
         jsonb_set(uc_data,
           '{points}', COALESCE(uc_data->'points','[]'::jsonb) || ?::jsonb),
         '{count}', (COALESCE((uc_data->>'count')::int, 0) + ?)::text::jsonb),
       '{last_time}', ?::text::jsonb)`,
			string(ucJSON),
			len(ucPoints),
			lastTimeUC,
		)
	}

	return db.db.Model(&models.CTGSession{}).
		Where("id = ?", sessionID).
		Updates(updates).Error
}

// RemoveSessionBuffer удаляет буфер завершенной сессии
func (db *DataBuffer) RemoveSessionBuffer(sessionID uuid.UUID) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, exists := db.sessionBuffers[sessionID]; exists {
		// Финальный флаш перед удалением
		go db.flushSessionAsync(sessionID)
		delete(db.sessionBuffers, sessionID)
		log.Printf("Удален буфер сессии: %s", sessionID)
	}
}

// autoFlushWorker периодически флашит старые буферы
func (db *DataBuffer) autoFlushWorker() {
	defer db.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			db.flushOldBuffers()
		case <-db.ctx.Done():
			db.finalFlush()
			log.Println("Auto flush worker остановлен")
			return
		}
	}
}

// flushOldBuffers флашит буферы, которые давно не флашились
func (db *DataBuffer) flushOldBuffers() {
	db.mu.RLock()
	var sessionsToFlush []uuid.UUID

	for sessionID, sessionBuffer := range db.sessionBuffers {
		if time.Since(sessionBuffer.LastFlush) > 15*time.Second {
			sessionsToFlush = append(sessionsToFlush, sessionID)
		}
	}
	db.mu.RUnlock()

	for _, sessionID := range sessionsToFlush {
		go db.flushSessionAsync(sessionID)
	}
}

// finalFlush финальный флаш при остановке
func (db *DataBuffer) finalFlush() {
	log.Println("🔄 Финальный флаш буферов...")

	db.mu.RLock()
	var sessionIDs []uuid.UUID
	for sessionID := range db.sessionBuffers {
		sessionIDs = append(sessionIDs, sessionID)
	}
	db.mu.RUnlock()

	for _, sessionID := range sessionIDs {
		db.flushSessionAsync(sessionID)
	}

	// Ждем завершения всех операций
	time.Sleep(2 * time.Second)
	log.Println("Финальный флаш завершен")
}

// Stop останавливает буфер
func (db *DataBuffer) Stop() {
	log.Println("Остановка Data Buffer...")
	db.cancel()
	db.wg.Wait()
	log.Println("Data Buffer остановлен")
}
