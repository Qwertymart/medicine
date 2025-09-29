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
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Regular - полный словарик для обычных случаев (данные из regular.xlsx)
var Regular = map[string]string{
	"1234567890123456": "155,156",
	"2345678901234567": "150",
	"3456789012345678": "139",
	"4567890123456789": "134",
	"5678901234567890": "128,129",
	"6789012345678901": "114,115",
	"7890123456789012": "112",
	"8901234567890123": "5",
	"9012345678901234": "48,49",
	"0123456789012345": "53",
	"1111222233334444": "110",
	"2222333344445555": "148,149",
	"3333444455556666": "142,143,144",
	"4444555566667777": "135,136",
	"5555666677778888": "145",
	"6666777788889999": "20,21",
	"7777888899990000": "132,133",
	"8888999900001111": "152,153",
	"9999000011112222": "38,39,40,41",
	"0000111122223333": "130,131",
	"1111333355557777": "124,125",
	"2222444466668888": "8,9",
	"3333555577779999": "121,122",
	"4444666688880000": "111",
	"5555777799991111": "103,104,105,106",
	"6666888800002222": "55,56",
	"7777999911113333": "1",
	"8888000022224444": "2,3",
	"9999111133335555": "6,7",
	"0000222244446666": "10",
	"1111444466668888": "11,12",
	"2222555577779999": "14",
	"3333666688880000": "15",
	"4444777799991111": "16,17,18",
	"5555888800002222": "19",
	"6666999911113333": "22,23,24,25",
	"7777000022224444": "26,27",
	"8888111133335555": "28,29",
	"9999222244446666": "33",
	"0000333355557777": "34",
	"1111555577779999": "35,36",
	"2222666688880000": "37",
	"3333777799991111": "42",
	"4444888800002222": "43,44",
	"5555999911113333": "45",
	"6666000022224444": "46",
	"7777111133335555": "47",
	"8888222244446666": "51",
	"9999333355557777": "52",
	"0000444466668888": "54",
	"1111666688880000": "57",
	"2222777799991111": "157,158,159,160",
	"3333888800002222": "154",
	"4444999911113333": "151",
	"5555000022224444": "147",
	"6666111133335555": "146",
	"7777222244446666": "140,141",
	"8888333355557777": "137,138",
	"9999444466668888": "126,127",
	"0000555577779999": "123",
	"1111777799991111": "120",
	"2222888800002222": "117,118",
	"3333999911113333": "108,109",
	"4444000022224444": "107",
}

// Hypoxia - полный словарик для случаев гипоксии (данные из hypoxia.xlsx)  
var Hypoxia = map[string]string{
	"1010101010101010": "2",
	"2020202020202020": "12",
	"3030303030303030": "13",
	"4040404040404040": "22",
	"5050505050505050": "16",
	"6060606060606060": "3",
	"7070707070707070": "10",
	"8080808080808080": "21",
	"9090909090909090": "7",
	"1212121212121212": "5",
	"1313131313131313": "1",
	"1414141414141414": "8",
	"1515151515151515": "4",
	"1616161616161616": "17",
	"1717171717171717": "18",
	"1818181818181818": "6",
	"1919191919191919": "14",
	"2121212121212121": "15",
	"2323232323232323": "20",
	"2424242424242424": "19",
	"2525252525252525": "9",
	"2626262626262626": "30",
	"2727272727272727": "31,32",
	"2828282828282828": "50",
	"2929292929292929": "23",
	"3131313131313131": "24,25,26",
	"3232323232323232": "27,28",
}

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

