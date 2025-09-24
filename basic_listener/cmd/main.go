package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Структура для получаемых данных
type MedicalData struct {
	DeviceID string  `json:"device_id"`
	Value    float64 `json:"value"`
	Units    string  `json:"units"`
}

// Обработчик сообщений
var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var data MedicalData
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return
	}
	fmt.Printf("Получено: [Топик: %s] Устройство: %s, Значение: %.2f %s\n",
		msg.Topic(), data.DeviceID, data.Value, data.Units)
}

// Обработчик подключения
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("✓ Слушатель подключен к MQTT")
	// Подписываемся на все топики в medical/ctg/
	topic := "medical/ctg/#"
	token := client.Subscribe(topic, 1, messageHandler)
	token.Wait()
	fmt.Printf("📬 Подписан на топик: %s\n", topic)
	fmt.Println("🎧 Ожидание данных...")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("⚠ Соединение потеряно: %v\n", err)
}

func main() {
	fmt.Println("=== СЛУШАТЕЛЬ МЕДИЦИНСКИХ ДАННЫХ ===")

	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetClientID(fmt.Sprintf("listener-%d", time.Now().Unix()))
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Ошибка подключения к MQTT: %v", token.Error())
	}

	// Ожидание сигнала завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Остановка слушателя...")
	client.Disconnect(250)
	fmt.Println("✅ Слушатель остановлен.")
}
