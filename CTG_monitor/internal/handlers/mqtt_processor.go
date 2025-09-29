// internal/handlers/mqtt_stream_processor.go - СПЕЦИАЛЬНАЯ ВЕРСИЯ ДЛЯ ЕДИНИЧНЫХ ВЫБРОСОВ

package handlers

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"CTG_monitor/internal/models"
	pb "CTG_monitor/proto"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SpikeDetectionFilter специальный фильтр для детекции единичных выбросов
type SpikeDetectionFilter struct {
	// Буферы для анализа контекста
	fhrBuffer  []float64 // Последние N значений ЧСС
	ucBuffer   []float64 // Последние N значений сокращений
	bufferSize int       // Размер буфера для анализа контекста

	// Параметры детекции спайков
	spikeDeviation  float64 // Минимальное отклонение для детекции спайка
	contextWindow   int     // Размер окна для анализа контекста (соседние точки)
	spikeConfidence float64 // Уровень уверенности для детекции спайка

	// Статистика
	totalProcessed int
	spikesDetected int

	mu sync.RWMutex
}

// NewSpikeDetectionFilter создает новый фильтр спайков
func NewSpikeDetectionFilter() *SpikeDetectionFilter {
	return &SpikeDetectionFilter{
		fhrBuffer:       make([]float64, 0, 20), // Буфер на 20 значений
		ucBuffer:        make([]float64, 0, 20),
		bufferSize:      20,
		spikeDeviation:  8.0, // Минимальное отклонение 8 единиц для спайка
		contextWindow:   3,   // Анализируем 3 точки до и после
		spikeConfidence: 0.7, // 70% уверенности для фильтрации
	}
}

