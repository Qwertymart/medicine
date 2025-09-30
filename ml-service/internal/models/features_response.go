package models

// FeaturesResponse структура ответа с вычисленными фичами
type FeaturesResponse struct {
	CardID           string             `json:"card_id" example:"550e8400-e29b-41d4-a716-446655440000"` // ID карты пациента
	TSec             int                `json:"t_sec" example:"960"`                                    // Время анализа в секундах
	FsHz             float64            `json:"fs_hz" example:"8.0"`                                    // Частота дискретизации
	AvailableWindows []string           `json:"available_windows" example:"240s,600s,900s"`             // Доступные временные окна
	Features         map[string]float64 `json:"features"`                                               // Словарь вычисленных фичей
}
