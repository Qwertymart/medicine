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

type MedicalData struct {
	DeviceID string  `json:"device_id"`
	DataType string  `json:"data_type"`
	Value    float64 `json:"value"`
	Units    string  `json:"units"`
	TimeSec  float64 `json:"time_sec"`
}

var messageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	var data MedicalData
	if err := json.Unmarshal(msg.Payload(), &data); err != nil {
		log.Printf("Ошибка декодирования JSON: %v", err)
		return
	}

	if data.DataType == "fetal_heart_rate" {
		fmt.Printf("BPM: %.3f, %.2f\n", data.TimeSec, data.Value)
	} else if data.DataType == "uterine_contractions" {
		fmt.Printf("UTERUS: %.3f, %.2f\n", data.TimeSec, data.Value)
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Слушатель подключен к MQTT")
	topic := "medical/ctg/#"
	token := client.Subscribe(topic, 1, messageHandler)
	token.Wait()
	fmt.Printf("Подписан на топик: %s\n", topic)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Соединение потеряно: %v\n", err)
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Остановка слушателя...")
	client.Disconnect(250)
	fmt.Println("Слушатель остановлен.")
}
