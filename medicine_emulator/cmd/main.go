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
	"strings"
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

// CSVRecord для чтения и записи данных из файла
type CSVRecord struct {
	TimeSec float64
	Value   float64
}

var mqttClient mqtt.Client

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
		// Пропускаем заголовок и некорректные строки
		if i == 0 || len(record) < 2 {
			continue
		}
		// Пропускаем строки с нечисловыми значениями (на случай старого заголовка)
		timeSec, errT := strconv.ParseFloat(record[0], 64)
		value, errV := strconv.ParseFloat(record[1], 64)
		if errT != nil || errV != nil {
			continue
		}
		csvRecords = append(csvRecords, CSVRecord{TimeSec: timeSec, Value: value})
	}
	return csvRecords, nil
}

// Новая функция для записи данных в CSV файл
func writeCSVFile(filename string, records []CSVRecord) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл %s: %v", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Записываем заголовок
	if err := writer.Write([]string{"time_sec", "value"}); err != nil {
		return fmt.Errorf("не удалось записать заголовок в %s: %v", filename, err)
	}

	// Записываем данные
	for _, record := range records {
		row := []string{
			strconv.FormatFloat(record.TimeSec, 'f', -1, 64),
			strconv.FormatFloat(record.Value, 'f', -1, 64),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("не удалось записать строку в %s: %v", filename, err)
		}
	}
	return nil
}

func findPairedFiles(bpmDir, uterusDir string) ([][2]string, error) {
	re := regexp.MustCompile(`^(\d{8}-\d{8})_\d+\.csv$`)

	createFileMap := func(dir string) (map[string]string, error) {
		fileMap := make(map[string]string)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать директорию %s: %v", dir, err)
		}
		for _, f := range files {
			// Игнорируем уже исправленные файлы
			if !f.IsDir() && !strings.HasSuffix(f.Name(), "_fixed.csv") {
				match := re.FindStringSubmatch(f.Name())
				if match != nil && len(match) > 1 {
					key := match[1]
					fileMap[key] = filepath.Join(dir, f.Name())
				}
			}
		}
		return fileMap, nil
	}

	bpmMap, err := createFileMap(bpmDir)
	if err != nil {
		return nil, err
	}
	uterusMap, err := createFileMap(uterusDir)
	if err != nil {
		return nil, err
	}

	var commonKeys []string
	for key := range bpmMap {
		if _, ok := uterusMap[key]; ok {
			commonKeys = append(commonKeys, key)
		}
	}
	sort.Strings(commonKeys)

	var pairedFiles [][2]string
	for _, key := range commonKeys {
		pair := [2]string{bpmMap[key], uterusMap[key]}
		pairedFiles = append(pairedFiles, pair)
	}
	return pairedFiles, nil
}

// --- Новая функция для нормализации и сохранения файлов ---
func normalizeAndSavePair(bpmPath, uterusPath string) (string, string, error) {
	bpmRecords, err := readCSVFile(bpmPath)
	if err != nil {
		return "", "", err
	}
	uterusRecords, err := readCSVFile(uterusPath)
	if err != nil {
		return "", "", err
	}

	// Создаем карты для быстрого доступа к значениям по времени
	bpmMap := make(map[float64]float64)
	for _, r := range bpmRecords {
		bpmMap[r.TimeSec] = r.Value
	}
	uterusMap := make(map[float64]float64)
	for _, r := range uterusRecords {
		uterusMap[r.TimeSec] = r.Value
	}

	// Собираем все уникальные временные метки из обоих файлов
	allTimestampsMap := make(map[float64]bool)
	for t := range bpmMap {
		allTimestampsMap[t] = true
	}
	for t := range uterusMap {
		allTimestampsMap[t] = true
	}

	// Конвертируем карту в слайс и сортируем
	var sortedTimestamps []float64
	for t := range allTimestampsMap {
		sortedTimestamps = append(sortedTimestamps, t)
	}
	sort.Float64s(sortedTimestamps)

	// Создаем новые, нормализованные записи
	var fixedBPM, fixedUterus []CSVRecord
	for _, ts := range sortedTimestamps {
		// Для BPM
		if val, ok := bpmMap[ts]; ok {
			fixedBPM = append(fixedBPM, CSVRecord{TimeSec: ts, Value: val})
		} else {
			fixedBPM = append(fixedBPM, CSVRecord{TimeSec: ts, Value: -1})
		}
		// Для Uterus
		if val, ok := uterusMap[ts]; ok {
			fixedUterus = append(fixedUterus, CSVRecord{TimeSec: ts, Value: val})
		} else {
			fixedUterus = append(fixedUterus, CSVRecord{TimeSec: ts, Value: -1})
		}
	}

	// Создаем имена для новых файлов
	fixedBPMPath := strings.Replace(bpmPath, ".csv", "_fixed.csv", 1)
	fixedUterusPath := strings.Replace(uterusPath, ".csv", "_fixed.csv", 1)

	// Записываем нормализованные данные в новые файлы
	if err := writeCSVFile(fixedBPMPath, fixedBPM); err != nil {
		return "", "", err
	}
	if err := writeCSVFile(fixedUterusPath, fixedUterus); err != nil {
		return "", "", err
	}

	fmt.Printf("✓ Файлы нормализованы:\n  -> %s\n  -> %s\n", filepath.Base(fixedBPMPath), filepath.Base(fixedUterusPath))

	return fixedBPMPath, fixedUterusPath, nil
}

