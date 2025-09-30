# ML Service на Golang

Исправленный сервис для связи машинного обучения с бэкендом на языке Go.

## Основные изменения

1. **Порядок действий**: Теперь сервис сначала вычисляет фичи, а затем запрашивает ML предсказание
2. **Исправлена модель данных**: Добавлена правильная модель `CTGSession` для работы с базой данных  
3. **Добавлен отдельный endpoint для вычисления фичей**: `/api/v1/ml/features`
4. **Исправлена работа с JSON данными FHR/UC**: Корректный парсинг данных из базы

## Структура проекта

```
ml-service/
├── main.go                 # Точка входа
├── config/
│   └── config.go          # Конфигурация
├── internal/
│   ├── models/            # Модели данных
│   │   ├── ctg_session.go 
│   │   ├── ml_request.go
│   │   └── ml_response.go
│   ├── features/          # Расчет фичей
│   │   ├── calculator.go
│   │   ├── fhr.go
│   │   ├── uc.go
│   │   └── xcorr.go
│   ├── services/          # Бизнес-логика
│   │   ├── ml_service.go
│   │   └── data_service.go
│   ├── handlers/          # HTTP handlers
│   │   └── ml_handler.go
│   └── database/          # База данных
│       └── connection.go
├── pkg/
│   └── utils/
│       └── math.go        # Математические утилиты
├── go.mod
├── Dockerfile
└── README.md
```

## main.go

```go
package main

import (
    "log"
    "net/http"
    
    "ml-service/config"
    "ml-service/internal/database"
    "ml-service/internal/handlers"
    "ml-service/internal/services"
    
    "github.com/gin-gonic/gin"
)

func main() {
    // Загрузка конфигурации
    cfg := config.Load()
    
    // Подключение к БД
    db, err := database.Connect(cfg)
    if err != nil {
        log.Fatalf("Ошибка подключения к БД: %v", err)
    }
    
    // Инициализация сервисов
    dataService := services.NewDataService(db)
    mlService := services.NewMLService(dataService, cfg.ML.ServiceURL)
    
    // Инициализация обработчиков
    mlHandler := handlers.NewMLHandler(mlService)
    
    // Настройка роутера
    router := gin.Default()
    
    // Middleware
    router.Use(gin.Logger())
    router.Use(gin.Recovery())
    
    // API endpoints
    api := router.Group("/api/v1")
    {
        api.POST("/ml/features", mlHandler.CalculateFeatures)
        api.POST("/ml/predict", mlHandler.Predict)
        api.GET("/ml/health", mlHandler.Health)
    }
    
    log.Printf("Запуск ML сервиса на порту %s", cfg.Port)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
```

## internal/models/ctg_session.go

```go
package models

import (
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// CTGSession представляет сессию CTG в базе данных
type CTGSession struct {
    ID        string    `gorm:"type:uuid;primary_key" json:"id"`
    CardID    string    `gorm:"type:uuid;not null;index" json:"card_id"`
    DeviceID  string    `gorm:"not null" json:"device_id"`
    StartTime time.Time `gorm:"not null" json:"start_time"`
    EndTime   *time.Time `json:"end_time"`
    FHRData   string    `gorm:"type:text" json:"fhr_data"`
    UCData    string    `gorm:"type:text" json:"uc_data"`
    Model15   *float64  `json:"model15"`
    Model30   *float64  `json:"model30"`
    Model45   *float64  `json:"model45"`
    Model60   *float64  `json:"model60"`
}

// TableName устанавливает имя таблицы
func (CTGSession) TableName() string {
    return "ctg_sessions"
}

// BeforeCreate устанавливает ID перед созданием
func (s *CTGSession) BeforeCreate(tx *gorm.DB) error {
    if s.ID == "" {
        s.ID = uuid.New().String()
    }
    return nil
}

// DataPoint представляет точку данных с временной меткой и значением
type DataPoint struct {
    T float64 `json:"t"` // время в секундах
    V float64 `json:"v"` // значение
}

// CTGData представляет структуру данных FHR или UC
type CTGData struct {
    Count  int         `json:"count"`
    Points []DataPoint `json:"points"`
}

// GetFHRPoints парсит и возвращает точки FHR данных
func (s *CTGSession) GetFHRPoints() ([]DataPoint, error) {
    if s.FHRData == "" {
        return []DataPoint{}, nil
    }
    
    var ctgData CTGData
    if err := json.Unmarshal([]byte(s.FHRData), &ctgData); err != nil {
        return nil, fmt.Errorf("ошибка парсинга FHR данных: %w", err)
    }
    
    return ctgData.Points, nil
}

// GetUCPoints парсит и возвращает точки UC данных  
func (s *CTGSession) GetUCPoints() ([]DataPoint, error) {
    if s.UCData == "" {
        return []DataPoint{}, nil
    }
    
    var ctgData CTGData
    if err := json.Unmarshal([]byte(s.UCData), &ctgData); err != nil {
        return nil, fmt.Errorf("ошибка парсинга UC данных: %w", err)
    }
    
    return ctgData.Points, nil
}

// GetDurationSeconds возвращает длительность сессии в секундах
func (s *CTGSession) GetDurationSeconds() int {
    if s.EndTime == nil {
        // Если сессия не завершена, вычисляем до текущего времени
        return int(time.Since(s.StartTime).Seconds())
    }
    return int(s.EndTime.Sub(s.StartTime).Seconds())
}
```

