package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"

	"CTG_monitor/internal/models"
	pb "CTG_monitor/proto"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Глобальная ссылка на gRPC сервер для стриминга
var grpcStreamServer *CTGStreamServer

// SetGRPCStreamServer устанавливает ссылку на gRPC сервер
func SetGRPCStreamServer(server *CTGStreamServer) {
	grpcStreamServer = server
}

// NoiseBuffer буфер для анализа тренда
type NoiseBuffer struct {
	values   []float64
	maxSize  int
	dataType string
}

// Глобальные буферы для разных устройств и типов данных
var (
	deviceBuffers = make(map[string]map[string]*NoiseBuffer)
)

// getOrCreateBuffer получает или создает буфер для устройства и типа данных
func getOrCreateBuffer(deviceID, dataType string) *NoiseBuffer {
	if deviceBuffers[deviceID] == nil {
		deviceBuffers[deviceID] = make(map[string]*NoiseBuffer)
	}

	if deviceBuffers[deviceID][dataType] == nil {
		deviceBuffers[deviceID][dataType] = &NoiseBuffer{
			values:   make([]float64, 0, 10),
			maxSize:  10, // храним последние 10 значений
			dataType: dataType,
		}
	}

	return deviceBuffers[deviceID][dataType]
}

// addValue добавляет значение в буфер (пропускаем -1)
func (nb *NoiseBuffer) addValue(value float64) {
	// НЕ добавляем -1 в буфер для анализа тренда
	if value == -1 {
		return
	}

	if len(nb.values) >= nb.maxSize {
		// Сдвигаем буфер
		copy(nb.values, nb.values[1:])
		nb.values = nb.values[:nb.maxSize-1]
	}
	nb.values = append(nb.values, value)
}

// isValidValue проверяет физиологические пределы
func isValidValue(value float64, dataType string) bool {
	switch dataType {
	case "fetal_heart_rate":
		// Более широкие пределы для FHR
		return value >= 30 && value <= 300 && !math.IsNaN(value) && !math.IsInf(value, 0)

	case "uterine_contractions":
		// Более широкие пределы для UC - отрицательные значения могут быть артефактами базовой линии
		return value >= -10 && value <= 200 && !math.IsNaN(value) && !math.IsInf(value, 0)

	default:
		return !math.IsNaN(value) && !math.IsInf(value, 0)
	}
}

// isCriticalAnomaly проверяет критические медицинские аномалии
func isCriticalAnomaly(value float64, dataType string) bool {
	switch dataType {
	case "fetal_heart_rate":
		// Только критические случаи
		return value < 50 || value > 250

	case "uterine_contractions":
		// Только явно невозможные значения
		return value < -5 || value > 150

	default:
		return false
	}
}

// isMotionArtifact умная проверка на артефакты движения
func (nb *NoiseBuffer) isMotionArtifact(newValue float64) bool {
	if len(nb.values) < 3 {
		return false // Недостаточно данных для анализа
	}

	lastValue := nb.values[len(nb.values)-1]
	jump := math.Abs(newValue - lastValue)

	switch nb.dataType {
	case "fetal_heart_rate":
		// Для FHR: резкий скачок >80 уд/мин считаем артефактом
		if jump > 80 {
			return true
		}

		// Дополнительная проверка: если значение выходит далеко за пределы недавнего тренда
		if len(nb.values) >= 5 {
			recentMean := nb.getRecentMean(5)
			recentStd := nb.getRecentStd(5, recentMean)

			// Если новое значение более чем в 4 стандартных отклонениях от среднего
			if math.Abs(newValue-recentMean) > 4*recentStd && recentStd > 5 {
				return true
			}
		}

	case "uterine_contractions":
		// Для UC: более мягкие критерии
		// Резкий скачок >60 мм рт.ст. может быть артефактом
		if jump > 60 {
			return true
		}

		// Проверка на физиологичность: UC не может мгновенно подняться с 0 до >50
		if lastValue < 10 && newValue > 50 && jump > 40 {
			return true
		}

	default:
		return false
	}

	return false
}

