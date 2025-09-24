package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "client_grpc/proto" // импорт сгенерированного кода
)

func main() {
	serverAddr := "localhost:50051" // адрес gRPC-сервера
	deviceIDs := []string{}         // [] = все устройства
	dataTypes := []string{          // нужные типы данных
		"fetal_heart_rate",
		"uterine_contractions",
	}

	//----------------------------------------
	// Подключаемся к gRPC-серверу
	//----------------------------------------
	conn, err := grpc.Dial(
		serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("gRPC dial error: %v", err)
	}
	defer conn.Close()

	client := pb.NewCTGStreamServiceClient(conn)

	//----------------------------------------
	// Создаём запрос на стриминг
	//----------------------------------------
	req := &pb.StreamRequest{
		DeviceIds: deviceIDs,
		DataTypes: dataTypes,
	}

	stream, err := client.StreamCTGData(context.Background(), req)
	if err != nil {
		log.Fatalf("stream error: %v", err)
	}

	log.Printf("🟢 Подключён к %s  (Ctrl-C для выхода)", serverAddr)

	//----------------------------------------
	// Обработка входящих данных
	//----------------------------------------
	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				log.Println("stream closed")
				return
			}
			if err != nil {
				log.Printf("recv error: %v", err)
				return
			}

			val := fmt.Sprintf("%.2f", msg.Value)
			if msg.Value == -1 {
				val = "SIGNAL_LOSS"
			}

			fmt.Printf("[%s] %s  %-13s  %s  (t=%.3fs)\n",
				msg.DeviceId,
				rightPad(msg.DataType, 20),
				val,
				"", // для отступов
				msg.TimeSec,
			)
		}
	}()

	//----------------------------------------
	// Graceful-shutdown
	//----------------------------------------
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("🛑 Клиент остановлен")
}

// маленький помощник для красивого вывода
func rightPad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + string(make([]byte, n-len(s)))
}
