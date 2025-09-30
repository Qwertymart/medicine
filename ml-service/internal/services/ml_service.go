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
	url := fmt.Sprintf("%s/infer?verbose=true", ms.mlURL)

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