// MQTTStreamProcessor обрабатывает потоковые данные от MQTT
type MQTTStreamProcessor struct {
	// Компоненты
	sessionManager *SessionManager
	grpcStreamer   *GRPCStreamer
	dataBuffer     *DataBuffer
	spikeFilter    *SpikeDetectionFilter

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
		spikeFilter:    NewSpikeDetectionFilter(),
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

	log.Println("🚀 MQTT Stream Processor со СПЕЦИАЛЬНОЙ фильтрацией единичных выбросов запущен")
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

// processData обрабатывает одну точку данных со специальной фильтрацией спайков
func (p *MQTTStreamProcessor) processData(data *models.MedicalData) {
	// 1. Проверка активной сессии
	session := p.sessionManager.GetActiveSession(data.DeviceID)
	if session == nil {
		cardID := uuid.New()
		var err error
		session, err = p.sessionManager.StartSession(cardID, data.DeviceID)
		if err != nil {
			log.Printf("❌ Ошибка создания автосессии для %s: %v", data.DeviceID, err)
			return
		}
		log.Printf("✅ Автоматически создана сессия для устройства: %s", data.DeviceID)
	}

	// 2. СПЕЦИАЛЬНАЯ ФИЛЬТРАЦИЯ ЕДИНИЧНЫХ ВЫБРОСОВ
	originalValue := data.Value

	// Добавляем значение в буфер и проверяем на выброс
	isSpike := p.spikeFilter.DetectSingleSpike(data.DataType, data.Value)

	if isSpike {
		// Заменяем спайк на интерполированное значение
		interpolatedValue := p.spikeFilter.InterpolateValue(data.DataType)
		data.Value = interpolatedValue
		log.Printf("🎯 ЕДИНИЧНЫЙ ВЫБРОС обнаружен и исправлен %s: %.2f -> %.2f",
			data.DataType, originalValue, interpolatedValue)
	}

	// 3. Базовая валидация диапазонов
	if !p.isValidDataRange(data) {
		data.Value = -1
		log.Printf("⛔ Значение вне допустимого диапазона %s: %.2f -> -1",
			data.DataType, originalValue)
	}

	// 4. Отправляем в gRPC стрим
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

	// 5. Добавляем в буфер для записи в БД
	p.dataBuffer.AddDataPoint(session.ID, data.DataType, data.Value, data.TimeSec)
}

// DetectSingleSpike обнаруживает единичные выбросы типа "30-30-30-50-30-30-30"
func (sf *SpikeDetectionFilter) DetectSingleSpike(dataType string, value float64) bool {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.totalProcessed++

	var buffer *[]float64
	switch dataType {
	case "fetal_heart_rate":
		buffer = &sf.fhrBuffer
	case "uterine_contractions":
		buffer = &sf.ucBuffer
	default:
		return false
	}

	// Добавляем новое значение
	*buffer = append(*buffer, value)
	if len(*buffer) > sf.bufferSize {
		*buffer = (*buffer)[1:]
	}

	// Нужно минимум 7 точек для анализа спайка (3 до + спайк + 3 после)
	if len(*buffer) < 7 {
		return false
	}

	// Анализируем текущую точку (предпоследнюю в буфере, так как последняя - новая)
	analyzeIndex := len(*buffer) - 2
	if analyzeIndex < sf.contextWindow {
		return false
	}

	currentValue := (*buffer)[analyzeIndex]

	// Анализируем контекст вокруг точки
	beforeValues := make([]float64, 0, sf.contextWindow)
	afterValues := make([]float64, 0, sf.contextWindow)

	// Собираем значения ДО предполагаемого спайка
	for i := analyzeIndex - sf.contextWindow; i < analyzeIndex; i++ {
		if i >= 0 {
			beforeValues = append(beforeValues, (*buffer)[i])
		}
	}

	// Собираем значения ПОСЛЕ предполагаемого спайка
	for i := analyzeIndex + 1; i <= analyzeIndex+sf.contextWindow && i < len(*buffer); i++ {
		afterValues = append(afterValues, (*buffer)[i])
	}

	// Должно быть достаточно контекстных точек
	if len(beforeValues) < 2 || len(afterValues) < 2 {
		return false
	}

	// Вычисляем средние значения до и после
	beforeMean := sf.calculateMean(beforeValues)
	afterMean := sf.calculateMean(afterValues)
	contextMean := (beforeMean + afterMean) / 2.0

	// Вычисляем стандартное отклонение контекста
	contextStd := sf.calculateStd(append(beforeValues, afterValues...), contextMean)

	// Проверяем условия для детекции спайка
	deviation := math.Abs(currentValue - contextMean)

	// Условие 1: Значение сильно отличается от контекста
	isDeviantFromContext := deviation > sf.spikeDeviation

	// Условие 2: Значения до и после спайка стабильны (похожи друг на друга)
	beforeAfterDiff := math.Abs(beforeMean - afterMean)
	isContextStable := beforeAfterDiff < sf.spikeDeviation/2.0

	// Условие 3: Статистическая значимость (если есть достаточно данных)
	isStatisticallySignificant := true
	if contextStd > 0 {
		zScore := deviation / contextStd
		isStatisticallySignificant = zScore > 2.0 // 2-сигма правило
	}

	// Условие 4: "Островной" спайк - соседние точки не являются спайками
	isIsolatedSpike := sf.checkIsolation(beforeValues, afterValues, currentValue)

	isSpike := isDeviantFromContext && isContextStable && isStatisticallySignificant && isIsolatedSpike

	if isSpike {
		sf.spikesDetected++
		log.Printf("🎯 ДЕТЕКЦИЯ СПАЙКА %s:")
		log.Printf("   Значение: %.2f, Контекст: %.2f (отклонение: %.2f)")
		log.Printf("   До спайка: %.2f, После спайка: %.2f (разность: %.2f)")
		log.Printf("   Z-score: %.2f, Изолированный: %v")

		// Обновляем статистику
		if sf.totalProcessed%100 == 0 {
			log.Printf("📊 Статистика фильтрации: %d/%d (%.1f%% спайков)",
				sf.spikesDetected, sf.totalProcessed,
				float64(sf.spikesDetected)/float64(sf.totalProcessed)*100)
		}
	}

	return isSpike
}

// InterpolateValue создает интерполированное значение вместо спайка
func (sf *SpikeDetectionFilter) InterpolateValue(dataType string) float64 {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	var buffer []float64
	switch dataType {
	case "fetal_heart_rate":
		buffer = sf.fhrBuffer
	case "uterine_contractions":
		buffer = sf.ucBuffer
	default:
		return -1
	}

	if len(buffer) < 4 {
		return -1
	}

	// Берем 2 точки до спайка и 2 точки после для интерполяции
	analyzeIndex := len(buffer) - 2 // Предпоследняя точка (спайк)

	if analyzeIndex < 2 || analyzeIndex >= len(buffer)-2 {
		return -1
	}

	// Линейная интерполяция между соседними стабильными точками
	beforeValue := buffer[analyzeIndex-1]
	afterValue := buffer[analyzeIndex+1]

	// Простая линейная интерполяция
	interpolated := (beforeValue + afterValue) / 2.0

	// Дополнительно учитываем тренд
	if analyzeIndex >= 3 && analyzeIndex < len(buffer)-2 {
		trendBefore := buffer[analyzeIndex-1] - buffer[analyzeIndex-2]
		trendAfter := buffer[analyzeIndex+2] - buffer[analyzeIndex+1]
		avgTrend := (trendBefore + trendAfter) / 2.0

		// Корректируем интерполяцию с учетом тренда
		interpolated += avgTrend * 0.1 // Небольшая коррекция на тренд
	}

	return interpolated
}

// calculateMean вычисляет среднее значение
func (sf *SpikeDetectionFilter) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStd вычисляет стандартное отклонение
func (sf *SpikeDetectionFilter) calculateStd(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	return math.Sqrt(variance / float64(len(values)-1))
}

// checkIsolation проверяет, является ли спайк изолированным (соседние точки не спайки)
func (sf *SpikeDetectionFilter) checkIsolation(beforeValues, afterValues []float64, spikeValue float64) bool {
	if len(beforeValues) == 0 || len(afterValues) == 0 {
		return false
	}

	// Проверяем, что соседние точки не отклоняются сильно от общего контекста
	lastBefore := beforeValues[len(beforeValues)-1]
	firstAfter := afterValues[0]

	// Среднее значение контекста (без спайка)
	allContext := append(beforeValues, afterValues...)
	contextMean := sf.calculateMean(allContext)

	// Соседние точки должны быть близки к контексту
	beforeDeviation := math.Abs(lastBefore - contextMean)
	afterDeviation := math.Abs(firstAfter - contextMean)
	spikeDeviation := math.Abs(spikeValue - contextMean)

	// Спайк должен отклоняться больше, чем соседние точки
	return beforeDeviation < spikeDeviation/2.0 && afterDeviation < spikeDeviation/2.0
}

// isValidDataRange базовая проверка диапазонов
func (p *MQTTStreamProcessor) isValidDataRange(data *models.MedicalData) bool {
	switch data.DataType {
	case "fetal_heart_rate":
		return data.Value == -1 || (data.Value >= 50 && data.Value <= 220)
	case "uterine_contractions":
		return data.Value == -1 || (data.Value >= -5 && data.Value <= 150)
	default:
		return true
	}
}

// grpcWorker отправляет данные в gRPC стрим
func (p *MQTTStreamProcessor) grpcWorker() {
	defer p.wg.Done()

	for {
		select {
		case data := <-p.grpcChannel:
			// Немедленная отправка для потокового режима
			p.grpcStreamer.BroadcastCTGData(data)

		case <-p.ctx.Done():
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

// Stop останавливает процессор
func (p *MQTTStreamProcessor) Stop() {
	log.Println("🛑 Остановка MQTT Stream Processor...")
	p.cancel()
	p.wg.Wait()
	close(p.dataChannel)
	close(p.grpcChannel)
	log.Println("✅ MQTT Stream Processor остановлен")
}
