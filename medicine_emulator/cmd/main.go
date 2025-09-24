package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MedicalData структура для данных
type MedicalData struct {
	DeviceID  string  `json:"device_id"`
	Timestamp int64   `json:"timestamp"`
	DataType  string  `json:"data_type"`
	Value     float64 `json:"value"`
	Units     string  `json:"units"`
	TimeSec   float64 `json:"time_sec"`
}

// CSVRecord для записей чтения CSV
type CSVRecord struct {
	TimeSec float64
	Value   float64
}

// Глобальные переменные
var (
	mqttClient mqtt.Client
	logger     *log.Logger
)

// Обработчики MQTT
var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("✓ Подключение к MQTT брокеру установлено")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("⚠ Соединение с MQTT брокером потеряно: %v\n", err)
}

// Функция для чтения CSV файла
func readCSVFile(filename string) ([]CSVRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла %s: %v", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения CSV файла: %v", err)
	}

	var csvRecords []CSVRecord
	// Пропускаем заголовок (первую строку)
	for i, record := range records {
		if i == 0 {
			continue
		}

		timeSec, err := strconv.ParseFloat(record[0], 64)
		if err != nil {
			// Пропускаем строки с ошибками парсинга
			continue
		}

		value, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			// Пропускаем строки с ошибками парсинга
			continue
		}

		csvRecords = append(csvRecords, CSVRecord{
			TimeSec: timeSec,
			Value:   value,
		})
	}

	return csvRecords, nil
}

// Новая функция для получения списка файлов из директории
func getFilesFromDir(dirPath string) ([]string, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать директорию %s: %v", dirPath, err)
	}

	var fileNames []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".csv") {
			fileNames = append(fileNames, filepath.Join(dirPath, file.Name()))
		}
	}

	// Сортируем файлы по имени для последовательной обработки
	sort.Strings(fileNames)
	return fileNames, nil
}

// Функция для инициализации MQTT клиента
func initMQTTClient() error {
	broker := "localhost"
	port := 1883
	clientID := fmt.Sprintf("medical-device-%d", time.Now().Unix())

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetCleanSession(true)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("ошибка подключения к MQTT: %v", token.Error())
	}

	return nil
}

// Функция для отправки данных через MQTT
func publishMQTT(topic string, data MedicalData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %v", err)
	}

	token := mqttClient.Publish(topic, 1, false, jsonData) // QoS 1
	if !token.WaitTimeout(5 * time.Second) {
		return fmt.Errorf("таймаут отправки MQTT сообщения")
	}
	if token.Error() != nil {
		return fmt.Errorf("ошибка отправки MQTT сообщения: %v", token.Error())
	}
	return nil
}

// Обновленная функция для эмуляции данных из файлов директории
func emulateDataFromFiles(dir, dataType, units, topic, deviceID string, speedMultiplier float64) {
	files, err := getFilesFromDir(dir)
	if err != nil || len(files) == 0 {
		log.Printf("⚠️  Директория %s пуста или не найдена. Поток '%s' не будет запущен.", dir, dataType)
		return
	}

	fmt.Printf("🚀 Запускаем эмуляцию из директории %s (найдено %d файлов)\n", dir, len(files))

	for { // Бесконечный цикл для повторения
		for _, file := range files {
			records, err := readCSVFile(file)
			if err != nil {
				log.Printf("Ошибка чтения файла %s: %v", file, err)
				continue
			}

			for i, record := range records {
				data := MedicalData{
					DeviceID:  deviceID,
					Timestamp: time.Now().Unix(),
					DataType:  dataType,
					Value:     record.Value,
					Units:     units,
					TimeSec:   record.TimeSec,
				}

				if err := publishMQTT(topic, data); err != nil {
					log.Printf("Ошибка отправки данных: %v", err)
					time.Sleep(1 * time.Second) // Пауза при ошибке отправки
					continue
				}

				if i%50 == 0 { // Логируем реже
					fmt.Printf("📊 [%s] %s: %.2f %s (файл: %s)\n",
						dataType, deviceID, data.Value, units, filepath.Base(file))
				}

				if i < len(records)-1 {
					nextTime := records[i+1].TimeSec
					currentTime := record.TimeSec
					sleepDuration := time.Duration((nextTime-currentTime)*1000/speedMultiplier) * time.Millisecond
					if sleepDuration > 0 {
						time.Sleep(sleepDuration)
					}
				}
			}
		}
		fmt.Printf("✅ Цикл для '%s' завершен. Начинаем заново...\n", dataType)
	}
}

// Главная функция
func main() {
	logger = log.New(os.Stdout, "[EMULATOR] ", log.LstdFlags)

	fmt.Println("=== ЭМУЛЯТОР МЕДИЦИНСКОГО ОБОРУДОВАНИЯ v2.0 ===")
	fmt.Println("Протокол: MQTT")
	fmt.Println("Режим: Чтение из директорий `data/bpm` и `data/uterus`")

	if err := initMQTTClient(); err != nil {
		log.Fatalf("Не удалось инициализировать MQTT клиент: %v", err)
	}
	defer mqttClient.Disconnect(250)

	deviceID := fmt.Sprintf("CTG-MONITOR-%04d", 1+rand.Intn(9998))

	fmt.Printf("🏥 Устройство: %s\n", deviceID)

	// Запускаем эмуляцию данных из директорий в параллельных горутинах
	go emulateDataFromFiles("./data/bpm", "fetal_heart_rate", "bpm", "medical/ctg/fhr", deviceID, 10.0)
	go emulateDataFromFiles("./data/uterus", "uterine_contractions", "mmHg", "medical/ctg/uterus", deviceID, 10.0)

	fmt.Println("\n🚀 Эмуляция запущена. Данные отправляются непрерывно.")
	fmt.Println("Для остановки нажмите Ctrl+C")

	// Бесконечное ожидание
	select {}
}
