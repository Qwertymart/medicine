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

// GRPCStreamer управляет gRPC стримингом данных
type GRPCStreamer struct {
	pb.UnimplementedCTGStreamServiceServer

	// Подписчики
	subscribers  map[string]*StreamSubscriber
	batchClients map[string]*BatchSubscriber
	mu           sync.RWMutex

	// Каналы для батчинга
	batchBuffer map[string][]*pb.CTGDataResponse
	batchMu     sync.RWMutex
	batchTicker *time.Ticker

	// Управление
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// StreamSubscriber подписчик на потоковые данные
type StreamSubscriber struct {
	ID        string
	DeviceIDs []string
	DataTypes []string
	Channel   chan *pb.CTGDataResponse
	Stream    pb.CTGStreamService_StreamCTGDataServer
	Context   context.Context
}

// BatchSubscriber подписчик на батчевые данные
type BatchSubscriber struct {
	ID        string
	DeviceIDs []string
	Channel   chan []*pb.CTGDataResponse
	Stream    pb.CTGStreamService_StreamBatchCTGDataServer
	Context   context.Context
}

// NewGRPCStreamer создает новый стример
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

	log.Println("🌊 gRPC Streamer инициализирован")
	return streamer
}

// StreamCTGData потоковая передача данных КТГ
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

	// Обработка отправки данных
	for {
		select {
		case data := <-subscriber.Channel:
			if gs.shouldSendData(data, req) {
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

// StreamBatchCTGData батчевая передача данных (каждые 4 минуты)
func (gs *GRPCStreamer) StreamBatchCTGData(req *pb.StreamRequest, stream pb.CTGStreamService_StreamBatchCTGDataServer) error {
	clientID := fmt.Sprintf("batch_client_%d", time.Now().UnixNano())
	log.Printf("📦 Новый батчевый клиент подключен: %s, устройства: %v", clientID, req.DeviceIds)

	// Создаем батчевого подписчика
	subscriber := &BatchSubscriber{
		ID:        clientID,
		DeviceIDs: req.DeviceIds,
		Channel:   make(chan []*pb.CTGDataResponse, 100),
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

// BroadcastCTGData рассылает данные всем потоковым подписчикам
func (gs *GRPCStreamer) BroadcastCTGData(data *pb.CTGDataResponse) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	// Отправляем потоковым подписчикам
	for clientID, subscriber := range gs.subscribers {
		select {
		case subscriber.Channel <- data:
		default:
			log.Printf("⚠️ Канал потокового клиента %s переполнен", clientID)
		}
	}

	// Добавляем в батчевый буфер
	gs.batchMu.Lock()
	deviceKey := data.DeviceId
	gs.batchBuffer[deviceKey] = append(gs.batchBuffer[deviceKey], data)
	gs.batchMu.Unlock()
}

// BroadcastBatch отправляет батч определенному устройству
func (gs *GRPCStreamer) BroadcastBatch(deviceID string, batch []*pb.CTGDataResponse) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	// Отправляем батчевым подписчикам
	for clientID, subscriber := range gs.batchClients {
		// Проверяем, подписан ли клиент на это устройство
		if len(subscriber.DeviceIDs) == 0 || gs.containsDevice(subscriber.DeviceIDs, deviceID) {
			select {
			case subscriber.Channel <- batch:
			default:
				log.Printf("⚠️ Канал батчевого клиента %s переполнен", clientID)
			}
		}
	}
}

// batchProcessor обрабатывает накопленные батчи каждые 4 минуты
func (gs *GRPCStreamer) batchProcessor() {
	defer gs.wg.Done()

	for {
		select {
		case <-gs.batchTicker.C:
			gs.processBatches()
		case <-gs.ctx.Done():
			// Финальная обработка батчей
			gs.processBatches()
			log.Println("🛑 Batch processor остановлен")
			return
		}
	}
}

// processBatches обрабатывает накопленные батчи
func (gs *GRPCStreamer) processBatches() {
	gs.batchMu.Lock()
	defer gs.batchMu.Unlock()

	totalPoints := 0
	deviceCount := 0

	for deviceID, batch := range gs.batchBuffer {
		if len(batch) > 0 {
			gs.BroadcastBatch(deviceID, batch)
			totalPoints += len(batch)
			deviceCount++
		}
	}

	// Очищаем буферы
	gs.batchBuffer = make(map[string][]*pb.CTGDataResponse)

	if totalPoints > 0 {
		log.Printf("📦 Обработаны батчи: %d точек для %d устройств", totalPoints, deviceCount)
	}
}

// shouldSendData проверяет, нужно ли отправлять данные клиенту
func (gs *GRPCStreamer) shouldSendData(data *pb.CTGDataResponse, req *pb.StreamRequest) bool {
	// Проверка устройства
	if len(req.DeviceIds) > 0 && !gs.containsDevice(req.DeviceIds, data.DeviceId) {
		return false
	}

	// Проверка типа данных
	if len(req.DataTypes) > 0 && !gs.containsDataType(req.DataTypes, data.DataType) {
		return false
	}

	return true
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

// containsDataType проверяет наличие типа данных в списке
func (gs *GRPCStreamer) containsDataType(dataTypes []string, dataType string) bool {
	for _, dt := range dataTypes {
		if dt == dataType {
			return true
		}
	}
	return false
}

// GetSubscriberCount возвращает количество подписчиков
func (gs *GRPCStreamer) GetSubscriberCount() (int, int) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return len(gs.subscribers), len(gs.batchClients)
}

// Stop останавливает стример
func (gs *GRPCStreamer) Stop() {
	log.Println("🛑 Остановка gRPC Streamer...")
	gs.cancel()
	gs.batchTicker.Stop()
	gs.wg.Wait()
	log.Println("✅ gRPC Streamer остановлен")
}
