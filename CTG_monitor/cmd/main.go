// main.go - ИСПРАВЛЕННАЯ ВЕРСИЯ
package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "CTG_monitor/docs"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	_ "github.com/swaggo/files"
	_ "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"

	"CTG_monitor/configs"
	"CTG_monitor/internal/database"
	"CTG_monitor/internal/handlers"
	pb "CTG_monitor/proto"
)

func main() {
	log.Println(" === CTG MONITOR v2.0 (Stream Processing Architecture) ===")

	// 1. Загрузка конфигурации
	cfg := configs.LoadConfig()
	log.Printf("Конфигурация загружена: DB=%s:%s, MQTT=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.MQTT.Broker)

	// 2. Инициализация базы данных
	db, err := database.InitDatabase(cfg)
	if err != nil {
		log.Fatalf("Ошибка инициализации БД: %v", err)
	}
	defer database.CloseDatabase()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Ошибка миграций: %v", err)
	}

	// 3. Создание основных компонентов
	dataBuffer := handlers.NewDataBuffer(db)
	sessionManager := handlers.NewSessionManager(db, dataBuffer)
	grpcStreamer := handlers.NewGRPCStreamer()

	// 4. Создание MQTT Stream Processor
	mqttProcessor := handlers.NewMQTTStreamProcessor(
		sessionManager,
		grpcStreamer,
		dataBuffer,
	)

	// 5. Инициализация MQTT клиента
	mqttClient, err := initMQTTWithAuth(cfg.MQTT)
	if err != nil {
		log.Fatalf("Ошибка MQTT: %v", err)
	}
	defer mqttClient.Disconnect(250)

	// 6. Подписка на MQTT топики с правильным обработчиком
	messageHandler := func(client mqtt.Client, msg mqtt.Message) {
		mqttProcessor.HandleIncomingMQTT(msg.Topic(), msg.Payload())
	}

	topic := "medical/ctg/+/+" // Подписываемся на все устройства и типы данных
	token := mqttClient.Subscribe(topic, byte(cfg.MQTT.QoS), messageHandler)
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Ошибка подписки MQTT: %v", token.Error())
	}

	log.Printf("MQTT клиент подключён к %s, топик: %s",
		cfg.MQTT.Broker, topic)

	// 7. Запуск gRPC сервера
	grpcServer := grpc.NewServer()
	pb.RegisterCTGStreamServiceServer(grpcServer, grpcStreamer)

	go func() {
		lis, err := net.Listen("tcp", ":"+cfg.App.GRPCPort)
		if err != nil {
			log.Fatalf("Ошибка gRPC listener: %v", err)
		}

		log.Printf("gRPC Stream Server запущен на :%s", cfg.App.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ Ошибка gRPC сервера: %v", err)
		}
	}()

	if err := handlers.InitMedicalRecordsClient("localhost:50052"); err != nil {
		log.Printf("Не удалось подключиться к сервису медкарт: %v", err)
		log.Println("Продолжаем работу без интеграции с медкартами")
	}
	defer handlers.CloseMedicalRecordsClient()

	// 8. Запуск REST API сервера
	restAPI := handlers.NewRESTAPIServer(sessionManager, grpcStreamer, mqttProcessor)
	router := restAPI.SetupRoutes()

	go func() {
		log.Printf("REST API Server запущен на :%s", cfg.App.Port)
		if err := http.ListenAndServe(":"+cfg.App.Port, router); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Ошибка HTTP сервера: %v", err)
		}
	}()

	log.Println("Сервис запущен → Ctrl+C для остановки")
	log.Println("Архитектура потокового процессинга:")
	log.Println("MQTT 🔄 Stream Processor → gRPC Stream")
	log.Println("MQTT → Stream Processor → Data Buffer → Database")
	log.Println("REST API → Session Manager")

	// 9. Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Graceful shutdown...")

	// Остановка компонентов в обратном порядке
	mqttProcessor.Stop()
	grpcStreamer.Stop()
	dataBuffer.Stop()
	grpcServer.GracefulStop()

	log.Println("Сервис полностью остановлен")
}

// initMQTTWithAuth инициализирует MQTT клиент с аутентификацией
func initMQTTWithAuth(mqttCfg configs.MQTTConfig) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(mqttCfg.Broker)
	opts.SetClientID(mqttCfg.ClientID)

	if mqttCfg.Username != "" && mqttCfg.Password != "" {
		opts.SetUsername(mqttCfg.Username)
		opts.SetPassword(mqttCfg.Password)
		log.Printf("MQTT аутентификация: пользователь %s", mqttCfg.Username)
	}

	opts.SetAutoReconnect(true)
	opts.SetCleanSession(true)
	opts.OnConnect = func(c mqtt.Client) {
		fmt.Println("MQTT подключен")
	}
	opts.OnConnectionLost = func(c mqtt.Client, err error) {
		log.Printf("MQTT соединение потеряно: %v", err)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("MQTT подключение не удалось: %w", token.Error())
	}

	return client, nil
}
