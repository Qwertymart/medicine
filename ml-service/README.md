У меня есть сервис он делает не совсем что мне нужно. Нужно чтобы я сначала должен у себя найти features а уже потом запросить ml_predict, вот такой порядок действий

кароче делая ml_predict фичи должны посчитать


Как считаются фичи:
Что именно считать (feat-спека)
Окна
240 с (4 мин) → префикс f_240s_*
600 с (10 мин) → префикс f_600s_*
900 с (15 мин) → префикс f_900s_*
Наборы признаков по моделям (ровно такие имена столбцов)
Тренд +5 мин (trend5)
Обновлённая «онлайн»-версия тренда использует только f_240s_* (без каких-либо *_future_*). Список из meta (для ориентира имён) .
Риск на +15 мин (h15) — строго f_600s_*:
mean, std, min, max, iqr, rmssd, abs_dev, brady_len, tachy_len по FHR;
mean, std, max, iqr, peak_cnt, area по UC;
fhr_decel_cnt; xcorr_maxabs, xcorr_lag. Полный список имён в meta .
Риски на +30/+45/+60 мин (h30/h45/h60) — строго f_900s_*, тот же состав метрик и имён, что и выше, только префикс 900s. Списки в meta: h30 , h45 , h60 .
В meta также лежат «пороги» (threshold) для принятия бинарного решения по вероятности — они подбирались на валидации и сохранены рядом с фич-листом (см. поля threshold) .
Как считать каждую метрику (синхронные ряды fhr и uc, 8 Гц)
Для каждого окна (240/600/900 с) рассчитываем одно и то же:
FHR
mean, std, min, max — обычные по окну.
iqr — P75−P25 на окне.
rmssd — √(mean(diff(FHR)²)) на окне.
abs_dev — mean(|FHR − median(FHR)|).
brady_len — суммарная длительность (в сек) участков FHR < 110.
tachy_len — суммарная длительность (в сек) участков FHR > 160.
fhr_decel_cnt — число «децеляций»: подряд ≥ 7.5 с ниже динамического порога median(FHR) − 15.
UC
mean, std, max, iqr.
peak_cnt, area: порог = p10 + 0.5*(p90−p10); минимум длительности пика 5 с;
peak_cnt — число таких эпизодов; area — суммарный интеграл (UC−threshold) по времени (ед·сек).
Связь FHR↔UC
xcorr_maxabs, xcorr_lag: нормированная кросс-корреляция с лагами до ±60 с; берём по модулю максимум и соответствующий лаг (в сек).

Эти определения мы уже применяли при оффлайн-подготовке — важно, чтобы реализация на бэке воспроизводила именно их (чтобы имена и смысл совпали с тем, на чём обучались модели). Требование по real-time потоку и API — в ТЗ кейса, для ориентира цитирую пункты про поток/интеграцию и демо/архитектуру

Псевдокод сгенеренный нейросетью для поиска фичей:
def iqr(x): return np.percentile(x,75)-np.percentile(x,25)

def rmssd(x): d=np.diff(x); return np.sqrt(np.mean(d*d)) if len(d)>0 else np.nan

def run_len_sec(mask, fs):
    total=0; run=0
    for v in mask:
        run = run+1 if v else (total:=total+run, 0)[1]
    if run>0: total += run
    return total/fs

def decel_cnt(fhr, fs):
    thr = np.median(fhr) - 15
    min_len = int(7.5*fs)
    cnt=0; run=0
    for v in fhr:
        run = run+1 if v < thr else (cnt:=cnt+(run>=min_len), 0)[1]
    if run>=min_len: cnt+=1
    return cnt

def uc_peaks_and_area(uc, fs):
    p10, p90 = np.percentile(uc,10), np.percentile(uc,90)
    thr = p10 + 0.5*(p90-p10); min_len=int(5*fs)
    cnt=0; run=0; area=0.0
    for v in uc:
        if v>thr: run+=1; area += (v-thr)
        else: cnt += (run>=min_len); run=0
    if run>=min_len: cnt+=1
    return cnt, area/fs

def xcorr_feats(fhr, uc, fs, max_lag_s=60):
    # z-score
    fx = (fhr - np.mean(fhr)); fy = (uc - np.mean(uc))
    sx = np.std(fx); sy = np.std(fy)
    if sx<1e-6 or sy<1e-6: return np.nan, np.nan
    fx/=sx; fy/=sy
    max_lag = int(max_lag_s*fs)
    best_val, best_lag = -1.0, 0
    for lag in range(-max_lag, max_lag+1):
        a = fx[lag:] if lag>=0 else fx[:lag]
        b = fy[:len(a)] if lag>=0 else fy[-lag:len(fy)]
        if len(a) < 5*fs: continue
        val = np.mean(a*b)
        if abs(val) > abs(best_val): best_val, best_lag = val, lag
    return abs(best_val), best_lag/fs

