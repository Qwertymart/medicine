// internal/handlers/mqtt_stream_processor.go - ИСПРАВЛЕННАЯ ВЕРСИЯ
package handlers

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"strings"
	"sync"
	"time"

	"CTG_monitor/internal/models"
	pb "CTG_monitor/proto"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MQTTStreamProcessor обрабатывает потоковые данные от MQTT
type MQTTStreamProcessor struct {
	// Компоненты
	sessionManager *SessionManager
	grpcStreamer   *GRPCStreamer
	dataBuffer     *DataBuffer

	// Каналы для потоковой обработки
	dataChannel chan *models.MedicalData
	grpcChannel chan *pb.CTGDataResponse

	// Управление
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// NewMQTTStreamProcessor создает новый процессор потоковых данных
func NewMQTTStreamProcessor(
	sessionManager *SessionManager,
	grpcStreamer *GRPCStreamer,
	dataBuffer *DataBuffer,
) *MQTTStreamProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	processor := &MQTTStreamProcessor{
		sessionManager: sessionManager,
		grpcStreamer:   grpcStreamer,
		dataBuffer:     dataBuffer,
		dataChannel:    make(chan *models.MedicalData, 1000),
		grpcChannel:    make(chan *pb.CTGDataResponse, 1000),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Запуск воркеров
	processor.wg.Add(3)
	go processor.dataWorker()   // Обработка данных
	go processor.grpcWorker()   // gRPC стриминг
	go processor.bufferWorker() // Буферизация

	log.Println("🚀 MQTT Stream Processor запущен")
	return processor
}

// HandleIncomingMQTT главный обработчик MQTT сообщений
func (p *MQTTStreamProcessor) HandleIncomingMQTT(topic string, payload []byte) {
	// Парсинг топика: medical/ctg/{datatype}/{deviceID}
	parts := strings.Split(topic, "/")
	if len(parts) != 4 {
		log.Printf("⚠️ Неверный формат топика: %s", topic)
		return
	}

	dataType := parts[2]

	// Парсинг JSON
	var data models.MedicalData
	if err := json.Unmarshal(payload, &data); err != nil {
		log.Printf("❌ Ошибка парсинга MQTT payload: %v", err)
		return
	}

	// Заполнение из топика, если не указано
	data.DeviceID = p.sessionManager.GetAllDevices()[0]
	if data.DataType == "" {
		data.DataType = dataType
	}

	// Отправляем в канал для обработки
	select {
	case p.dataChannel <- &data:
	default:
		log.Printf("⚠️ Канал данных переполнен, пропускаем сообщение")
	}
}

// MessageHandler обработчик MQTT сообщений (глобальная функция)
func MessageHandler(client mqtt.Client, msg mqtt.Message) {
	// Эта функция должна быть связана с конкретным процессором
	// В main.go нужно использовать замыкание для передачи процессора
	log.Printf("📡 MQTT сообщение получено: %s", msg.Topic())
}

// dataWorker обрабатывает входящие данные
func (p *MQTTStreamProcessor) dataWorker() {
	defer p.wg.Done()

	for {
		select {
		case data := <-p.dataChannel:
			p.processData(data)
		case <-p.ctx.Done():
			log.Println("🛑 Data worker остановлен")
			return
		}
	}
}

// processData обрабатывает одну точку данных
func (p *MQTTStreamProcessor) processData(data *models.MedicalData) {
	// 1. Проверка активной сессии
	session := p.sessionManager.GetActiveSession(data.DeviceID)
	if session == nil {
		// Автоматически создаем сессию с дефолтной картой
		cardID := uuid.New() // или получить из конфига
		var err error
		session, err = p.sessionManager.StartSession(cardID, data.DeviceID)
		if err != nil {
			log.Printf("❌ Ошибка создания автосессии для %s: %v", data.DeviceID, err)
			return
		}
		log.Printf("✅ Автоматически создана сессия для устройства: %s", data.DeviceID)
	}

	// 2. Валидация данных
	if !p.isValidData(data) {
		data.Value = -1 // Маркер невалидных данных
	}

	// 3. Отправляем в gRPC стрим немедленно (потоковый режим)
	grpcData := &pb.CTGDataResponse{
		DeviceId: data.DeviceID,
		DataType: data.DataType,
		Value:    data.Value,
		TimeSec:  data.TimeSec,
	}

	select {
	case p.grpcChannel <- grpcData:
	default:
		log.Printf("⚠️ gRPC канал переполнен для устройства %s", data.DeviceID)
	}

	// 4. Добавляем в буфер для записи в БД
	p.dataBuffer.AddDataPoint(session.ID, data.DataType, data.Value, data.TimeSec)
}

// grpcWorker отправляет данные в gRPC стрим
func (p *MQTTStreamProcessor) grpcWorker() {
	defer p.wg.Done()

	// Буфер для батчинга отправки по 4 минуты
	batchBuffer := make([]*pb.CTGDataResponse, 0, 1000)
	batchTimer := time.NewTimer(4 * time.Minute)

	for {
		select {
		case data := <-p.grpcChannel:
			// Немедленная отправка для потокового режима
			p.grpcStreamer.BroadcastCTGData(data)

			// Добавляем в батч буфер
			batchBuffer = append(batchBuffer, data)

			// Отправляем батч если накопилось достаточно или время вышло
			if len(batchBuffer) >= 100 {
				p.sendBatch(batchBuffer)
				batchBuffer = batchBuffer[:0]
				batchTimer.Reset(4 * time.Minute)
			}

		case <-batchTimer.C:
			// Отправляем накопленный батч по таймеру
			if len(batchBuffer) > 0 {
				p.sendBatch(batchBuffer)
				batchBuffer = batchBuffer[:0]
			}
			batchTimer.Reset(4 * time.Minute)

		case <-p.ctx.Done():
			// Отправляем оставшиеся данные
			if len(batchBuffer) > 0 {
				p.sendBatch(batchBuffer)
			}
			log.Println("🛑 gRPC worker остановлен")
			return
		}
	}
}

// bufferWorker периодически флашит буфер в БД
func (p *MQTTStreamProcessor) bufferWorker() {
	defer p.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.dataBuffer.FlushAll()
		case <-p.ctx.Done():
			// Финальный флаш
			p.dataBuffer.FlushAll()
			log.Println("🛑 Buffer worker остановлен")
			return
		}
	}
}

