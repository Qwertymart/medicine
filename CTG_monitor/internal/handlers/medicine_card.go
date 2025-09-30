package handlers

import (
	"CTG_monitor/internal/database"
	"context"
	"github.com/google/uuid"
	"log"
	"time"

	"CTG_monitor/internal/models"
	medpb "CTG_monitor/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var medicalRecordsClient medpb.MedicalRecordsServiceClient
var grpcConn *grpc.ClientConn

// InitMedicalRecordsClient инициализирует gRPC клиент для медкарт
func InitMedicalRecordsClient(address string) error {
	// Создаем подключение к gRPC серверу медкарт
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	grpcConn = conn
	medicalRecordsClient = medpb.NewMedicalRecordsServiceClient(conn)

	log.Printf("gRPC клиент медкарт инициализирован: %s", address)
	return nil
}

// CloseMedicalRecordsClient закрывает соединение
func CloseMedicalRecordsClient() error {
	if grpcConn != nil {
		return grpcConn.Close()
	}
	return nil
}

// sendSessionToMedicalRecordsGRPC отправляет сессию через gRPC
func sendSessionToMedicalRecordsGRPC(session *models.CTGSession) error {
	if medicalRecordsClient == nil {
		return nil // Клиент не инициализирован
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Преобразуем FHR данные
	var fhrPoints []*medpb.CTGDataPoint
	for _, point := range session.FHRData.Points {
		fhrPoints = append(fhrPoints, &medpb.CTGDataPoint{
			TimeSec: point.T,
			Value:   point.V,
		})
	}

	// Преобразуем UC данные
	var ucPoints []*medpb.CTGDataPoint
	for _, point := range session.UCData.Points {
		ucPoints = append(ucPoints, &medpb.CTGDataPoint{
			TimeSec: point.T,
			Value:   point.V,
		})
	}

	// Вычисляем продолжительность
	var duration int32
	if session.EndTime != nil {
		duration = int32(session.EndTime.Sub(session.StartTime).Seconds())
	}

	// Формируем запрос
	request := &medpb.CTGSessionRequest{
		SessionId:       session.ID.String(),
		CardId:          session.CardID.String(),
		DeviceId:        session.DeviceID,
		StartTime:       session.StartTime.Unix(),
		EndTime:         session.EndTime.Unix(),
		DurationSeconds: duration,
		FhrData:         fhrPoints,
		UcData:          ucPoints,
		TotalFhrPoints:  int32(len(fhrPoints)),
		TotalUcPoints:   int32(len(ucPoints)),
	}

	log.Printf("Отправка сессии %s в медкарты через gRPC: FHR=%d, UC=%d точек",
		session.ID.String(), len(fhrPoints), len(ucPoints))

	// Отправляем запрос
	response, err := medicalRecordsClient.SaveCTGSession(ctx, request)
	if err != nil {
		return err
	}

	if !response.Success {
		log.Printf("Сервис медкарт вернул ошибку: %s", response.Message)
		return nil // Не возвращаем ошибку, просто логируем
	}

	log.Printf("Сессия %s успешно сохранена в медкарты (Record ID: %s)",
		session.ID.String(), response.RecordId)

	return nil
}

func SendSessionToMedicalRecords(sessionID uuid.UUID) {
	log.Printf("Начинаем отправку сессии %s в медкарты", sessionID)

	db := database.GetDB()

	var session models.CTGSession
	if err := db.First(&session, "id = ?", sessionID).Error; err != nil {
		log.Printf("Ошибка получения сессии %s из БД: %v", sessionID, err)
		return
	}
	if session.EndTime == nil {
		log.Printf("Сессия %s ещё не завершена", sessionID)
		return
	}

	// 2. Извлечь точки из JSONB полей
	fhrCount := len(session.FHRData.Points)
	ucCount := len(session.UCData.Points)

	log.Printf("Данные сессии %s: FHR=%d точек, UC=%d точек", sessionID, fhrCount, ucCount)

	// 3. Отправить через существующий gRPC клиент
	if err := sendSessionToMedicalRecordsGRPC(&session); err != nil {
		log.Printf("Ошибка отправки сессии %s в медкарты: %v", sessionID, err)
	} else {
		log.Printf("Сессия %s успешно отправлена в медкарты", sessionID)
	}
}