def build_features(fhr, uc, fs, win_sec):
    fhr = np.asarray(fhr[-int(win_sec*fs):], float)
    uc  = np.asarray(uc [-int(win_sec*fs):], float)
    row={}
    # FHR
    row[f"f_{win_sec}s_fhr_mean"]=np.mean(fhr)
    row[f"f_{win_sec}s_fhr_std"] =np.std(fhr)
    row[f"f_{win_sec}s_fhr_min"] =np.min(fhr)
    row[f"f_{win_sec}s_fhr_max"] =np.max(fhr)
    row[f"f_{win_sec}s_fhr_iqr"] =iqr(fhr)
    row[f"f_{win_sec}s_fhr_rmssd"]=rmssd(fhr)
    row[f"f_{win_sec}s_fhr_abs_dev"]=np.mean(np.abs(fhr-np.median(fhr)))
    row[f"f_{win_sec}s_fhr_brady_len"]=run_len_sec(fhr<110, fs)
    row[f"f_{win_sec}s_fhr_tachy_len"]=run_len_sec(fhr>160, fs)
    row[f"f_{win_sec}s_fhr_decel_cnt"]=decel_cnt(fhr, fs)
    # UC
    row[f"f_{win_sec}s_uc_mean"]=np.mean(uc)
    row[f"f_{win_sec}s_uc_std"] =np.std(uc)
    row[f"f_{win_sec}s_uc_max"] =np.max(uc)
    row[f"f_{win_sec}s_uc_iqr"] =iqr(uc)
    pk, area = uc_peaks_and_area(uc, fs)
    row[f"f_{win_sec}s_uc_peak_cnt"]=pk
    row[f"f_{win_sec}s_uc_area"]=area
    # XCorr
    mx, lag = xcorr_feats(fhr, uc, fs)
    row[f"f_{win_sec}s_xcorr_maxabs"]=mx
    row[f"f_{win_sec}s_xcorr_lag"]=lag
    return row

ПРИМЕР ДЖСОНА (ЛЕЖИТ УЖЕ В ВЕТКЕ РЯДОМ С ПРИЛОЖЕНИЕМ)

Но тут patien_id, в БД по факту за это отвечает card_id

{
  "patient_id": "DEMO",
  "t_sec": 960,
  "fs_hz": 8,
  "available_windows": ["240s","600s","900s"],
  "features": {
    "f_240s_fhr_mean": 140.0, "f_240s_fhr_std": 5.0, "f_240s_fhr_min": 120.0,
    "f_240s_fhr_max": 160.0, "f_240s_fhr_iqr": 8.0, "f_240s_fhr_rmssd": 2.4,
    "f_240s_fhr_abs_dev": 3.2, "f_240s_fhr_brady_len": 0.0, "f_240s_fhr_tachy_len": 12.0,
    "f_240s_uc_mean": 6.0, "f_240s_uc_std": 2.0, "f_240s_uc_max": 18.0, "f_240s_uc_iqr": 3.0,
    "f_240s_uc_peak_cnt": 1, "f_240s_uc_area": 20.0, "f_240s_fhr_decel_cnt": 0,
    "f_240s_xcorr_maxabs": 0.22, "f_240s_xcorr_lag": 5.0,

    "f_600s_fhr_mean": 141.0, "f_600s_fhr_std": 5.5, "f_600s_fhr_min": 118.0,
    "f_600s_fhr_max": 162.0, "f_600s_fhr_iqr": 9.0, "f_600s_fhr_rmssd": 2.7,
    "f_600s_fhr_abs_dev": 3.3, "f_600s_fhr_brady_len": 0.0, "f_600s_fhr_tachy_len": 30.0,
    "f_600s_uc_mean": 6.5, "f_600s_uc_std": 2.3, "f_600s_uc_max": 19.0, "f_600s_uc_iqr": 3.2,
    "f_600s_uc_peak_cnt": 2, "f_600s_uc_area": 46.0, "f_600s_fhr_decel_cnt": 0,
    "f_600s_xcorr_maxabs": 0.25, "f_600s_xcorr_lag": 6.0,

    "f_900s_fhr_mean": 142.0, "f_900s_fhr_std": 6.2, "f_900s_fhr_min": 116.0,
    "f_900s_fhr_max": 164.0, "f_900s_fhr_iqr": 9.5, "f_900s_fhr_rmssd": 2.9,
    "f_900s_fhr_abs_dev": 3.6, "f_900s_fhr_brady_len": 0.0, "f_900s_fhr_tachy_len": 45.0,
    "f_900s_uc_mean": 7.0, "f_900s_uc_std": 2.5, "f_900s_uc_max": 20.0, "f_900s_uc_iqr": 3.4,
    "f_900s_uc_peak_cnt": 3, "f_900s_uc_area": 70.0, "f_900s_fhr_decel_cnt": 0,
    "f_900s_xcorr_maxabs": 0.28, "f_900s_xcorr_lag": 7.0
  }
}


