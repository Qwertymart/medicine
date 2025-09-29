// internal/handlers/grpc_streamer.go - ЗАМЕНИТЬ grpc.go

package handlers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pb "CTG_monitor/proto"
)

// GRPCStreamer управляет только батчевой отправкой данных каждые 4 минуты
type GRPCStreamer struct {
	pb.UnimplementedCTGStreamServiceServer

	// Батчевые клиенты
	batchClients map[string]*BatchSubscriber
	mu           sync.RWMutex

	// Буфер для накопления данных
	batchBuffer map[string][]*pb.CTGDataResponse
	batchMu     sync.RWMutex
	subscribers map[string]*StreamSubscriber

	// Таймер строго на 4 минуты
	batchTicker *time.Ticker

	// Управление жизненным циклом
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// BatchSubscriber подписчик на батчевые данные
type BatchSubscriber struct {
	ID        string
	DeviceIDs []string
	Channel   chan []*pb.CTGDataResponse
	Stream    pb.CTGStreamService_StreamBatchCTGDataServer
	Context   context.Context
}

type StreamSubscriber struct {
	ID        string
	DeviceIDs []string
	DataTypes []string
	Channel   chan *pb.CTGDataResponse
	Stream    pb.CTGStreamService_StreamCTGDataServer
	Context   context.Context
}

// NewGRPCStreamer создает новый батчевый стример
func NewGRPCStreamer() *GRPCStreamer {
	ctx, cancel := context.WithCancel(context.Background())

	streamer := &GRPCStreamer{
		subscribers:  make(map[string]*StreamSubscriber),
		batchClients: make(map[string]*BatchSubscriber),
		batchBuffer:  make(map[string][]*pb.CTGDataResponse),
		batchTicker:  time.NewTicker(4 * time.Minute),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Запуск батчевого процессора
	streamer.wg.Add(1)
	go streamer.batchProcessor()

	log.Println("📦 gRPC Batch Streamer инициализирован (только батчи каждые 4 минуты)")
	return streamer
}

// StreamBatchCTGData батчевая передача данных строго каждые 4 минуты
func (gs *GRPCStreamer) StreamBatchCTGData(req *pb.StreamRequest, stream pb.CTGStreamService_StreamBatchCTGDataServer) error {
	clientID := fmt.Sprintf("batch_client_%d", time.Now().UnixNano())
	log.Printf("📦 Новый батчевый клиент подключен: %s, устройства: %v", clientID, req.DeviceIds)

	// Создаем батчевого подписчика
	subscriber := &BatchSubscriber{
		ID:        clientID,
		DeviceIDs: req.DeviceIds,
		Channel:   make(chan []*pb.CTGDataResponse, 1000),
		Stream:    stream,
		Context:   stream.Context(),
	}

	// Регистрируем подписчика
	gs.mu.Lock()
	gs.batchClients[clientID] = subscriber
	gs.mu.Unlock()

	// Очистка при отключении
	defer func() {
		gs.mu.Lock()
		delete(gs.batchClients, clientID)
		close(subscriber.Channel)
		gs.mu.Unlock()
		log.Printf("📦 Батчевый клиент отключен: %s", clientID)
	}()

	// Обработка отправки батчей
	for {
		select {
		case batch := <-subscriber.Channel:
			if len(batch) == 0 {
				continue
			}

			batchResponse := &pb.CTGBatchResponse{
				Data:      batch,
				Timestamp: time.Now().Unix(),
				Count:     int32(len(batch)),
			}

			if err := stream.Send(batchResponse); err != nil {
				log.Printf("❌ Ошибка отправки батча клиенту %s: %v", clientID, err)
				return err
			}

			log.Printf("📤 Отправлен батч клиенту %s: %d точек", clientID, len(batch))

		case <-subscriber.Context.Done():
			log.Printf("🛑 Контекст батчевого клиента завершен: %s", clientID)
			return subscriber.Context.Err()
		}
	}
}

// BroadcastCTGData добавляет данные в буфер для батчевой отправки
func (gs *GRPCStreamer) BroadcastCTGData(data *pb.CTGDataResponse) {
	gs.mu.RLock()

	// Отправляем потоковым подписчикам
	for clientID, subscriber := range gs.subscribers {
		select {
		case subscriber.Channel <- data:
			// Успешно отправлено
		default:
			log.Printf("⚠️ Канал потокового клиента %s переполнен", clientID)
		}
	}
	gs.mu.RUnlock()

	// Добавляем в батчевый буфер
	gs.batchMu.Lock()
	defer gs.batchMu.Unlock()

	deviceKey := data.DeviceId
	gs.batchBuffer[deviceKey] = append(gs.batchBuffer[deviceKey], data)

	if len(gs.batchBuffer[deviceKey])%1000 == 0 {
		log.Printf("📊 Накоплено %d точек для устройства %s", len(gs.batchBuffer[deviceKey]), deviceKey)
	}
}

// batchProcessor обрабатывает накопленные батчи строго каждые 4 минуты
func (gs *GRPCStreamer) batchProcessor() {
	defer gs.wg.Done()

	log.Printf("⏰ Батчевый процессор запущен. Отправка каждые 4 минуты")

	for {
		select {
		case <-gs.batchTicker.C:
			gs.processBatches()

		case <-gs.ctx.Done():
			// Финальная обработка батчей при завершении
			log.Println("🛑 Завершение работы - отправка финальных батчей")
			gs.processBatches()
			log.Println("🛑 Batch processor остановлен")
			return
		}
	}
}

// processBatches обрабатывает и отправляет все накопленные батчи
func (gs *GRPCStreamer) processBatches() {
	currentTime := time.Now()
	log.Printf("⏰ Запуск processBatches в %s", currentTime.Format("15:04:05.000"))

	gs.batchMu.Lock()

	// Копируем текущие буферы для отправки
	batchesToSend := make(map[string][]*pb.CTGDataResponse)
	totalPoints := 0
	deviceCount := 0

	for deviceID, batch := range gs.batchBuffer {
		if len(batch) > 0 {
			// Создаем копию батча
			batchCopy := make([]*pb.CTGDataResponse, len(batch))
			copy(batchCopy, batch)
			batchesToSend[deviceID] = batchCopy

			totalPoints += len(batch)
			deviceCount++

			log.Printf("📤 Подготовлен батч для %s: %d точек", deviceID, len(batch))
		}
	}

	// Очищаем буферы после копирования
	gs.batchBuffer = make(map[string][]*pb.CTGDataResponse)
	gs.batchMu.Unlock()

	// Отправляем батчи клиентам (вне блокировки)
	if len(batchesToSend) > 0 {
		for deviceID, batch := range batchesToSend {
			gs.broadcastBatch(deviceID, batch)
		}

		log.Printf("📦 Обработаны временные батчи: %d точек для %d устройств в %s",
			totalPoints, deviceCount, currentTime.Format("15:04:05.000"))
	} else {
		log.Printf("📦 Нет данных для отправки в %s", currentTime.Format("15:04:05.000"))
	}
}

// broadcastBatch отправляет батч всем подписанным клиентам
func (gs *GRPCStreamer) broadcastBatch(deviceID string, batch []*pb.CTGDataResponse) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	for clientID, subscriber := range gs.batchClients {
		// Проверяем, подписан ли клиент на это устройство
		if len(subscriber.DeviceIDs) == 0 || gs.containsDevice(subscriber.DeviceIDs, deviceID) {
			select {
			case subscriber.Channel <- batch:
				// Успешно отправлено
			default:
				log.Printf("⚠️ Канал батчевого клиента %s переполнен", clientID)
			}
		}
	}
}

// containsDevice проверяет наличие устройства в списке
func (gs *GRPCStreamer) containsDevice(devices []string, deviceID string) bool {
	for _, device := range devices {
		if device == deviceID {
			return true
		}
	}
	return false
}

// GetBatchSubscriberCount возвращает количество батчевых подписчиков
func (gs *GRPCStreamer) GetBatchSubscriberCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return len(gs.batchClients)
}