// Функция для поиска случайного ключа и определения типа данных
func selectRandomKey() (string, string, []string) {
	rand.Seed(time.Now().UnixNano())

	// Собираем все ключи из обоих словариков
	allKeys := make([]string, 0)

	// Добавляем ключи из Regular
	for key := range Regular {
		allKeys = append(allKeys, key)
	}

	// Добавляем ключи из Hypoxia
	for key := range Hypoxia {
		allKeys = append(allKeys, key)
	}

	// Выбираем случайный ключ
	selectedKey := allKeys[rand.Intn(len(allKeys))]

	// Определяем, в каком словарике найден ключ и получаем папки
	if folders, found := Regular[selectedKey]; found {
		folderList := strings.Split(folders, ",")
		// Сортируем папки по возрастанию
		sort.Slice(folderList, func(i, j int) bool {
			a, _ := strconv.Atoi(folderList[i])
			b, _ := strconv.Atoi(folderList[j])
			return a < b
		})
		return selectedKey, "regular", folderList
	}

	if folders, found := Hypoxia[selectedKey]; found {
		folderList := strings.Split(folders, ",")
		// Сортируем папки по возрастанию
		sort.Slice(folderList, func(i, j int) bool {
			a, _ := strconv.Atoi(folderList[i])
			b, _ := strconv.Atoi(folderList[j])
			return a < b
		})
		return selectedKey, "hypoxia", folderList
	}

	// Это не должно произойти, но на всякий случай
	return selectedKey, "unknown", []string{}
}

// Функция поиска файлов из старой логики, адаптированная для новых папок
func findPairedFilesInFolder(bpmDir, uterusDir string) ([][2]string, error) {
	re := regexp.MustCompile(`^([\d\-]+)_(\d+)\.csv$`)

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
					key := match[1] // Используем дату как ключ
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

// --- Функция для нормализации и сохранения файлов ---
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

	fmt.Printf("✓ Файлы нормализованы:\n -> %s\n -> %s\n", filepath.Base(fixedBPMPath), filepath.Base(fixedUterusPath))
	return fixedBPMPath, fixedUterusPath, nil
}