// sendBatch отправляет батч данных в gRPC
func (p *MQTTStreamProcessor) sendBatch(batch []*pb.CTGDataResponse) {
	// Группируем данные по устройствам для оптимальной отправки
	deviceBatches := make(map[string][]*pb.CTGDataResponse)
	for _, data := range batch {
		deviceBatches[data.DeviceId] = append(deviceBatches[data.DeviceId], data)
	}

	// Отправляем батчи для каждого устройства
	for _, deviceBatch := range deviceBatches {
		for _, data := range deviceBatch {
			p.grpcStreamer.BroadcastCTGData(data)
		}
	}

	log.Printf("📦 Отправлен батч данных: %d точек для %d устройств",
		len(batch), len(deviceBatches))
}

// isValidData проверяет валидность данных
func (p *MQTTStreamProcessor) isValidData(data *models.MedicalData) bool {
	switch data.DataType {
	case "fetal_heart_rate":
		return data.Value >= 50 && data.Value <= 220
	case "uterine_contractions":
		return data.Value >= -5 && data.Value <= 150
	default:
		return true
	}
}

// Stop останавливает процессор
func (p *MQTTStreamProcessor) Stop() {
	log.Println("🛑 Остановка MQTT Stream Processor...")
	p.cancel()
	p.wg.Wait()
	close(p.dataChannel)
	close(p.grpcChannel)
	log.Println("✅ MQTT Stream Processor остановлен")
}
