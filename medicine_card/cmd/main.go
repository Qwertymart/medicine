// Сервис медицинских карт
// main.go
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	medpb "medicine_card/proto"
)

// MedicalRecordsServer реализует gRPC сервис медкарт
type MedicalRecordsServer struct {
	medpb.UnimplementedMedicalRecordsServiceServer
}

// SaveCTGSession обрабатывает запросы на сохранение сессий КТГ
func (s *MedicalRecordsServer) SaveCTGSession(
	ctx context.Context,
	req *medpb.CTGSessionRequest,
) (*medpb.SaveSessionResponse, error) {

	log.Printf("🏥 === ПОЛУЧЕНА СЕССИЯ КТГ ===")
	log.Printf("📋 Session ID: %s", req.SessionId)
	log.Printf("📋 Card ID: %s", req.CardId)
	log.Printf("📋 Device ID: %s", req.DeviceId)
	log.Printf("📋 Start Time: %s", time.Unix(req.StartTime, 0).Format("2006-01-02 15:04:05"))
	log.Printf("📋 End Time: %s", time.Unix(req.EndTime, 0).Format("2006-01-02 15:04:05"))
	log.Printf("📋 Duration: %d секунд", req.DurationSeconds)
	log.Printf("📋 FHR точек: %d", req.TotalFhrPoints)
	log.Printf("📋 UC точек: %d", req.TotalUcPoints)

	// Выводим первые 5 точек FHR данных для примера
	if len(req.FhrData) > 0 {
		log.Printf("📊 Первые FHR данные:")
		count := min(5, len(req.FhrData))
		for i := 0; i < count; i++ {
			point := req.FhrData[i]
			log.Printf("   FHR[%d]: время=%.2fs, значение=%.2f",
				i, point.TimeSec, point.Value)
		}
		if len(req.FhrData) > 5 {
			log.Printf("   ... и еще %d точек FHR", len(req.FhrData)-5)
		}
	}

	// Выводим первые 5 точек UC данных для примера
	if len(req.UcData) > 0 {
		log.Printf("📊 Первые UC данные:")
		count := min(5, len(req.UcData))
		for i := 0; i < count; i++ {
			point := req.UcData[i]
			log.Printf("   UC[%d]: время=%.2fs, значение=%.2f",
				i, point.TimeSec, point.Value)
		}
		if len(req.UcData) > 5 {
			log.Printf("   ... и еще %d точек UC", len(req.UcData)-5)
		}
	}

	// Статистика по данным
	if len(req.FhrData) > 0 {
		var fhrSum, fhrMin, fhrMax float64
		fhrMin = req.FhrData[0].Value
		fhrMax = req.FhrData[0].Value

		for _, point := range req.FhrData {
			fhrSum += point.Value
			if point.Value < fhrMin {
				fhrMin = point.Value
			}
			if point.Value > fhrMax {
				fhrMax = point.Value
			}
		}

		fhrAvg := fhrSum / float64(len(req.FhrData))
		log.Printf("📈 FHR статистика: мин=%.2f, макс=%.2f, среднее=%.2f",
			fhrMin, fhrMax, fhrAvg)
	}

	if len(req.UcData) > 0 {
		var ucSum, ucMin, ucMax float64
		ucMin = req.UcData[0].Value
		ucMax = req.UcData[0].Value

		for _, point := range req.UcData {
			ucSum += point.Value
			if point.Value < ucMin {
				ucMin = point.Value
			}
			if point.Value > ucMax {
				ucMax = point.Value
			}
		}

		ucAvg := ucSum / float64(len(req.UcData))
		log.Printf("📈 UC статистика: мин=%.2f, макс=%.2f, среднее=%.2f",
			ucMin, ucMax, ucAvg)
	}

	// Имитируем сохранение в БД (здесь можно добавить реальное сохранение)
	recordID := "MED_" + req.SessionId[:8] + "_" + time.Now().Format("20060102150405")

	log.Printf("💾 Сессия сохранена с ID: %s", recordID)
	log.Printf("✅ === ОБРАБОТКА ЗАВЕРШЕНА ===\n")

	// Возвращаем успешный ответ
	return &medpb.SaveSessionResponse{
		Success:  true,
		Message:  "Сессия КТГ успешно сохранена в медицинские карты",
		RecordId: recordID,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	log.Println("🏥 === СЕРВИС МЕДИЦИНСКИХ КАРТ ===")
	log.Println("🚀 Запуск gRPC сервера на порту 50052...")

	// Создаем TCP listener
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("❌ Ошибка создания listener: %v", err)
	}

	// Создаем gRPC сервер
	grpcServer := grpc.NewServer()

	// Регистрируем сервис медкарт
	medicalServer := &MedicalRecordsServer{}
	medpb.RegisterMedicalRecordsServiceServer(grpcServer, medicalServer)

	log.Printf("🎯 gRPC сервер медкарт слушает порт :50052")
	log.Println("📝 Готов принимать сессии КТГ от CTG Monitor...")
	log.Println("🛑 Ctrl+C для остановки")

	// Запускаем сервер в отдельной горутине
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ Ошибка запуска gRPC сервера: %v", err)
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("🛑 Получен сигнал остановки...")
	grpcServer.GracefulStop()
	log.Println("✅ Сервис медицинских карт остановлен")
}
