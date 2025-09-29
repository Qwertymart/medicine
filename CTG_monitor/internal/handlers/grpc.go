package handlers

import (
	_ "context"
	"fmt"
	"log"
	_ "net"
	"sync"
	"time"

	_ "CTG_monitor/internal/models"
	pb "CTG_monitor/proto"
)

// CTGStreamServer простая реализация gRPC сервера
type CTGStreamServer struct {
	pb.UnimplementedCTGStreamServiceServer

	// Каналы для стриминга данных
	subscribers map[string]chan *pb.CTGDataResponse
	mu          sync.RWMutex
}

// NewCTGStreamServer создание нового сервера
func NewCTGStreamServer() *CTGStreamServer {
	return &CTGStreamServer{
		subscribers: make(map[string]chan *pb.CTGDataResponse),
	}
}

// StreamCTGData стриминг данных КТГ (только значения)
func (s *CTGStreamServer) StreamCTGData(req *pb.StreamRequest, stream pb.CTGStreamService_StreamCTGDataServer) error {
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	log.Printf("🌊 Новый стриминг клиент: %s, устройства: %v", clientID, req.DeviceIds)

	// Создаем канал для клиента
	clientChan := make(chan *pb.CTGDataResponse, 1000)

	s.mu.Lock()
	s.subscribers[clientID] = clientChan
	s.mu.Unlock()

	// Очистка при отключении клиента
	defer func() {
		s.mu.Lock()
		delete(s.subscribers, clientID)
		close(clientChan)
		s.mu.Unlock()
		log.Printf("🔌 Клиент отключен: %s", clientID)
	}()

	// Отправка данных клиенту
	for {
		select {
		case data := <-clientChan:
			// Фильтруем по запрошенным устройствам и типам данных
			if s.shouldSendData(data, req) {
				if err := stream.Send(data); err != nil {
					log.Printf("❌ Ошибка отправки данных клиенту %s: %v", clientID, err)
					return err
				}
			}

		case <-stream.Context().Done():
			log.Printf("🛑 Контекст стрима завершен для клиента: %s", clientID)
			return stream.Context().Err()
		}
	}
}

// shouldSendData проверяет, нужно ли отправлять данные клиенту
func (s *CTGStreamServer) shouldSendData(data *pb.CTGDataResponse, req *pb.StreamRequest) bool {
	// Проверка устройства
	if len(req.DeviceIds) > 0 {
		found := false
		for _, deviceID := range req.DeviceIds {
			if data.DeviceId == deviceID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Проверка типа данных
	if len(req.DataTypes) > 0 {
		found := false
		for _, dataType := range req.DataTypes {
			if data.DataType == dataType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// BroadcastCTGData рассылка данных всем подписчикам (упрощенная версия)
func (s *CTGStreamServer) BroadcastCTGData(data *pb.CTGDataResponse) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Отправляем данные всем подписчикам
	for clientID, ch := range s.subscribers {
		select {
		case ch <- data:
			// Данные отправлены
		default:
			// Канал заполнен, пропускаем
			log.Printf("⚠️ Канал клиента %s переполнен, пропускаем данные", clientID)
		}
	}
}
