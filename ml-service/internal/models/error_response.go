package models

// ErrorResponse стандартная структура ошибки
type ErrorResponse struct {
	Error   string `json:"error" example:"validation error"`                    // Сообщение об ошибке
	Details string `json:"details,omitempty" example:"field validation failed"` //Дополнительные детали
}
