package models

import (
    "database/sql"
    "database/sql/driver"
    "encoding/json"
    "fmt"
    "strconv"
    "time"
    
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// NullFloat64 для обработки пустых строк в float64 полях
type NullFloat64 struct {
    sql.NullFloat64
}

// Scan реализует интерфейс Scanner для обработки пустых строк
func (nf *NullFloat64) Scan(value interface{}) error {
    if value == nil {
        nf.NullFloat64.Float64, nf.NullFloat64.Valid = 0.0, false
        return nil
    }
    
    switch v := value.(type) {
    case float64:
        nf.NullFloat64.Float64, nf.NullFloat64.Valid = v, true
        return nil
    case string:
        if v == "" {
            nf.NullFloat64.Float64, nf.NullFloat64.Valid = 0.0, false
            return nil
        }
        f, err := strconv.ParseFloat(v, 64)
        if err != nil {
            nf.NullFloat64.Float64, nf.NullFloat64.Valid = 0.0, false
            return nil
        }
        nf.NullFloat64.Float64, nf.NullFloat64.Valid = f, true
        return nil
    case []byte:
        if len(v) == 0 {
            nf.NullFloat64.Float64, nf.NullFloat64.Valid = 0.0, false
            return nil
        }
        f, err := strconv.ParseFloat(string(v), 64)
        if err != nil {
            nf.NullFloat64.Float64, nf.NullFloat64.Valid = 0.0, false
            return nil
        }
        nf.NullFloat64.Float64, nf.NullFloat64.Valid = f, true
        return nil
    }
    
    return fmt.Errorf("не удается конвертировать %T в NullFloat64", value)
}

// Value реализует интерфейс Valuer
func (nf NullFloat64) Value() (driver.Value, error) {
    if !nf.Valid {
        return nil, nil
    }
    return nf.Float64, nil
}

// MarshalJSON для корректной сериализации в JSON
func (nf NullFloat64) MarshalJSON() ([]byte, error) {
    if !nf.Valid {
        return []byte("null"), nil
    }
    return json.Marshal(nf.Float64)
}

// CTGSession представляет сессию CTG в базе данных
type CTGSession struct {
    ID        string      `gorm:"type:uuid;primary_key" json:"id"`
    CardID    string      `gorm:"type:uuid;not null;index" json:"card_id"`
    DeviceID  string      `gorm:"not null" json:"device_id"`
    StartTime time.Time   `gorm:"not null" json:"start_time"`
    EndTime   *time.Time  `json:"end_time"`
    FHRData   string      `gorm:"type:text" json:"fhr_data"`
    UCData    string      `gorm:"type:text" json:"uc_data"`
    Model15   NullFloat64 `json:"model15"`
    Model30   NullFloat64 `json:"model30"`
    Model45   NullFloat64 `json:"model45"`
    Model60   NullFloat64 `json:"model60"`
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