## internal/features/calculator.go

```go
package features

import (
    "fmt"
    "ml-service/pkg/utils"
)

// FeatureCalculator отвечает за расчет всех фичей
type FeatureCalculator struct {
    fs float64 // частота дискретизации
}

// NewFeatureCalculator создает новый калькулятор фичей
func NewFeatureCalculator(fs float64) *FeatureCalculator {
    return &FeatureCalculator{fs: fs}
}

// Calculate вычисляет все фичи для заданного окна
func (fc *FeatureCalculator) Calculate(fhr, uc []float64, windowSec int) map[string]float64 {
    windowSize := int(float64(windowSec) * fc.fs)
    
    // Берем данные из конца массива (последние windowSize точек)
    fhrWindow := fc.getLastWindow(fhr, windowSize)
    ucWindow := fc.getLastWindow(uc, windowSize)
    
    features := make(map[string]float64)
    prefix := fmt.Sprintf("f_%ds_", windowSec)
    
    // FHR фичи с проверкой на NaN
    fhrFeats := CalculateFHRFeatures(fhrWindow, fc.fs)
    features[prefix+"fhr_mean"] = utils.SafeFloat(fhrFeats.Mean)
    features[prefix+"fhr_std"] = utils.SafeFloat(fhrFeats.Std)
    features[prefix+"fhr_min"] = utils.SafeFloat(fhrFeats.Min)
    features[prefix+"fhr_max"] = utils.SafeFloat(fhrFeats.Max)
    features[prefix+"fhr_iqr"] = utils.SafeFloat(fhrFeats.IQR)
    features[prefix+"fhr_rmssd"] = utils.SafeFloat(fhrFeats.RMSSD)
    features[prefix+"fhr_abs_dev"] = utils.SafeFloat(fhrFeats.AbsDev)
    features[prefix+"fhr_brady_len"] = utils.SafeFloat(fhrFeats.BradyLen)
    features[prefix+"fhr_tachy_len"] = utils.SafeFloat(fhrFeats.TachyLen)
    features[prefix+"fhr_decel_cnt"] = utils.SafeFloat(float64(fhrFeats.DecelCnt))
    
    // UC фичи с проверкой на NaN
    ucFeats := CalculateUCFeatures(ucWindow, fc.fs)
    features[prefix+"uc_mean"] = utils.SafeFloat(ucFeats.Mean)
    features[prefix+"uc_std"] = utils.SafeFloat(ucFeats.Std)
    features[prefix+"uc_max"] = utils.SafeFloat(ucFeats.Max)
    features[prefix+"uc_iqr"] = utils.SafeFloat(ucFeats.IQR)
    features[prefix+"uc_peak_cnt"] = utils.SafeFloat(float64(ucFeats.PeakCnt))
    features[prefix+"uc_area"] = utils.SafeFloat(ucFeats.Area)
    
    // Кросс-корреляция с проверкой на NaN
    xcorrFeats := CalculateXCorrFeatures(fhrWindow, ucWindow, fc.fs, 60.0)
    features[prefix+"xcorr_maxabs"] = utils.SafeFloat(xcorrFeats.MaxAbs)
    features[prefix+"xcorr_lag"] = utils.SafeFloat(xcorrFeats.Lag)
    
    return features
}

// getLastWindow возвращает последние N точек из массива
func (fc *FeatureCalculator) getLastWindow(data []float64, windowSize int) []float64 {
    if len(data) <= windowSize {
        return data
    }
    return data[len(data)-windowSize:]
}

// CalculateAllFeatures вычисляет фичи для всех доступных окон
func (fc *FeatureCalculator) CalculateAllFeatures(fhr, uc []float64, duration int) map[string]float64 {
    features := make(map[string]float64)
    windows := []int{240, 600, 900}
    
    for _, window := range windows {
        if duration >= window {
            windowFeatures := fc.Calculate(fhr, uc, window)
            for k, v := range windowFeatures {
                features[k] = v
            }
        }
    }
    
    return features
}

// GetAvailableWindows возвращает список доступных окон на основе длительности данных
func (fc *FeatureCalculator) GetAvailableWindows(duration int) []string {
    var windows []string
    
    if duration >= 240 {
        windows = append(windows, "240s")
    }
    if duration >= 600 {
        windows = append(windows, "600s")
    }
    if duration >= 900 {
        windows = append(windows, "900s")
    }
    
    return windows
}
```

