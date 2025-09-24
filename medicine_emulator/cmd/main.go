package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MedicalData структура для отправки данных
type MedicalData struct {
	DeviceID  string  `json:"device_id"`
	Timestamp int64   `json:"timestamp"`
	DataType  string  `json:"data_type"`
	Value     float64 `json:"value"`
	Units     string  `json:"units"`
	TimeSec   float64 `json:"time_sec"`
}

// CSVRecord для чтения данных из файла
type CSVRecord struct {
	TimeSec float64
	Value   float64
}

var (
	mqttClient mqtt.Client
	logger     *log.Logger
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("✓ Подключение к MQTT брокеру установлено")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Соединение с MQTT брокером потеряно: %v\n", err)
}

func initMQTTClient() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1883")
	opts.SetClientID(fmt.Sprintf("medical-device-%d", time.Now().Unix()))
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

func publishMQTT(topic string, data MedicalData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка сериализации JSON: %v", err)
	}
	token := mqttClient.Publish(topic, 1, false, jsonData)
	if !token.WaitTimeout(2 * time.Second) {
		return fmt.Errorf("таймаут отправки MQTT")
	}
	return token.Error()
}

// --- Функции для работы с файлами ---
func readCSVFile(filename string) ([]CSVRecord, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла %s: %v", filename, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения CSV файла %s: %v", filename, err)
	}
	var csvRecords []CSVRecord
	for i, record := range records {
		if i == 0 || len(record) < 2 {
			continue
		}
		timeSec, errT := strconv.ParseFloat(record[0], 64)
		value, errV := strconv.ParseFloat(record[1], 64)
		if errT != nil || errV != nil {
			continue
		}
		csvRecords = append(csvRecords, CSVRecord{TimeSec: timeSec, Value: value})
	}
	return csvRecords, nil
}

// Новая функция для поиска парных файлов по общему идентификатору
func findPairedFiles(bpmDir, uterusDir string) ([][2]string, error) {
	// Регулярное выражение для извлечения идентификатора из имени файла
	// Пример: "20250829-01200001_1.csv" -> ключ: "20250829-01200001"
	re := regexp.MustCompile(`^(\d{8}-\d{8})_\d+\.csv$`)

	// Функция для создания карты "ключ -> полный путь"
	createFileMap := func(dir string) (map[string]string, error) {
		fileMap := make(map[string]string)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать директорию %s: %v", dir, err)
		}
		for _, f := range files {
			if !f.IsDir() {
				match := re.FindStringSubmatch(f.Name())
				if match != nil && len(match) > 1 {
					key := match[1]
					fileMap[key] = filepath.Join(dir, f.Name())
				}
			}
		}
		return fileMap, nil
	}

	// Создаем карты для bpm и uterus файлов
	bpmMap, err := createFileMap(bpmDir)
	if err != nil {
		return nil, err
	}
	uterusMap, err := createFileMap(uterusDir)
	if err != nil {
		return nil, err
	}

	// Находим общие ключи
	var commonKeys []string
	for key := range bpmMap {
		if _, ok := uterusMap[key]; ok {
			commonKeys = append(commonKeys, key)
		}
	}

	// Сортируем ключи для хронологического порядка
	sort.Strings(commonKeys)

	// Формируем итоговый список парных файлов
	var pairedFiles [][2]string
	for _, key := range commonKeys {
		pair := [2]string{bpmMap[key], uterusMap[key]}
		pairedFiles = append(pairedFiles, pair)
	}

	return pairedFiles, nil
}

