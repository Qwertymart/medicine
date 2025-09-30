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