ДАМП БД скинул в csv файле


Также напиши инструкции как пользоваться сервисом в отдельный README.md

Жду от тебя полностью исправленный сервис на golang


# ML Service на Go


Полный сервис для связи машинного обучения с бэкендом на языке Go.


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
└── go.mod
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


## config/config.go


```go
package config


import (
    "os"
)


type Config struct {
    Port     string
    Database DatabaseConfig
    ML       MLConfig
}


type DatabaseConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    SSLMode  string
}


type MLConfig struct {
    ServiceURL string
    Timeout    int
}


func Load() *Config {
    return &Config{
        Port: "8052",
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnv("DB_PORT", "5432"),
            User:     getEnv("DB_USER", "postgres"),
            Password: getEnv("DB_PASSWORD", ""),
            DBName:   getEnv("DB_NAME", "ctg_db"),
            SSLMode:  getEnv("DB_SSL_MODE", "disable"),
        },
        ML: MLConfig{
            ServiceURL: getEnv("ML_SERVICE_URL", "http://localhost:8000"),
            Timeout:    30,
        },
    }
}


func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}


```


## internal/models/ml_request.go
package models


// import "encoding/json"


type MLRequest struct {
    CardID           string                 `json:"card_id"`
    TSec             int                    `json:"t_sec"`
    FsHz             int                    `json:"fs_hz"`
    AvailableWindows []string               `json:"available_windows"`
    Features         map[string]float64     `json:"features"`
}


type MLResponse struct {
    OK        bool                   `json:"ok"`
    CardID    string                 `json:"card_id"`
    TSec      int                    `json:"t_sec"`
    Ran       []string               `json:"ran"`
    Missing   map[string][]string    `json:"missing"`
    Notes     []string               `json:"notes"`
    Result    map[string]interface{} `json:"result"`
}


// TrendResult - результат модели тренда
type TrendResult struct {
    Class string             `json:"class"`
    Proba map[string]float64 `json:"proba"`
}


// RiskResult - результат модели риска
type RiskResult struct {
    Proba float64 `json:"proba"`
    Thr   float64 `json:"thr"`
    Pred  int     `json:"pred"`
}


## internal/models/ctg_session.go


package config


import (
    "os"
)


type Config struct {
    Port     string
    Database DatabaseConfig
    ML       MLConfig
}


type DatabaseConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    SSLMode  string
}


type MLConfig struct {
    ServiceURL string
    Timeout    int
}


func Load() *Config {
    return &Config{
        Port: "8052",
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnv("DB_PORT", "5432"),
            User:     getEnv("DB_USER", "postgres"),
            Password: getEnv("DB_PASSWORD", ""),
            DBName:   getEnv("DB_NAME", "ctg_db"),
            SSLMode:  getEnv("DB_SSL_MODE", "disable"),
        },
        ML: MLConfig{
            ServiceURL: getEnv("ML_SERVICE_URL", "http://localhost:8000"),
            Timeout:    30,
        },
    }
}


func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}


## pkg/utils/math.go


package utils


import (
    "math"
    "sort"
)


func SafeFloat(v float64) float64 {
    if math.IsNaN(v) || math.IsInf(v, 0) {
        return 0.0
    }
    return v
}


// Percentile вычисляет процентиль массива
func Percentile(data []float64, p float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }


    sorted := make([]float64, len(data))
    copy(sorted, data)
    sort.Float64s(sorted)


    if p <= 0 {
        return sorted[0]
    }
    if p >= 100 {
        return sorted[len(sorted)-1]
    }


    n := float64(len(sorted) - 1)
    index := (p / 100.0) * n


    lower := int(math.Floor(index))
    upper := int(math.Ceil(index))


    if lower == upper {
        return sorted[lower]
    }


    weight := index - float64(lower)
    return sorted[lower]*(1-weight) + sorted[upper]*weight
}


// Mean вычисляет среднее значение
func Mean(data []float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }


    sum := 0.0
    for _, v := range data {
        sum += v
    }
    return sum / float64(len(data))
}