// --- Основная логика эмуляции (без изменений) ---
func emulateSession(bpmFile, uterusFile, deviceID string, speedMultiplier float64, wg *sync.WaitGroup) {
	defer wg.Done()

	var bpmRecords, uterusRecords []CSVRecord
	var readErrBPM, readErrUterus error
	var readWg sync.WaitGroup
	readWg.Add(2)

	go func() {
		defer readWg.Done()
		bpmRecords, readErrBPM = readCSVFile(bpmFile)
	}()
	go func() {
		defer readWg.Done()
		uterusRecords, readErrUterus = readCSVFile(uterusFile)
	}()
	readWg.Wait()

	if readErrBPM != nil || readErrUterus != nil {
		log.Printf("Ошибка чтения одного из файлов для сеанса %s. Пропуск.", filepath.Base(bpmFile))
		return
	}
	if len(bpmRecords) == 0 || len(uterusRecords) == 0 {
		log.Printf("Сеанс для %s пропущен: один из файлов пуст.", filepath.Base(bpmFile))
		return
	}

	numRecords := len(bpmRecords)
	if len(uterusRecords) < numRecords {
		numRecords = len(uterusRecords)
	}

	fmt.Printf("✅ Сеанс %s начат. Записей для обработки: %d\n", filepath.Base(bpmFile), numRecords)

	for i := 0; i < numRecords; i++ {
		var wgPublish sync.WaitGroup
		wgPublish.Add(2)

		go func(record CSVRecord) {
			defer wgPublish.Done()
			if record.Value == -1 {
				return
			} // Не отправляем "пустые" значения
			data := MedicalData{
				DeviceID: deviceID, Timestamp: time.Now().UnixNano(), DataType: "fetal_heart_rate",
				Value: record.Value, Units: "bpm", TimeSec: record.TimeSec,
			}
			if err := publishMQTT("medical/ctg/fhr", data); err != nil {
				log.Printf("Ошибка отправки BPM: %v", err)
			}
		}(bpmRecords[i])

		go func(record CSVRecord) {
			defer wgPublish.Done()
			if record.Value == -1 {
				return
			} // Не отправляем "пустые" значения
			data := MedicalData{
				DeviceID: deviceID, Timestamp: time.Now().UnixNano(), DataType: "uterine_contractions",
				Value: record.Value, Units: "mmHg", TimeSec: record.TimeSec,
			}
			if err := publishMQTT("medical/ctg/uterus", data); err != nil {
				log.Printf("Ошибка отправки Uterus: %v", err)
			}
		}(uterusRecords[i])

		wgPublish.Wait()

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
	log.SetFlags(log.LstdFlags)
	fmt.Println("=== ЭМУЛЯТОР МЕДИЦИНСКОГО ОБОРУДОВАНИЯ v3.2 (с нормализацией данных) ===")

	if err := initMQTTClient(); err != nil {
		log.Fatalf("Не удалось инициализировать MQTT клиент: %v", err)
	}
	defer mqttClient.Disconnect(250)

	deviceID := fmt.Sprintf("CTG-MONITOR-%04d", 1+time.Now().Unix()%9998)
	bpmDir := "././data/bpm"
	uterusDir := "././data/uterus"

	// 1. Находим исходные парные файлы
	pairedFiles, err := findPairedFiles(bpmDir, uterusDir)
	if err != nil || len(pairedFiles) == 0 {
		log.Fatalf("Не найдены парные файлы для обработки в директориях %s и %s. Завершение работы.", bpmDir, uterusDir)
	}
	fmt.Printf("📂 Найдено %d парных сеансов для обработки.\n\n", len(pairedFiles))

	// 2. Нормализуем каждую пару и собираем пути к новым файлам
	var normalizedFiles [][2]string
	for _, pair := range pairedFiles {
		fixedBPM, fixedUterus, err := normalizeAndSavePair(pair[0], pair[1])
		if err != nil {
			log.Printf("Ошибка нормализации пары %s и %s: %v. Пропуск.", pair[0], pair[1], err)
			continue
		}
		normalizedFiles = append(normalizedFiles, [2]string{fixedBPM, fixedUterus})
	}

	if len(normalizedFiles) == 0 {
		log.Fatalf("Не удалось нормализовать ни одного сеанса. Завершение работы.")
	}

	fmt.Printf("\n🔄 Нормализация завершена. Готово к эмуляции %d сеансов.\n", len(normalizedFiles))

	// 3. Запускаем бесконечный цикл эмуляции с использованием _fixed файлов
	for {
		for _, pair := range normalizedFiles {
			fmt.Printf("\n==================== НАЧАЛО СЕАНСА КТГ (%s) ====================\n", filepath.Base(pair[0]))

			var wg sync.WaitGroup
			wg.Add(1)
			go emulateSession(pair[0], pair[1], deviceID, 1.0, &wg)
			wg.Wait()

			fmt.Printf("==================== СЕАНС КТГ %s ЗАВЕРШЕН ====================\n", filepath.Base(pair[0]))
			fmt.Println("⏸️  Пауза 5 секунд перед следующим сеансом...")
			time.Sleep(5 * time.Second)
		}
		fmt.Println("\n🏁 Все сеансы завершены. Начинаем цикл заново через 10 секунд.")
		time.Sleep(10 * time.Second)
	}
}
