package models

import (
	"github.com/google/uuid"
)

type Card struct {
	ID           uuid.UUID     `json:"id" gorm:"type:uuid;primaryKey"`
	Examinations []Examination `json:"examinations" gorm:"serializer:json;type:jsonb"`
}

type Examination struct {
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`

	// КТГ данные - сериализуем в JSON для хранения в БД
	BMP    *CTGData `json:"bmp,omitempty" gorm:"serializer:json;type:jsonb"`
	Uterus *CTGData `json:"uterus,omitempty" gorm:"serializer:json;type:jsonb"`

	// Модели прогнозирования
	Model15 string `json:"model_15" gorm:"column:model_15;type:varchar(255)"`
	Model30 string `json:"model_30" gorm:"column:model_30;type:varchar(255)"`
	Model45 string `json:"model_45" gorm:"column:model_45;type:varchar(255)"`
	Model60 string `json:"model_60" gorm:"column:model_60;type:varchar(255)"`
}

// CTGData структура для хранения временных рядов КТГ
type CTGData struct {
	TimePoints []float64 `json:"time_points"` // Временные отметки
	Values     []float64 `json:"values"`      // Значения измерений
}