// --- Основная логика эмуляции ---
func emulateSession(bpmFile, uterusFile, deviceID string, speedMultiplier float64, wg *sync.WaitGroup) {
	defer wg.Done()

	var bpmRecords, uterusRecords []CSVRecord
	var readErr error

	// Параллельно читаем оба файла
	var readWg sync.WaitGroup
	readWg.Add(2)

	go func() {
		defer readWg.Done()
		bpmRecords, readErr = readCSVFile(bpmFile)
		if readErr != nil {
			log.Printf("Ошибка чтения файла BPM %s: %v", bpmFile, readErr)
		}
	}()

	go func() {
		defer readWg.Done()
		uterusRecords, readErr = readCSVFile(uterusFile)
		if readErr != nil {
			log.Printf("Ошибка чтения файла Uterus %s: %v", uterusFile, readErr)
		}
	}()

	readWg.Wait()

	if len(bpmRecords) == 0 || len(uterusRecords) == 0 {
		log.Printf("Сеанс для %s пропущен: один из файлов пуст или нечитаем.", filepath.Base(bpmFile))
		return
	}

	// Синхронизируем по минимальной длине
	numRecords := len(bpmRecords)
	if len(uterusRecords) < numRecords {
		numRecords = len(uterusRecords)
	}

	fmt.Printf("✅ Сеанс %s начат. Записей для обработки: %d\n", filepath.Base(bpmFile)[:17], numRecords)

	for i := 0; i < numRecords; i++ {
		var wgPublish sync.WaitGroup
		wgPublish.Add(2)

		// Отправляем данные BPM
		go func(record CSVRecord) {
			defer wgPublish.Done()
			data := MedicalData{
				DeviceID:  deviceID,
				Timestamp: time.Now().UnixNano(),
				DataType:  "fetal_heart_rate",
				Value:     record.Value,
				Units:     "bpm",
				TimeSec:   record.TimeSec,
			}
			if err := publishMQTT("medical/ctg/fhr", data); err != nil {
				log.Printf("Ошибка отправки BPM: %v", err)
			}
		}(bpmRecords[i])

		// Отправляем данные Uterus
		go func(record CSVRecord) {
			defer wgPublish.Done()
			data := MedicalData{
				DeviceID:  deviceID,
				Timestamp: time.Now().UnixNano(),
				DataType:  "uterine_contractions",
				Value:     record.Value,
				Units:     "mmHg",
				TimeSec:   record.TimeSec,
			}
			if err := publishMQTT("medical/ctg/uterus", data); err != nil {
				log.Printf("Ошибка отправки Uterus: %v", err)
			}
		}(uterusRecords[i])

		wgPublish.Wait()

		// Задержка для симуляции реального времени
		if i < numRecords-1 {
			sleepSeconds := (bpmRecords[i+1].TimeSec - bpmRecords[i].TimeSec) / speedMultiplier
			if sleepSeconds > 0 {
				time.Sleep(time.Duration(sleepSeconds * float64(time.Second)))
			}
		}
	}
}

// Главная функция
func main() {
	logger = log.New(os.Stdout, "[EMULATOR] ", log.LstdFlags)
	fmt.Println("=== ЭМУЛЯТОР МЕДИЦИНСКОГО ОБОРУДОВАНИЯ v3.1 (Синхронные сеансы) ===")

	if err := initMQTTClient(); err != nil {
		log.Fatalf("Не удалось инициализировать MQTT клиент: %v", err)
	}
	defer mqttClient.Disconnect(250)

	deviceID := fmt.Sprintf("CTG-MONITOR-%04d", 1+time.Now().Unix()%9998)

	bpmDir := "./data/bpm"
	uterusDir := "./data/uterus"

	pairedFiles, err := findPairedFiles(bpmDir, uterusDir)
	if err != nil || len(pairedFiles) == 0 {
		log.Fatalf("Не найдены парные файлы в директориях %s и %s. Завершение работы.", bpmDir, uterusDir)
	}

	fmt.Printf("📂 Найдено %d парных сеансов для эмуляции.\n\n", len(pairedFiles))

	for { // Бесконечный цикл для повторения всех сеансов
		for _, pair := range pairedFiles {
			fmt.Printf("\n==================== НАЧАЛО НОВОГО СЕАНСА КТГ (%s) ====================\n", pair[0])

			var wg sync.WaitGroup
			wg.Add(1)

			go emulateSession(pair[0], pair[1], deviceID, 1.0, &wg)
			wg.Wait()

			fmt.Printf("==================== СЕАНС КТГ %s ЗАВЕРШЕН ====================\n", pair[0])
			fmt.Println("⏸️  Пауза 5 секунд перед следующим сеансом...")
			time.Sleep(5 * time.Second)
		}
		fmt.Println("\n🏁 Все сеансы завершены. Начинаем цикл заново.")
	}
}