## internal/services/ml_service.go

```go
package services

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "ml-service/internal/features"
    "ml-service/internal/models"
    "net/http"
    "time"
)

// MLService отвечает за взаимодействие с ML моделями
type MLService struct {
    dataService *DataService
    calculator  *features.FeatureCalculator
    mlURL       string
    httpClient  *http.Client
}

// NewMLService создает новый ML сервис
func NewMLService(dataService *DataService, mlURL string) *MLService {
    return &MLService{
        dataService: dataService,
        calculator:  features.NewFeatureCalculator(8.0), // 8 Гц
        mlURL:       mlURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// ProcessMLRequest обрабатывает запрос на ML предсказание
func (ms *MLService) ProcessMLRequest(cardID string, targetTime int) (*models.MLResponse, error) {
    // Получить данные пациента
    patientData, err := ms.dataService.GetPatientDataForTime(cardID, targetTime)
    if err != nil {
        return nil, fmt.Errorf("ошибка получения данных: %w", err)
    }

    // Вычислить фичи
    features := ms.calculator.CalculateAllFeatures(
        patientData.FHR, 
        patientData.UC, 
        patientData.Duration,
    )

    // Определить доступные окна
    availableWindows := ms.calculator.GetAvailableWindows(patientData.Duration)

    // Подготовить запрос к ML сервису
    mlRequest := models.MLRequest{
        CardID:           cardID,
        TSec:             targetTime,
        FsHz:             patientData.SampleRate,
        AvailableWindows: availableWindows,
        Features:         features,
    }

    // Отправить запрос к ML сервису
    return ms.callMLService(mlRequest)
}

// callMLService отправляет запрос к внешнему ML сервису
func (ms *MLService) callMLService(request models.MLRequest) (*models.MLResponse, error) {
    // Сериализовать запрос
    requestBody, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("ошибка сериализации запроса: %w", err)
    }

    // Создать HTTP запрос
    url := fmt.Sprintf("%s/infer", ms.mlURL)
    httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
    if err != nil {
        return nil, fmt.Errorf("ошибка создания запроса: %w", err)
    }

    httpReq.Header.Set("Content-Type", "application/json")

    // Выполнить запрос
    resp, err := ms.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("ML сервис вернул ошибку %d: %s", resp.StatusCode, string(body))
    }

    // Прочитать ответ
    responseBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
    }

    // Десериализовать ответ
    var mlResponse models.MLResponse
    if err := json.Unmarshal(responseBody, &mlResponse); err != nil {
        return nil, fmt.Errorf("ошибка десериализации ответа: %w", err)
    }

    return &mlResponse, nil
}

// CalculateFeatures вычисляет только фичи без обращения к ML
func (ms *MLService) CalculateFeatures(cardID string, targetTime int) (*models.MLRequest, error) {
    // Получить данные пациента
    patientData, err := ms.dataService.GetPatientDataForTime(cardID, targetTime)
    if err != nil {
        return nil, fmt.Errorf("ошибка получения данных: %w", err)
    }

    // Вычислить фичи
    features := ms.calculator.CalculateAllFeatures(
        patientData.FHR, 
        patientData.UC, 
        patientData.Duration,
    )

    // Определить доступные окна
    availableWindows := ms.calculator.GetAvailableWindows(patientData.Duration)

    return &models.MLRequest{
        CardID:           cardID,
        TSec:             targetTime,
        FsHz:             patientData.SampleRate,
        AvailableWindows: availableWindows,
        Features:         features,
    }, nil
}
```

## internal/handlers/ml_handler.go

