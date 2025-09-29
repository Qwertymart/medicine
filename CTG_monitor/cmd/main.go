package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"CTG_monitor/internal/handlers"
	"CTG_monitor/internal/mqtt_client"
	pb "CTG_monitor/proto"
)

func main() {
	log.Println("🏥 === CTG MONITOR (gRPC + MQTT) ===")

	//------------------------------------
	// 1. gRPC-сервер
	//------------------------------------
	grpcServer := grpc.NewServer()
	ctgSrv := handlers.NewCTGStreamServer()
	pb.RegisterCTGStreamServiceServer(grpcServer, ctgSrv)
	handlers.SetGRPCStreamServer(ctgSrv)

	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("❌ gRPC listener error: %v", err)
		}
		log.Println("🌊 gRPC Stream Server на :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ gRPC server error: %v", err)
		}
	}()

	//------------------------------------
	// 2. MQTT-клиент-слушатель
	//------------------------------------
	mqttClient, err := mqtt_client.InitClient("tcp://localhost:1883")
	if err != nil {
		log.Fatalf("❌ MQTT init error: %v", err)
	}
	defer mqttClient.Disconnect(250)

	log.Println("📡 MQTT клиент подключён – данные поступают в MessageHandler")
	log.Println("📈 На фронт уходит ТОЛЬКО: device_id, data_type, value, time_sec")

	//------------------------------------
	// 3. Graceful shutdown
	//------------------------------------
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	log.Println("🚀 Сервис запущен → Ctrl+C для остановки")
	<-sig

	log.Println("🛑 Остановка…")
	grpcServer.GracefulStop()
	log.Println("✅ Завершено")
}