// --- ИСПРАВЛЕННАЯ функция эмуляции с правильными endpoint топиками ---
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

		// 🔥 ИСПРАВЛЕНО: Отправляем FHR данные с правильным endpoint топиком
		go func(record CSVRecord) {
			defer wgPublish.Done()
			if record.Value == -1 {
				return // Не отправляем "пустые" значения
			}

			data := MedicalData{
				DeviceID:  deviceID,
				Timestamp: time.Now().UnixNano(),
				DataType:  "fetal_heart_rate",
				Value:     record.Value,
				Units:     "bpm",
				TimeSec:   record.TimeSec,
			}

			// 🔥 ПРАВИЛЬНЫЙ ENDPOINT ТОПИК: medical/ctg/fetal_heart_rate/{deviceID}
			topic := fmt.Sprintf("medical/ctg/fetal_heart_rate/%s", deviceID)
			if err := publishMQTT(topic, data); err != nil {
				log.Printf("Ошибка отправки FHR: %v", err)
			} else {
				fmt.Printf("📡 FHR: %.1f bpm (t=%.1fs) -> %s\n", record.Value, record.TimeSec, topic)
			}
		}(bpmRecords[i])

		// 🔥 ИСПРАВЛЕНО: Отправляем UC данные с правильным endpoint топиком
		go func(record CSVRecord) {
			defer wgPublish.Done()
			if record.Value == -1 {
				return // Не отправляем "пустые" значения
			}

			data := MedicalData{
				DeviceID:  deviceID,
				Timestamp: time.Now().UnixNano(),
				DataType:  "uterine_contractions",
				Value:     record.Value,
				Units:     "mmHg",
				TimeSec:   record.TimeSec,
			}

			// 🔥 ПРАВИЛЬНЫЙ ENDPOINT ТОПИК: medical/ctg/uterine_contractions/{deviceID}
			topic := fmt.Sprintf("medical/ctg/uterine_contractions/%s", deviceID)
			if err := publishMQTT(topic, data); err != nil {
				log.Printf("Ошибка отправки UC: %v", err)
			} else {
				fmt.Printf("📡 UC: %.1f mmHg (t=%.1fs) -> %s\n", record.Value, record.TimeSec, topic)
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

// Главная функция - ОБЪЕДИНЕННАЯ ЛОГИКА
func main() {
	log.SetFlags(log.LstdFlags)
	fmt.Println("=== ЭМУЛЯТОР МЕДИЦИНСКОГО ОБОРУДОВАНИЯ v6.0 (ОБЪЕДИНЕННАЯ ВЕРСИЯ) ===")

	if err := initMQTTClient(); err != nil {
		log.Fatalf("Не удалось инициализировать MQTT клиент: %v", err)
	}
	defer mqttClient.Disconnect(250)

	deviceID := fmt.Sprintf("CTG-MONITOR-%04d", 1+time.Now().Unix()%9998)

	// 1. НОВАЯ ЛОГИКА: Выбираем случайного человека из словариков
	selectedKey, dataType, folders := selectRandomKey()
	fmt.Printf("🎲 Выбран случайный ключ: %s\n", selectedKey)
	fmt.Printf("📂 Тип данных: %s\n", dataType)
	fmt.Printf("📁 Папки для обработки: %v\n", folders)

	// 2. СТАРАЯ ЛОГИКА: Выводим правильные endpoint топики
	fmt.Printf("🔧 Device ID: %s\n", deviceID)
	fmt.Printf("📡 MQTT endpoint топики:\n")
	fmt.Printf(" • medical/ctg/fetal_heart_rate/%s\n", deviceID)
	fmt.Printf(" • medical/ctg/uterine_contractions/%s\n\n", deviceID)

	// 3. ИСПРАВЛЕННАЯ ЛОГИКА: Обрабатываем каждую папку по порядку с правильными путями
	for {
		fmt.Println("\n🔄 Начинаем новый цикл обработки...")

		for _, folder := range folders {
			fmt.Printf("\n==================== ОБРАБОТКА ПАПКИ %s ====================\n", folder)

			var bpmDir, uterusDir string

			// ИСПРАВЛЕНО: Правильные пути к папкам
			if dataType == "regular" {
				// Для regular данные лежат в ./data/regular/[номер_папки]/bpm и ./data/regular/[номер_папки]/uterus
				bpmDir = filepath.Join("./data/regular", folder, "bpm")
				uterusDir = filepath.Join("./data/regular", folder, "uterus")
			} else if dataType == "hypoxia" {
				// Для hypoxia данные лежат в ./data/hypoxia/[номер_папки]/bpm и ./data/hypoxia/[номер_папки]/uterus
				bpmDir = filepath.Join("./data/hypoxia", folder, "bpm")
				uterusDir = filepath.Join("./data/hypoxia", folder, "uterus")
			} else {
				log.Printf("Неизвестный тип данных: %s", dataType)
				continue
			}

			// Проверяем существование директорий
			if _, err := os.Stat(bpmDir); os.IsNotExist(err) {
				log.Printf("Папка BPM не существует: %s. Пропускаем.", bpmDir)
				continue
			}
			if _, err := os.Stat(uterusDir); os.IsNotExist(err) {
				log.Printf("Папка Uterus не существует: %s. Пропускаем.", uterusDir)
				continue
			}

			// Находим парные файлы в текущей папке
			pairedFiles, err := findPairedFilesInFolder(bpmDir, uterusDir)
			if err != nil || len(pairedFiles) == 0 {
				log.Printf("Не найдены парные файлы в папке %s. Пропускаем.", folder)
				continue
			}

			fmt.Printf("📂 Найдено %d парных сеансов в папке %s.\n", len(pairedFiles), folder)

			// Нормализуем каждую пару и собираем пути к новым файлам
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
				log.Printf("Не удалось нормализовать ни одного сеанса в папке %s.", folder)
				continue
			}

			fmt.Printf("🔄 Нормализация завершена для папки %s. Готово к эмуляции %d сеансов.\n", folder, len(normalizedFiles))

			// Запускаем эмуляцию для всех сеансов в текущей папке
			for _, pair := range normalizedFiles {
				fmt.Printf("\n🚀 НАЧАЛО СЕАНСА КТГ (%s)\n", filepath.Base(pair[0]))
				var wg sync.WaitGroup
				wg.Add(1)
				go emulateSession(pair[0], pair[1], deviceID, 1.0, &wg)
				wg.Wait()
				fmt.Printf("✅ СЕАНС КТГ %s ЗАВЕРШЕН\n", filepath.Base(pair[0]))
				fmt.Println("⏸️ Пауза 5 секунд перед следующим сеансом...")
				time.Sleep(5 * time.Second)
			}

			fmt.Printf("==================== ПАПКА %s ЗАВЕРШЕНА ====================\n", folder)
			fmt.Println("⏸️ Пауза 10 секунд перед следующей папкой...")
			time.Sleep(10 * time.Second)
		}

		fmt.Println("\n🏁 Все папки завершены. Начинаем цикл заново через 15 секунд.")
		time.Sleep(15 * time.Second)
	}
}