```go
package handlers

import (
    "net/http"
    "time"

    "ml-service/internal/services"
    "github.com/gin-gonic/gin"
)

// MLHandler обрабатывает HTTP запросы для ML
type MLHandler struct {
    mlService *services.MLService
}

// NewMLHandler создает новый обработчик ML запросов
func NewMLHandler(mlService *services.MLService) *MLHandler {
    return &MLHandler{mlService: mlService}
}

// PredictRequest структура запроса на предсказание
type PredictRequest struct {
    CardID     string `json:"card_id" binding:"required"`
    TargetTime int    `json:"target_time" binding:"required"`
}

// FeaturesRequest структура запроса на вычисление фичей
type FeaturesRequest struct {
    CardID     string `json:"card_id" binding:"required"`
    TargetTime int    `json:"target_time" binding:"required"`
}

// Predict обрабатывает запрос на ML предсказание
func (h *MLHandler) Predict(c *gin.Context) {
    var req PredictRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid request",
            "details": err.Error(),
        })
        return
    }

    response, err := h.mlService.ProcessMLRequest(req.CardID, req.TargetTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "ml service error",
            "details": err.Error(),
        })
        return
    }
    c.JSON(http.StatusOK, response)
}

// CalculateFeatures обрабатывает запрос на вычисление фичей
func (h *MLHandler) CalculateFeatures(c *gin.Context) {
    var req FeaturesRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "invalid request",
            "details": err.Error(),
        })
        return
    }

    features, err := h.mlService.CalculateFeatures(req.CardID, req.TargetTime)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "feature calculation error",
            "details": err.Error(),
        })
        return
    }
    c.JSON(http.StatusOK, features)
}

// Health проверяет состояние сервиса
func (h *MLHandler) Health(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":    "healthy",
        "timestamp": time.Now().UTC(),
    })
}
```

## Dockerfile

```dockerfile
# Dockerfile для Go ML Service
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Копируем go mod и sum файлы
COPY go.mod go.sum ./

# Скачиваем зависимости
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Финальная стадия
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Копируем исполняемый файл
COPY --from=builder /app/main .

# Открываем порт
EXPOSE 8052

# Запускаем приложение
CMD ["./main"]
```

## go.mod

```go
module ml-service

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/google/uuid v1.3.0
    gorm.io/driver/postgres v1.5.2
    gorm.io/gorm v1.25.4
)
```

## API Endpoints

### 1. Вычисление фичей
**POST** `/api/v1/ml/features`

Запрос:
```json
{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "target_time": 960
}
```

Ответ:
```json
{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "t_sec": 960,
    "fs_hz": 8,
    "available_windows": ["240s", "600s", "900s"],
    "features": {
        "f_240s_fhr_mean": 140.0,
        "f_240s_fhr_std": 5.0,
        ...
    }
}
```

### 2. ML Предсказание
**POST** `/api/v1/ml/predict`

Запрос:
```json
{
    "card_id": "550e8400-e29b-41d4-a716-446655440000", 
    "target_time": 960
}
```

Ответ:
```json
{
    "ok": true,
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "t_sec": 960,
    "ran": ["trend5", "h15", "h30"],
    "missing": {},
    "notes": [],
    "result": {
        "trend5": {
            "class": "normal",
            "proba": {
                "normal": 0.85,
                "pathological": 0.15
            }
        }
    }
}
```

### 3. Health Check
**GET** `/api/v1/ml/health`

Ответ:
```json
{
    "status": "healthy",
    "timestamp": "2025-09-30T14:57:16.306253Z"
}
```

## Переменные окружения

```env
# Сервис
PORT=8052

# База данных
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=ctg_db

# ML Service
ML_SERVICE_URL=http://localhost:8000
```

## Запуск в Docker

```bash
# Сборка
docker build -t ml-service:latest .

# Запуск
docker run -p 8052:8052 \
    -e DB_HOST=postgres \
    -e DB_USER=postgres \
    -e DB_PASSWORD=password \
    -e DB_NAME=ctg_db \
    -e ML_SERVICE_URL=http://temp-ml:8000 \
    ml-service:latest
```

## Тестирование

```bash
# Проверка здоровья
curl http://localhost:8052/api/v1/ml/health

# Вычисление фичей
curl -X POST http://localhost:8052/api/v1/ml/features \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "550e8400-e29b-41d4-a716-446655440000",
    "target_time": 960
  }'

# ML предсказание
curl -X POST http://localhost:8052/api/v1/ml/predict \
  -H "Content-Type: application/json" \
  -d '{
    "card_id": "550e8400-e29b-41d4-a716-446655440000", 
    "target_time": 960
  }'
```