// Std вычисляет стандартное отклонение
func Std(data []float64) float64 {
    if len(data) <= 1 {
        return math.NaN()
    }


    mean := Mean(data)
    sumSquares := 0.0


    for _, v := range data {
        diff := v - mean
        sumSquares += diff * diff
    }


    return math.Sqrt(sumSquares / float64(len(data)-1))
}


// Min находит минимальное значение
func Min(data []float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }


    min := data[0]
    for _, v := range data[1:] {
        if v < min {
            min = v
        }
    }
    return min
}


// Max находит максимальное значение
func Max(data []float64) float64 {
    if len(data) == 0 {
        return math.NaN()
    }


    max := data[0]
    for _, v := range data[1:] {
        if v > max {
            max = v
        }
    }
    return max
}


// IQR вычисляет межквартильный размах
func IQR(data []float64) float64 {
    p75 := Percentile(data, 75)
    p25 := Percentile(data, 25)
    return p75 - p25
}


// Abs возвращает абсолютное значение
func Abs(x float64) float64 {
    if x < 0 {
        return -x
    }
    return x
}


// Diff вычисляет разности соседних элементов
func Diff(data []float64) []float64 {
    if len(data) <= 1 {
        return []float64{}
    }


    result := make([]float64, len(data)-1)
    for i := 1; i < len(data); i++ {
        result[i-1] = data[i] - data[i-1]
    }
    return result
}


## internal/features/fhr.go


package features


import (
    "math"
    "ml-service/pkg/utils"
)


// FHRFeatures вычисляет признаки для FHR данных
type FHRFeatures struct {
    Mean     float64 `json:"mean"`
    Std      float64 `json:"std"`
    Min      float64 `json:"min"`
    Max      float64 `json:"max"`
    IQR      float64 `json:"iqr"`
    RMSSD    float64 `json:"rmssd"`
    AbsDev   float64 `json:"abs_dev"`
    BradyLen float64 `json:"brady_len"`
    TachyLen float64 `json:"tachy_len"`
    DecelCnt int     `json:"decel_cnt"`
}


// CalculateFHRFeatures вычисляет все признаки FHR
func CalculateFHRFeatures(fhr []float64, fs float64) FHRFeatures {
    return FHRFeatures{
        Mean:     utils.Mean(fhr),
        Std:      utils.Std(fhr),
        Min:      utils.Min(fhr),
        Max:      utils.Max(fhr),
        IQR:      utils.IQR(fhr),
        RMSSD:    calculateRMSSD(fhr),
        AbsDev:   calculateAbsDev(fhr),
        BradyLen: calculateBradyLen(fhr, fs),
        TachyLen: calculateTachyLen(fhr, fs),
        DecelCnt: calculateDecelCnt(fhr, fs),
    }
}


// calculateRMSSD вычисляет RMSSD (Root Mean Square of Successive Differences)
func calculateRMSSD(fhr []float64) float64 {
    if len(fhr) <= 1 {
        return math.NaN()
    }


    diff := utils.Diff(fhr)
    sumSquares := 0.0


    for _, d := range diff {
        sumSquares += d * d
    }


    if len(diff) == 0 {
        return math.NaN()
    }


    return math.Sqrt(sumSquares / float64(len(diff)))
}


// calculateAbsDev вычисляет среднее абсолютное отклонение от медианы
func calculateAbsDev(fhr []float64) float64 {
    if len(fhr) == 0 {
        return math.NaN()
    }


    median := utils.Percentile(fhr, 50)
    sum := 0.0


    for _, v := range fhr {
        sum += utils.Abs(v - median)
    }


    return sum / float64(len(fhr))
}


// calculateBradyLen вычисляет суммарную длительность брадикардии (FHR < 110)
func calculateBradyLen(fhr []float64, fs float64) float64 {
    return calculateRunLength(fhr, fs, func(v float64) bool {
        return v < 110
    })
}


// calculateTachyLen вычисляет суммарную длительность тахикардии (FHR > 160)
func calculateTachyLen(fhr []float64, fs float64) float64 {
    return calculateRunLength(fhr, fs, func(v float64) bool {
        return v > 160
    })
}


// calculateRunLength вычисляет суммарную длительность состояний
func calculateRunLength(data []float64, fs float64, condition func(float64) bool) float64 {
    total := 0.0
    run := 0.0


    for _, v := range data {
        if condition(v) {
            run += 1.0
        } else {
            total += run
            run = 0.0
        }
    }


    if run > 0 {
        total += run
    }


    return total / fs // конвертируем в секунды
}