// getRecentMean вычисляет среднее для последних n значений
func (nb *NoiseBuffer) getRecentMean(n int) float64 {
	if len(nb.values) == 0 {
		return 0
	}

	start := len(nb.values) - n
	if start < 0 {
		start = 0
	}

	sum := 0.0
	count := 0
	for i := start; i < len(nb.values); i++ {
		sum += nb.values[i]
		count++
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// getRecentStd вычисляет стандартное отклонение
func (nb *NoiseBuffer) getRecentStd(n int, mean float64) float64 {
	if len(nb.values) < 2 {
		return 0
	}

	start := len(nb.values) - n
	if start < 0 {
		start = 0
	}

	sum := 0.0
	count := 0
	for i := start; i < len(nb.values); i++ {
		diff := nb.values[i] - mean
		sum += diff * diff
		count++
	}

	if count < 2 {
		return 0
	}
	return math.Sqrt(sum / float64(count-1))
}

func MessageHandler(client mqtt.Client, msg mqtt.Message) {
	var data models.MedicalData
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return
	}

	// Получаем буфер для данного устройства и типа данных
	buffer := getOrCreateBuffer(data.DeviceID, data.DataType)

	// Сохраняем оригинальное значение
	originalValue := data.Value
	isNoiseDetected := false
	noiseType := ""

	// 1. Проверка на базовую валидность
	if !isValidValue(data.Value, data.DataType) {
		data.Value = -1
		isNoiseDetected = true
		noiseType = "INVALID_VALUE"
	} else if isCriticalAnomaly(data.Value, data.DataType) {
		// 2. Проверка на критические аномалии
		data.Value = -1
		isNoiseDetected = true
		noiseType = "CRITICAL_ANOMALY"
	} else if buffer.isMotionArtifact(data.Value) {
		// 3. Проверка на артефакты движения (только если значение в принципе валидно)
		data.Value = -1
		isNoiseDetected = true
		noiseType = "MOTION_ARTIFACT"
	}

	// Добавляем значение в буфер (функция сама проверит, не -1 ли это)
	buffer.addValue(data.Value)

	// Вывод с цветовой индикацией
	switch data.DataType {
	case "fetal_heart_rate":
		if isNoiseDetected {
			fmt.Printf("BPM 🚨: %.3f, %.2f (было: %.2f, причина: %s)\n",
				data.TimeSec, data.Value, originalValue, noiseType)
		} else {
			quality := "✅"
			// Предупреждения для пограничных значений
			if data.Value > 180 || data.Value < 100 {
				quality = "⚠️"
			}
			fmt.Printf("BPM %s: %.3f, %.2f\n", quality, data.TimeSec, data.Value)
		}

	case "uterine_contractions":
		if isNoiseDetected {
			fmt.Printf("UTERUS 🚨: %.3f, %.2f (было: %.2f, причина: %s)\n",
				data.TimeSec, data.Value, originalValue, noiseType)
		} else {
			quality := "✅"
			// Предупреждения для высоких значений
			if data.Value > 80 {
				quality = "⚠️"
			}
			fmt.Printf("UTERUS %s: %.3f, %.2f\n", quality, data.TimeSec, data.Value)
		}

	default:
		fmt.Printf("UNKNOWN TYPE: %s - %.3f, %.2f\n", data.DataType, data.TimeSec, data.Value)
	}

	// ⭐ ГЛАВНОЕ: Отправляем ТОЛЬКО БАЗОВЫЕ данные в gRPC стрим
	if grpcStreamServer != nil {
		// Формируем МИНИМАЛЬНОЕ gRPC сообщение (только значения)
		grpcData := &pb.CTGDataResponse{
			DeviceId: data.DeviceID,
			DataType: data.DataType,
			Value:    data.Value, // Включая -1!
			TimeSec:  data.TimeSec,
		}

		// Отправляем в gRPC стрим
		grpcStreamServer.BroadcastCTGData(grpcData)
	}

	// Логируем только действительно критические случаи
	if isNoiseDetected && (noiseType == "CRITICAL_ANOMALY" || noiseType == "INVALID_VALUE") {
		log.Printf("🚨 Критический шум: тип=%s, устройство=%s, время=%.3f, значение=%.2f, причина=%s",
			data.DataType, data.DeviceID, data.TimeSec, originalValue, noiseType)
	}
}