// GetBufferStatus возвращает статус буферов
func (gs *GRPCStreamer) GetBufferStatus() map[string]int {
	gs.batchMu.RLock()
	defer gs.batchMu.RUnlock()

	status := make(map[string]int)
	for deviceID, buffer := range gs.batchBuffer {
		status[deviceID] = len(buffer)
	}
	return status
}

// Stop останавливает стример
func (gs *GRPCStreamer) Stop() {
	log.Println("🛑 Остановка gRPC Batch Streamer...")
	gs.cancel()
	gs.batchTicker.Stop()
	gs.wg.Wait()
	log.Println("✅ gRPC Batch Streamer остановлен")
}

func (gs *GRPCStreamer) StreamCTGData(req *pb.StreamRequest, stream pb.CTGStreamService_StreamCTGDataServer) error {
	clientID := fmt.Sprintf("stream_client_%d", time.Now().UnixNano())
	log.Printf("🔌 Новый потоковый клиент подключен: %s, устройства: %v", clientID, req.DeviceIds)

	// Создаем подписчика
	subscriber := &StreamSubscriber{
		ID:        clientID,
		DeviceIDs: req.DeviceIds,
		DataTypes: req.DataTypes,
		Channel:   make(chan *pb.CTGDataResponse, 2000),
		Stream:    stream,
		Context:   stream.Context(),
	}

	// Регистрируем подписчика
	gs.mu.Lock()
	gs.subscribers[clientID] = subscriber
	gs.mu.Unlock()

	// Очистка при отключении
	defer func() {
		gs.mu.Lock()
		delete(gs.subscribers, clientID)
		close(subscriber.Channel)
		gs.mu.Unlock()
		log.Printf("🔌 Потоковый клиент отключен: %s", clientID)
	}()

	var counter int

	// Обработка отправки данных
	for {
		select {
		case data := <-subscriber.Channel:
			if gs.shouldSendData(data, req, &counter) {
				if err := stream.Send(data); err != nil {
					log.Printf("❌ Ошибка отправки потоковых данных клиенту %s: %v", clientID, err)
					return err
				}
			}
		case <-subscriber.Context.Done():
			log.Printf("🛑 Контекст потокового клиента завершен: %s", clientID)
			return subscriber.Context.Err()
		}
	}
}

// shouldSendData проверяет, нужно ли отправлять данные клиенту
func (gs *GRPCStreamer) shouldSendData(data *pb.CTGDataResponse, req *pb.StreamRequest, counter *int) bool {
	if data.Value == -1 {
		return false
	}

	// Все остальные проверки без изменений...
	if len(req.DeviceIds) > 0 {
		deviceMatch := false
		for _, deviceID := range req.DeviceIds {
			if data.DeviceId == deviceID {
				deviceMatch = true
				break
			}
		}
		if !deviceMatch {
			return false
		}
	}

	if len(req.DataTypes) > 0 {
		typeMatch := false
		for _, dataType := range req.DataTypes {
			if data.DataType == dataType {
				typeMatch = true
				break
			}
		}
		if !typeMatch {
			return false
		}
	}

	//// Увеличиваем локальный счетчик
	//(*counter)++
	//
	//// Отправляем каждое 3-е значение
	return true
}

// containsDataType проверяет наличие типа данных в списке
func (gs *GRPCStreamer) containsDataType(dataTypes []string, dataType string) bool {
	for _, dt := range dataTypes {
		if dt == dataType {
			return true
		}
	}
	return false
}