// calculateDecelCnt вычисляет количество децелераций
func calculateDecelCnt(fhr []float64, fs float64) int {
    if len(fhr) == 0 {
        return 0
    }


    threshold := utils.Percentile(fhr, 50) - 15
    minLen := int(7.5 * fs) // минимум 7.5 секунд


    count := 0
    run := 0


    for _, v := range fhr {
        if v < threshold {
            run++
        } else {
            if run >= minLen {
                count++
            }
            run = 0
        }
    }


    if run >= minLen {
        count++
    }


    return count
}


## internal/features/uc.go


package features


import (
    "ml-service/pkg/utils"
)


// UCFeatures вычисляет признаки для UC данных
type UCFeatures struct {
    Mean     float64 `json:"mean"`
    Std      float64 `json:"std"`
    Max      float64 `json:"max"`
    IQR      float64 `json:"iqr"`
    PeakCnt  int     `json:"peak_cnt"`
    Area     float64 `json:"area"`
}


// CalculateUCFeatures вычисляет все признаки UC
func CalculateUCFeatures(uc []float64, fs float64) UCFeatures {
    peakCnt, area := calculateUCPeaksAndArea(uc, fs)
    
    return UCFeatures{
        Mean:    utils.Mean(uc),
        Std:     utils.Std(uc),
        Max:     utils.Max(uc),
        IQR:     utils.IQR(uc),
        PeakCnt: peakCnt,
        Area:    area,
    }
}


// calculateUCPeaksAndArea вычисляет количество пиков и площадь под кривой UC
func calculateUCPeaksAndArea(uc []float64, fs float64) (int, float64) {
    if len(uc) == 0 {
        return 0, 0.0
    }
    
    p10 := utils.Percentile(uc, 10)
    p90 := utils.Percentile(uc, 90)
    threshold := p10 + 0.5*(p90-p10)
    minLen := int(5 * fs) // минимум 5 секунд
    
    count := 0
    run := 0
    area := 0.0
    
    for _, v := range uc {
        if v > threshold {
            run++
            area += (v - threshold)
        } else {
            if run >= minLen {
                count++
            }
            run = 0
        }
    }
    
    if run >= minLen {
        count++
    }
    
    return count, area / fs // конвертируем в секунды
}
## internal/features/xcorr.go


package features


import (
    "math"
    "ml-service/pkg/utils"
)


// XCorrFeatures вычисляет признаки кросс-корреляции FHR и UC
type XCorrFeatures struct {
    MaxAbs float64 `json:"maxabs"`
    Lag    float64 `json:"lag"`
}


// CalculateXCorrFeatures вычисляет признаки кросс-корреляции
func CalculateXCorrFeatures(fhr, uc []float64, fs float64, maxLagS float64) XCorrFeatures {
    if len(fhr) == 0 || len(uc) == 0 {
        return XCorrFeatures{
            MaxAbs: math.NaN(),
            Lag:    math.NaN(),
        }
    }
    
    // Z-score нормализация
    fhrMean := utils.Mean(fhr)
    fhrStd := utils.Std(fhr)
    ucMean := utils.Mean(uc)
    ucStd := utils.Std(uc)
    
    if fhrStd < 1e-6 || ucStd < 1e-6 {
        return XCorrFeatures{
            MaxAbs: math.NaN(),
            Lag:    math.NaN(),
        }
    }
    
    // Нормализация
    fhrNorm := make([]float64, len(fhr))
    ucNorm := make([]float64, len(uc))
    
    for i, v := range fhr {
        fhrNorm[i] = (v - fhrMean) / fhrStd
    }
    
    for i, v := range uc {
        ucNorm[i] = (v - ucMean) / ucStd
    }
    
    maxLag := int(maxLagS * fs)
    bestVal := -1.0
    bestLag := 0
    
    // Поиск максимальной корреляции по лагам
    for lag := -maxLag; lag <= maxLag; lag++ {
        var a, b []float64
        
        if lag >= 0 {
            // fhr смещен вперед
            if lag < len(fhrNorm) {
                a = fhrNorm[lag:]
                minLen := int(math.Min(float64(len(a)), float64(len(ucNorm))))
                if minLen > 0 {
                    a = a[:minLen]
                    b = ucNorm[:minLen]
                }
            }
        } else {
            // uc смещен вперед
            absLag := -lag
            if absLag < len(ucNorm) {
                b = ucNorm[absLag:]
                minLen := int(math.Min(float64(len(fhrNorm)), float64(len(b))))
                if minLen > 0 {
                    a = fhrNorm[:minLen]
                    b = b[:minLen]
                }
            }
        }
        
        if len(a) < int(5*fs) { // минимум 5 секунд данных
            continue
        }
        
        // Вычисляем корреляцию
        corr := 0.0
        for i := 0; i < len(a); i++ {
            corr += a[i] * b[i]
        }
        corr /= float64(len(a))
        
        if utils.Abs(corr) > utils.Abs(bestVal) {
            bestVal = corr
            bestLag = lag
        }
    }
    
    return XCorrFeatures{
        MaxAbs: utils.Abs(bestVal),
        Lag:    float64(bestLag) / fs,
    }
}
## internal/features/calculator.go


