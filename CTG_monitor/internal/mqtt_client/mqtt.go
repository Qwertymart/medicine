package mqtt_client

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"CTG_monitor/internal/handlers"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("Слушатель подключен к MQTT")
	topic := "medical/ctg/#"
	token := client.Subscribe(topic, 1, handlers.MessageHandler)
	token.Wait()
	fmt.Printf("Подписан на топик: %s\n", topic)
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Соединение потеряно: %v\n", err)
}

func InitClient(broker string) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(fmt.Sprintf("listener-%d", time.Now().Unix()))
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	return client, nil
}
