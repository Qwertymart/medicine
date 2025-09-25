package models

import (
	"github.com/google/uuid"
	"time"
)

// CTGSession единая таблица для всех КТГ данных
type CTGSession struct {
	// Основные идентификаторы
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CardID   uuid.UUID `json:"card_id" gorm:"type:uuid;not null;index"`
	DeviceID string    `json:"device_id" gorm:"type:varchar(100);not null;index"`

	// Метаданные сессии
	StartTime time.Time  `json:"start_time" gorm:"not null;index"`
	EndTime   *time.Time `json:"end_time" gorm:"index"` // null пока сессия активна

	// 🔥 КТГ данные как аппендабельные JSONB массивы
	FHRData CTGTimeSeries `json:"fhr_data" gorm:"serializer:json;type:jsonb"` // fetal heart rate
	UCData  CTGTimeSeries `json:"uc_data" gorm:"serializer:json;type:jsonb"`  // uterine contractions

	// Модели прогнозирования
	Model15 string `json:"model_15" gorm:"type:varchar(255)"`
	Model30 string `json:"model_30" gorm:"type:varchar(255)"`
	Model45 string `json:"model_45" gorm:"type:varchar(255)"`
	Model60 string `json:"model_60" gorm:"type:varchar(255)"`
}

// CTGTimeSeries оптимизированная структура для аппенда
type CTGTimeSeries struct {
	Points   []CTGPoint `json:"points"`    // Массив точек данных
	LastTime float64    `json:"last_time"` // Последняя временная отметка
	Count    int        `json:"count"`     // Количество точек
}

// CTGPoint одна точка данных
type CTGPoint struct {
	T float64 `json:"t"` // Время в секундах (компактно)
	V float64 `json:"v"` // Значение
}

func (CTGSession) TableName() string {
	return "ctg_sessions"
}