package features


import (
    "fmt"
    "ml-service/pkg/utils"
    _ "ml-service/internal/models"
)


// FeatureCalculator отвечает за расчет всех фичей
type FeatureCalculator struct {
    fs float64 // частота дискретизации
}


// NewFeatureCalculator создает новый калькулятор фичей
func NewFeatureCalculator(fs float64) *FeatureCalculator {
    return &FeatureCalculator{fs: fs}
}


// Calculate вычисляет все фичей для заданного окна
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
## internal/services/data_service.go


package services


import (
    "fmt"
    "log"
    "ml-service/internal/models"


    "gorm.io/gorm"
)


// DataService отвечает за работу с данными
type DataService struct {
    db *gorm.DB
}


// NewDataService создает новый сервис для работы с данными
func NewDataService(db *gorm.DB) *DataService {
    return &DataService{db: db}
}


// GetPatientDataForTime получает данные пациента для заданного времени
func (ds *DataService) GetPatientDataForTime(cardID string, targetTime int) (*PatientData, error) {
    log.Printf("Поиск данных для пациента %s на время %d секунд", cardID, targetTime)
    
    // Найти сессии пациента
    var sessions []models.CTGSession
    err := ds.db.Where("card_id = ?", cardID).
        Order("start_time ASC").
        Find(&sessions).Error


    if err != nil {
        return nil, fmt.Errorf("ошибка получения сессий: %w", err)
    }


    if len(sessions) == 0 {
        return nil, fmt.Errorf("сессии для пациента %s не найдены", cardID)
    }


    log.Printf("Найдено сессий: %d", len(sessions))


    // Объединить данные из всех сессий
    allFHR := []float64{}
    allUC := []float64{}
    totalTime := 0


    for sessionIdx, session := range sessions {
        log.Printf("Обрабатываем сессию %d (ID: %s)", sessionIdx+1, session.ID)
        
        // Получить данные FHR
        fhrPoints, err := session.GetFHRPoints()
        if err != nil {
            log.Printf("Ошибка парсинга FHR для сессии %s: %v", session.ID, err)
            continue
        }
        log.Printf("FHR точек получено: %d", len(fhrPoints))


        // Получить данные UC
        ucPoints, err := session.GetUCPoints()
        if err != nil {
            log.Printf("Ошибка парсинга UC для сессии %s: %v", session.ID, err)
            continue
        }
        log.Printf("UC точек получено: %d", len(ucPoints))


        // Логируем первые несколько точек для диагностики
        if len(fhrPoints) > 0 {
            log.Printf("Первые 3 FHR точки:")
            for i := 0; i < min(3, len(fhrPoints)); i++ {
                log.Printf("  FHR[%d]: t=%.3f, v=%.3f", i, fhrPoints[i].T, fhrPoints[i].V)
            }
        }


        if len(ucPoints) > 0 {
            log.Printf("Первые 3 UC точки:")
            for i := 0; i < min(3, len(ucPoints)); i++ {
                log.Printf("  UC[%d]: t=%.3f, v=%.3f", i, ucPoints[i].T, ucPoints[i].V)
            }
        }


        // Конвертировать в массивы значений
        sessionFHR := make([]float64, 0)
        sessionUC := make([]float64, 0)


        // Фильтруем FHR данные (исключаем -1)
        for _, point := range fhrPoints {
            if point.V != -1.0 {
                sessionFHR = append(sessionFHR, point.V)
            }
        }


        // Фильтруем UC данные (исключаем -1)
        for _, point := range ucPoints {
            if point.V != -1.0 {
                sessionUC = append(sessionUC, point.V)
            }
        }


        log.Printf("Сессия %d: FHR отфильтровано %d из %d, UC отфильтровано %d из %d", 
            sessionIdx+1, len(sessionFHR), len(fhrPoints), len(sessionUC), len(ucPoints))


        // Добавляем к общим массивам
        allFHRBefore := len(allFHR)
        allUCBefore := len(allUC)
        
        allFHR = append(allFHR, sessionFHR...)
        allUC = append(allUC, sessionUC...)
        
        log.Printf("Общие массивы: FHR %d -> %d (+%d), UC %d -> %d (+%d)", 
            allFHRBefore, len(allFHR), len(sessionFHR),
            allUCBefore, len(allUC), len(sessionUC))


        sessionDuration := session.GetDurationSeconds()
        totalTime += sessionDuration
        log.Printf("Длительность сессии: %d сек, общая длительность: %d сек", sessionDuration, totalTime)


        // Проверить, достигли ли нужного времени
        // Проверить, достигли ли нужного времени
        if totalTime >= targetTime {
            log.Printf("Достигнуто целевое время %d сек, прерываем обработку", targetTime)
            
            // Фиксированная частота
            const sampleRate = 8
            
            // Обрезка до нужного числа семплов
            samplesNeeded := targetTime * sampleRate
            log.Printf("Нужно семплов для %d сек при %d Гц: %d", targetTime, sampleRate, samplesNeeded)
            
            if len(allFHR) > samplesNeeded {
                allFHR = allFHR[:samplesNeeded]
                log.Printf("FHR обрезан до %d семплов", len(allFHR))
            }
            if len(allUC) > samplesNeeded {
                allUC = allUC[:samplesNeeded]
                log.Printf("UC обрезан до %d семплов", len(allUC))
            }
            break
        }


    }
    
    log.Printf("Итого FHR: %d, UC: %d, Duration: %d", len(allFHR), len(allUC), totalTime)


    return &PatientData{
        CardID:     cardID,
        FHR:        allFHR,
        UC:         allUC,
        Duration:   totalTime,
        SampleRate: 8, // По умолчанию 8 Гц
    }, nil
}


// min возвращает минимальное из двух целых чисел
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}


// PatientData содержит данные пациента
type PatientData struct {
    CardID     string    `json:"card_id"`
    FHR        []float64 `json:"fhr"`
    UC         []float64 `json:"uc"`
    Duration   int       `json:"duration"`
    SampleRate int       `json:"sample_rate"`
}


## internal/services/ml_service.go


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


## internal/handlers/ml_handler.go


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


## internal/database/connection.go


package database


import (
    "fmt"
    "ml-service/config"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)


// Connect подключается к базе данных
func Connect(cfg *config.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
        cfg.Database.Host,
        cfg.Database.User, 
        cfg.Database.Password,
        cfg.Database.DBName,
        cfg.Database.Port,
    )
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    // Настроить connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    
    return db, nil
}
## go.mod
module ml-service


go 1.24.2


require (
    github.com/gin-gonic/gin v1.11.0
    github.com/google/uuid v1.6.0
    gorm.io/driver/postgres v1.6.0
    gorm.io/gorm v1.31.0
)


require (
    github.com/bytedance/sonic v1.14.0 // indirect
    github.com/bytedance/sonic/loader v0.3.0 // indirect
    github.com/cloudwego/base64x v0.1.6 // indirect
    github.com/gabriel-vasile/mimetype v1.4.8 // indirect
    github.com/gin-contrib/sse v1.1.0 // indirect
    github.com/go-playground/locales v0.14.1 // indirect
    github.com/go-playground/universal-translator v0.18.1 // indirect
    github.com/go-playground/validator/v10 v10.27.0 // indirect
    github.com/goccy/go-json v0.10.2 // indirect
    github.com/goccy/go-yaml v1.18.0 // indirect
    github.com/jackc/pgpassfile v1.0.0 // indirect
    github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
    github.com/jackc/pgx/v5 v5.6.0 // indirect
    github.com/jackc/puddle/v2 v2.2.2 // indirect
    github.com/jinzhu/inflection v1.0.0 // indirect
    github.com/jinzhu/now v1.1.5 // indirect
    github.com/json-iterator/go v1.1.12 // indirect
    github.com/klauspost/cpuid/v2 v2.3.0 // indirect
    github.com/leodido/go-urn v1.4.0 // indirect
    github.com/mattn/go-isatty v0.0.20 // indirect
    github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
    github.com/modern-go/reflect2 v1.0.2 // indirect
    github.com/pelletier/go-toml/v2 v2.2.4 // indirect
    github.com/quic-go/qpack v0.5.1 // indirect
    github.com/quic-go/quic-go v0.54.0 // indirect
    github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
    github.com/ugorji/go/codec v1.3.0 // indirect
    go.uber.org/mock v0.5.0 // indirect
    golang.org/x/arch v0.20.0 // indirect
    golang.org/x/crypto v0.40.0 // indirect
    golang.org/x/mod v0.25.0 // indirect
    golang.org/x/net v0.42.0 // indirect
    golang.org/x/sync v0.16.0 // indirect
    golang.org/x/sys v0.35.0 // indirect
    golang.org/x/text v0.27.0 // indirect
    golang.org/x/tools v0.34.0 // indirect
    google.golang.org/protobuf v1.36.9 // indirect
)


## Dockerfile


# Dockerfile для Python ML сервиса (temp_ml)


FROM python:3.11-slim


# Установка системных зависимостей
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*


# Установка рабочей директории
WORKDIR /app


# Копирование файлов requirements
COPY requirements.txt .


# Установка Python зависимостей
RUN pip install --no-cache-dir -r requirements.txt


# Копирование исходного кода
COPY . .


# Создание директории для моделей
RUN mkdir -p /app/out


# Установка переменных окружения
ENV PYTHONPATH=/app
ENV OUT_DIR=/app/out
ENV PORT=8000


# Открытие порта
EXPOSE 8000


# Команда запуска
CMD ["python", "app.py"]


## docker-compose.yml


version: '3.8'


services:
  postgres:
    image: postgres:15-alpine
    container_name: ctg_postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks: [ctg-network]


  mosquitto:
    image: eclipse-mosquitto:2.0
    container_name: ctg_mosquitto
    restart: unless-stopped
    ports:
      - "1883:1883"
      - "9001:9001"
    volumes:
      - ./mosquitto/mosquitto.conf:/mosquitto/config/mosquitto.conf
      - ./mosquitto/data:/mosquitto/data
      - ./mosquitto/log:/mosquitto/log
    networks: [ctg-network]


  ctg-emulator:
    build: { context: ./medicine_emulator , dockerfile: Dockerfile }
    image: ctg-emulator:latest
    container_name: ctg_emulator_service
    restart: unless-stopped
    ports:
      - "8081:8081"
    volumes:
      - ./medicine_emulator/data:/app/data
    depends_on: [mosquitto]
    environment:
      - MQTT_BROKER=tcp://mosquitto:1883
      - MQTT_HOST=mosquitto
      - MQTT_PORT=1883
      - MQTT_CLIENT_ID=ctg_emulator
      - MQTT_USERNAME=ctg_mqtt_user
      - MQTT_PASSWORD=ctg_mqtt_password
    extra_hosts:
      - "host.docker.internal:host-gateway"
    networks: [ctg-network]


  ctg-monitor:
    build: { context: ./CTG_monitor , dockerfile: Dockerfile }
    image: ctg-monitor:latest
    container_name: ctg_monitor_service
    restart: unless-stopped
    ports:
      - "8080:8080"
      - "50051:50051"
    depends_on: [postgres, mosquitto, ml-service]
    environment:
      # DB
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - DB_SSLMODE=disable
      - DB_TIMEZONE=Europe/Moscow
      # MQTT
      - MQTT_BROKER=tcp://mosquitto:1883
      - MQTT_HOST=mosquitto
      - MQTT_PORT=1883
      - MQTT_CLIENT_ID=ctg_monitor_service
      - MQTT_USERNAME=ctg_mqtt_user
      - MQTT_PASSWORD=ctg_mqtt_password
      - MQTT_QOS=1
      # App
      - HTTP_PORT=8080
      - GRPC_PORT=50051
      - LOG_LEVEL=info
      - BUFFER_SIZE=100
      - BUFFER_FLUSH_INTERVAL=10
      - MAX_BUFFER_SIZE=1000
    healthcheck:
      test: ["CMD","wget","--no-verbose","--tries=1","--spider","http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks: [ctg-network]


  temp-ml:
    build: { context: ./temp_ml , dockerfile: Dockerfile }
    image: temp-ml:latest
    container_name: temp_ml_service
    restart: unless-stopped
    ports:
      - "8000:8000"
    volumes:
      - ./temp_ml/out:/app/out
    environment:
      - OUT_DIR=/app/out
      - PORT=8000
    networks: [ctg-network]


  ml-service:
    build: { context: ./ml-service , dockerfile: Dockerfile }
    image: ctg-ml:latest
    container_name: ctg_ml_service
    restart: unless-stopped
    ports:
      - "8052:8052"
    depends_on: [postgres, temp-ml]
    environment:
      - PORT=8082
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=${DB_NAME}
      - ML_SERVICE_URL=http://temp-ml:8000  # внутреннее DNS-имя
    networks: [ctg-network]


networks:
  ctg-network:
    driver: bridge


volumes:
  postgres_data:


