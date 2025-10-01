package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
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

// EmulatorState для управления состоянием эмулятора
type EmulatorState struct {
	IsRunning    bool          `json:"is_running"`
	DeviceID     string        `json:"device_id"`
	SelectedKey  string        `json:"selected_key"`
	DataType     string        `json:"data_type"`
	Folders      []string      `json:"folders"`
	CurrentCycle int           `json:"current_cycle"`
	StartTime    time.Time     `json:"start_time"`
	Sessions     []SessionInfo `json:"sessions"`
	ctx          context.Context
	cancel       context.CancelFunc
}

// SessionInfo информация о текущем сеансе
type SessionInfo struct {
	FolderName       string    `json:"folder_name"`
	SessionName      string    `json:"session_name"`
	Status           string    `json:"status"`
	RecordsTotal     int       `json:"records_total"`
	RecordsProcessed int       `json:"records_processed"`
	StartTime        time.Time `json:"start_time"`
}

var (
	mqttClient    mqtt.Client
	emulatorState *EmulatorState
	emulatorMutex sync.RWMutex
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	fmt.Println("✓ Подключение к MQTT брокеру установлено")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Соединение с MQTT брокером потеряно: %v\n", err)
}

func initMQTTClient() error {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://mosquitto:1883")
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

func writeCSVFile(filename string, records []CSVRecord) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("не удалось создать файл %s: %v", filename, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{"time_sec", "value"}); err != nil {
		return fmt.Errorf("не удалось записать заголовок в %s: %v", filename, err)
	}

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

func selectRandomKey() (string, string, []string) {
	rand.Seed(time.Now().UnixNano())

	allKeys := make([]string, 0)

	for key := range Regular {
		allKeys = append(allKeys, key)
	}

	for key := range Hypoxia {
		allKeys = append(allKeys, key)
	}

	selectedKey := allKeys[rand.Intn(len(allKeys))]

	if folders, found := Regular[selectedKey]; found {
		folderList := strings.Split(folders, ",")
		sort.Slice(folderList, func(i, j int) bool {
			a, _ := strconv.Atoi(folderList[i])
			b, _ := strconv.Atoi(folderList[j])
			return a < b
		})
		return selectedKey, "regular", folderList
	}

	if folders, found := Hypoxia[selectedKey]; found {
		folderList := strings.Split(folders, ",")
		sort.Slice(folderList, func(i, j int) bool {
			a, _ := strconv.Atoi(folderList[i])
			b, _ := strconv.Atoi(folderList[j])
			return a < b
		})
		return selectedKey, "hypoxia", folderList
	}

	return selectedKey, "unknown", []string{}
}

func findPairedFilesInFolder(bpmDir, uterusDir string) ([][2]string, error) {
	re := regexp.MustCompile(`^([\d\-]+)_(\d+)\.csv$`)

	createFileMap := func(dir string) (map[string]string, error) {
		fileMap := make(map[string]string)
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return nil, fmt.Errorf("не удалось прочитать директорию %s: %v", dir, err)
		}
		

		for _, f := range files {
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

func normalizeAndSavePair(bpmPath, uterusPath string) (string, string, error) {
	bpmRecords, err := readCSVFile(bpmPath)
	if err != nil {
		return "", "", err
	}

	uterusRecords, err := readCSVFile(uterusPath)
	if err != nil {
		return "", "", err
	}

	bpmMap := make(map[float64]float64)
	for _, r := range bpmRecords {
		bpmMap[r.TimeSec] = r.Value
	}

	uterusMap := make(map[float64]float64)
	for _, r := range uterusRecords {
		uterusMap[r.TimeSec] = r.Value
	}

	allTimestampsMap := make(map[float64]bool)
	for t := range bpmMap {
		allTimestampsMap[t] = true
	}

	for t := range uterusMap {
		allTimestampsMap[t] = true
	}

	var sortedTimestamps []float64
	for t := range allTimestampsMap {
		sortedTimestamps = append(sortedTimestamps, t)
	}

	sort.Float64s(sortedTimestamps)

	var fixedBPM, fixedUterus []CSVRecord
	for _, ts := range sortedTimestamps {
		if val, ok := bpmMap[ts]; ok {
			fixedBPM = append(fixedBPM, CSVRecord{TimeSec: ts, Value: val})
		} else {
			fixedBPM = append(fixedBPM, CSVRecord{TimeSec: ts, Value: -1})
		}

		if val, ok := uterusMap[ts]; ok {
			fixedUterus = append(fixedUterus, CSVRecord{TimeSec: ts, Value: val})
		} else {
			fixedUterus = append(fixedUterus, CSVRecord{TimeSec: ts, Value: -1})
		}
	}

	fixedBPMPath := strings.Replace(bpmPath, ".csv", "_fixed.csv", 1)
	fixedUterusPath := strings.Replace(uterusPath, ".csv", "_fixed.csv", 1)

	if err := writeCSVFile(fixedBPMPath, fixedBPM); err != nil {
		return "", "", err
	}

	if err := writeCSVFile(fixedUterusPath, fixedUterus); err != nil {
		return "", "", err
	}

	fmt.Printf("Файлы нормализованы:\n -> %s\n -> %s\n", filepath.Base(fixedBPMPath), filepath.Base(fixedUterusPath))
	return fixedBPMPath, fixedUterusPath, nil
}

func emulateSession(bpmFile, uterusFile, deviceID string, speedMultiplier float64, sessionIndex int) error {
	select {
	case <-emulatorState.ctx.Done():
		return fmt.Errorf("эмуляция остановлена")
	default:
	}

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
		emulatorMutex.Lock()
		if sessionIndex < len(emulatorState.Sessions) {
			emulatorState.Sessions[sessionIndex].Status = "error"
		}
		emulatorMutex.Unlock()
		return fmt.Errorf("ошибка чтения файлов")
	}

	if len(bpmRecords) == 0 || len(uterusRecords) == 0 {
		emulatorMutex.Lock()
		if sessionIndex < len(emulatorState.Sessions) {
			emulatorState.Sessions[sessionIndex].Status = "error"
		}
		emulatorMutex.Unlock()
		return fmt.Errorf("один из файлов пуст")
	}

	numRecords := len(bpmRecords)
	if len(uterusRecords) < numRecords {
		numRecords = len(uterusRecords)
	}

	// Обновляем статус сеанса
	emulatorMutex.Lock()
	if sessionIndex < len(emulatorState.Sessions) {
		emulatorState.Sessions[sessionIndex].Status = "running"
		emulatorState.Sessions[sessionIndex].RecordsTotal = numRecords
		emulatorState.Sessions[sessionIndex].StartTime = time.Now()
	}
	emulatorMutex.Unlock()

	fmt.Printf("Сеанс %s начат. Записей для обработки: %d\n", filepath.Base(bpmFile), numRecords)

	for i := 0; i < numRecords; i++ {
		select {
		case <-emulatorState.ctx.Done():
			return fmt.Errorf("эмуляция остановлена")
		default:
		}

		var wgPublish sync.WaitGroup
		wgPublish.Add(2)

		go func(record CSVRecord) {
			defer wgPublish.Done()
			if record.Value == -1 {
				return
			}

			data := MedicalData{
				DeviceID:  deviceID,
				Timestamp: time.Now().UnixNano(),
				DataType:  "fetal_heart_rate",
				Value:     record.Value,
				Units:     "bpm",
				TimeSec:   record.TimeSec,
			}

			topic := fmt.Sprintf("medical/ctg/fetal_heart_rate/%s", deviceID)
			if err := publishMQTT(topic, data); err != nil {
				log.Printf("Ошибка отправки FHR: %v", err)
			} else {
				fmt.Printf("📡 FHR: %.1f bpm (t=%.1fs) -> %s\n", record.Value, record.TimeSec, topic)
			}
		}(bpmRecords[i])

		go func(record CSVRecord) {
			defer wgPublish.Done()
			if record.Value == -1 {
				return
			}

			data := MedicalData{
				DeviceID:  deviceID,
				Timestamp: time.Now().UnixNano(),
				DataType:  "uterine_contractions",
				Value:     record.Value,
				Units:     "mmHg",
				TimeSec:   record.TimeSec,
			}

			topic := fmt.Sprintf("medical/ctg/uterine_contractions/%s", deviceID)
			if err := publishMQTT(topic, data); err != nil {
				log.Printf("Ошибка отправки UC: %v", err)
			} else {
				fmt.Printf("📡 UC: %.1f mmHg (t=%.1fs) -> %s\n", record.Value, record.TimeSec, topic)
			}
		}(uterusRecords[i])

		wgPublish.Wait()

		// Обновляем прогресс
		emulatorMutex.Lock()
		if sessionIndex < len(emulatorState.Sessions) {
			emulatorState.Sessions[sessionIndex].RecordsProcessed = i + 1
		}
		emulatorMutex.Unlock()

		if i < numRecords-1 {
			sleepSeconds := (bpmRecords[i+1].TimeSec - bpmRecords[i].TimeSec) / speedMultiplier
			if sleepSeconds > 0 {
				sleepDuration := time.Duration(sleepSeconds * float64(time.Second))

				select {
				case <-time.After(sleepDuration):
				case <-emulatorState.ctx.Done():
					return fmt.Errorf("эмуляция остановлена")
				}
			}
		}
	}

	// Отмечаем сеанс как завершенный
	emulatorMutex.Lock()
	if sessionIndex < len(emulatorState.Sessions) {
		emulatorState.Sessions[sessionIndex].Status = "completed"
	}
	emulatorMutex.Unlock()

	return nil
}

// === HTTP API для управления эмулятором ===

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func getStatusHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	emulatorMutex.RLock()
	response := *emulatorState
	emulatorMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func startEmulatorHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	emulatorMutex.Lock()
	defer emulatorMutex.Unlock()

	if emulatorState.IsRunning {
		http.Error(w, "Эмулятор уже запущен", http.StatusConflict)
		return
	}

	// Инициализируем новую сессию
	selectedKey, dataType, folders := selectRandomKey()
	deviceID := fmt.Sprintf("CTG-MONITOR-%04d", 1+time.Now().Unix()%9998)

	ctx, cancel := context.WithCancel(context.Background())

	emulatorState.IsRunning = true
	emulatorState.DeviceID = deviceID
	emulatorState.SelectedKey = selectedKey
	emulatorState.DataType = dataType
	emulatorState.Folders = folders
	emulatorState.CurrentCycle = 0
	emulatorState.StartTime = time.Now()
	emulatorState.Sessions = []SessionInfo{}
	emulatorState.ctx = ctx
	emulatorState.cancel = cancel

	// Запускаем эмулятор в отдельной горутине
	go runEmulator()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "started",
		"device_id":    deviceID,
		"selected_key": selectedKey,
		"data_type":    dataType,
		"folders":      folders,
	})
}

func stopEmulatorHandler(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	emulatorMutex.Lock()
	defer emulatorMutex.Unlock()

	if !emulatorState.IsRunning {
		http.Error(w, "Эмулятор не запущен", http.StatusConflict)
		return
	}

	emulatorState.cancel()
	emulatorState.IsRunning = false

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func runEmulator() {
	defer func() {
		emulatorMutex.Lock()
		emulatorState.IsRunning = false
		emulatorMutex.Unlock()
	}()

	for {
		select {
		case <-emulatorState.ctx.Done():
			fmt.Println("\n🛑 Эмуляция остановлена пользователем")
			return
		default:
		}

		emulatorMutex.Lock()
		emulatorState.CurrentCycle++
		cycle := emulatorState.CurrentCycle
		folders := emulatorState.Folders
		dataType := emulatorState.DataType
		deviceID := emulatorState.DeviceID
		emulatorMutex.Unlock()

		fmt.Printf("\n🔄 Начинаем цикл обработки #%d...\n", cycle)

		for _, folder := range folders {
			select {
			case <-emulatorState.ctx.Done():
				return
			default:
			}

			fmt.Printf("\n==================== ОБРАБОТКА ПАПКИ %s ====================\n", folder)

			var bpmDir, uterusDir string

			if dataType == "regular" {
				bpmDir = filepath.Join("./data/regular", folder, "bpm")
				uterusDir = filepath.Join("./data/regular", folder, "uterus")
			} else if dataType == "hypoxia" {
				bpmDir = filepath.Join("./data/hypoxia", folder, "bpm")
				uterusDir = filepath.Join("./data/hypoxia", folder, "uterus")
			} else {
				log.Printf("Неизвестный тип данных: %s", dataType)
				continue
			}

			if _, err := os.Stat(bpmDir); os.IsNotExist(err) {
				log.Printf("Папка BPM не существует: %s. Пропускаем.", bpmDir)
				continue
			}
			if _, err := os.Stat(uterusDir); os.IsNotExist(err) {
				log.Printf("Папка Uterus не существует: %s. Пропускаем.", uterusDir)
				continue
			}

			pairedFiles, err := findPairedFilesInFolder(bpmDir, uterusDir)
			if err != nil || len(pairedFiles) == 0 {
				log.Printf("Не найдены парные файлы в папке %s. Пропускаем.", folder)
				continue
			}

			fmt.Printf("📂 Найдено %d парных сеансов в папке %s.\n", len(pairedFiles), folder)

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

			// Добавляем сеансы в состояние
			emulatorMutex.Lock()
			sessionStartIndex := len(emulatorState.Sessions)
			for _, pair := range normalizedFiles {
				emulatorState.Sessions = append(emulatorState.Sessions, SessionInfo{
					FolderName:  folder,
					SessionName: filepath.Base(pair[0]),
					Status:      "pending",
				})
			}
			emulatorMutex.Unlock()

			fmt.Printf("🔄 Нормализация завершена для папки %s. Готово к эмуляции %d сеансов.\n", folder, len(normalizedFiles))

			for i, pair := range normalizedFiles {
				select {
				case <-emulatorState.ctx.Done():
					return
				default:
				}

				sessionIndex := sessionStartIndex + i
				fmt.Printf("\n🚀 НАЧАЛО СЕАНСА КТГ (%s)\n", filepath.Base(pair[0]))

				if err := emulateSession(pair[0], pair[1], deviceID, 1.0, sessionIndex); err != nil {
					fmt.Printf("❌ ОШИБКА в сеансе КТГ %s: %v\n", filepath.Base(pair[0]), err)
					if err.Error() == "эмуляция остановлена" {
						return
					}
				} else {
					fmt.Printf("✅ СЕАНС КТГ %s ЗАВЕРШЕН\n", filepath.Base(pair[0]))
				}

				fmt.Println("⏸️ Пауза 5 секунд перед следующим сеансом...")
				select {
				case <-time.After(5 * time.Second):
				case <-emulatorState.ctx.Done():
					return
				}
			}

			fmt.Printf("==================== ПАПКА %s ЗАВЕРШЕНА ====================\n", folder)
			fmt.Println("⏸️ Пауза 10 секунд перед следующей папкой...")
			select {
			case <-time.After(10 * time.Second):
			case <-emulatorState.ctx.Done():
				return
			}
		}

		fmt.Println("\n🏁 Все папки завершены. Начинаем цикл заново через 15 секунд.")
		select {
		case <-time.After(15 * time.Second):
		case <-emulatorState.ctx.Done():
			return
		}
	}
}

func webInterfaceHandler(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>КТГ Эмулятор - Панель управления</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            margin: 0;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            overflow: hidden;
        }
        .header {
            background: linear-gradient(135deg, #4CAF50, #45a049);
            color: white;
            padding: 30px;
            text-align: center;
        }
        .header h1 {
            margin: 0;
            font-size: 2.5em;
            font-weight: 300;
        }
        .content {
            padding: 30px;
        }
        .control-panel {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 30px;
            margin-bottom: 30px;
        }
        .card {
            background: #f8f9fa;
            border-radius: 10px;
            padding: 25px;
            border: 1px solid #e9ecef;
        }
        .card h3 {
            color: #333;
            margin-top: 0;
            margin-bottom: 20px;
            font-size: 1.3em;
        }
        .btn {
            padding: 12px 25px;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 16px;
            font-weight: 600;
            transition: all 0.3s ease;
            margin: 5px;
            text-transform: uppercase;
        }
        .btn-start {
            background: linear-gradient(135deg, #4CAF50, #45a049);
            color: white;
        }
        .btn-start:hover:not(:disabled) {
            background: linear-gradient(135deg, #45a049, #4CAF50);
            transform: translateY(-2px);
        }
        .btn-stop {
            background: linear-gradient(135deg, #f44336, #d32f2f);
            color: white;
        }
        .btn-stop:hover:not(:disabled) {
            background: linear-gradient(135deg, #d32f2f, #f44336);
            transform: translateY(-2px);
        }
        .btn:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none !important;
        }
        .status {
            display: flex;
            align-items: center;
            margin: 15px 0;
            padding: 15px;
            border-radius: 8px;
            font-weight: 600;
        }
        .status.running {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
        }
        .status.stopped {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
        }
        .status-indicator {
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 10px;
            animation: pulse 2s infinite;
        }
        .status.running .status-indicator {
            background: #28a745;
        }
        .status.stopped .status-indicator {
            background: #dc3545;
            animation: none;
        }
        @keyframes pulse {
            0% { opacity: 1; }
            50% { opacity: 0.5; }
            100% { opacity: 1; }
        }
        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
            margin: 20px 0;
        }
        .info-item {
            background: white;
            padding: 15px;
            border-radius: 8px;
            border: 1px solid #e9ecef;
        }
        .info-label {
            font-weight: 600;
            color: #6c757d;
            font-size: 0.9em;
            margin-bottom: 5px;
        }
        .info-value {
            color: #333;
            font-size: 1.1em;
        }
        .sessions-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
            background: white;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .sessions-table th,
        .sessions-table td {
            padding: 12px 15px;
            text-align: left;
            border-bottom: 1px solid #e9ecef;
        }
        .sessions-table th {
            background: #f8f9fa;
            font-weight: 600;
            color: #495057;
        }
        .session-status {
            padding: 4px 10px;
            border-radius: 20px;
            font-size: 0.85em;
            font-weight: 600;
            text-transform: uppercase;
        }
        .session-status.pending { background: #fff3cd; color: #856404; }
        .session-status.running { background: #d1ecf1; color: #0c5460; }
        .session-status.completed { background: #d4edda; color: #155724; }
        .session-status.error { background: #f8d7da; color: #721c24; }
        .progress-bar {
            width: 100%;
            height: 8px;
            background: #e9ecef;
            border-radius: 4px;
            overflow: hidden;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #4CAF50, #45a049);
            transition: width 0.3s ease;
        }
        .logs {
            background: #1e1e1e;
            color: #00ff00;
            padding: 20px;
            border-radius: 8px;
            font-family: 'Courier New', monospace;
            font-size: 14px;
            max-height: 300px;
            overflow-y: auto;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🏥 КТГ Эмулятор</h1>
            <p>Панель управления медицинскими данными</p>
        </div>
        
        <div class="content">
            <div class="control-panel">
                <div class="card">
                    <h3>🎮 Управление эмулятором</h3>
                    <div id="status" class="status stopped">
                        <div class="status-indicator"></div>
                        <span>Эмулятор остановлен</span>
                    </div>
                    <button id="startBtn" class="btn btn-start">▶️ Запустить эмуляцию</button>
                    <button id="stopBtn" class="btn btn-stop" disabled>⏹️ Остановить эмуляцию</button>
                </div>
                
                <div class="card">
                    <h3>📊 Текущая статистика</h3>
                    <div class="info-grid">
                        <div class="info-item">
                            <div class="info-label">Устройство</div>
                            <div class="info-value" id="deviceId">-</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">Тип данных</div>
                            <div class="info-value" id="dataType">-</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">Цикл</div>
                            <div class="info-value" id="currentCycle">-</div>
                        </div>
                        <div class="info-item">
                            <div class="info-label">Время работы</div>
                            <div class="info-value" id="uptime">-</div>
                        </div>
                    </div>
                </div>
            </div>
            
            <div class="card">
                <h3>📋 Активные сеансы</h3>
                <table class="sessions-table">
                    <thead>
                        <tr>
                            <th>Папка</th>
                            <th>Сеанс</th>
                            <th>Статус</th>
                            <th>Прогресс</th>
                            <th>Время</th>
                        </tr>
                    </thead>
                    <tbody id="sessionsTable">
                        <tr>
                            <td colspan="5" style="text-align: center; color: #6c757d; padding: 40px;">
                                Нет активных сеансов
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>

    <script>
        const startBtn = document.getElementById('startBtn');
        const stopBtn = document.getElementById('stopBtn');
        const status = document.getElementById('status');
        
        function updateStatus(data) {
            const isRunning = data.is_running;
            
            if (isRunning) {
                status.className = 'status running';
                status.innerHTML = '<div class="status-indicator"></div><span>Эмулятор работает</span>';
                startBtn.disabled = true;
                stopBtn.disabled = false;
            } else {
                status.className = 'status stopped';
                status.innerHTML = '<div class="status-indicator"></div><span>Эмулятор остановлен</span>';
                startBtn.disabled = false;
                stopBtn.disabled = true;
            }
            
            document.getElementById('deviceId').textContent = data.device_id || '-';
            document.getElementById('dataType').textContent = data.data_type || '-';
            document.getElementById('currentCycle').textContent = data.current_cycle || '-';
            
            // Обновляем время работы
            if (data.start_time && isRunning) {
                const startTime = new Date(data.start_time);
                const now = new Date();
                const diff = now - startTime;
                const hours = Math.floor(diff / 3600000);
                const minutes = Math.floor((diff % 3600000) / 60000);
                const seconds = Math.floor((diff % 60000) / 1000);
                document.getElementById('uptime').textContent = 
                    hours.toString().padStart(2, '0') + ':' + 
                    minutes.toString().padStart(2, '0') + ':' + 
                    seconds.toString().padStart(2, '0');
            } else {
                document.getElementById('uptime').textContent = '-';
            }
            
            updateSessionsTable(data.sessions || []);
        }
        
        function updateSessionsTable(sessions) {
            const tbody = document.getElementById('sessionsTable');
            
            if (sessions.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" style="text-align: center; color: #6c757d; padding: 40px;">Нет активных сеансов</td></tr>';
                return;
            }
            
            tbody.innerHTML = sessions.map(session => {
                const progress = session.records_total > 0 ? 
                    Math.round((session.records_processed / session.records_total) * 100) : 0;
                
                const startTime = session.start_time ? 
                    new Date(session.start_time).toLocaleTimeString() : '-';
                
                return '<tr>' +
                    '<td>' + session.folder_name + '</td>' +
                    '<td>' + session.session_name + '</td>' +
                    '<td><span class="session-status ' + session.status + '">' + session.status + '</span></td>' +
                    '<td>' +
                        '<div class="progress-bar">' +
                            '<div class="progress-fill" style="width: ' + progress + '%"></div>' +
                        '</div>' +
                        '<small>' + session.records_processed + '/' + session.records_total + ' (' + progress + '%)</small>' +
                    '</td>' +
                    '<td>' + startTime + '</td>' +
                '</tr>';
            }).join('');
        }
        
        startBtn.addEventListener('click', async () => {
            try {
                const response = await fetch('/api/start', { method: 'POST' });
                if (!response.ok) {
                    const error = await response.text();
                    alert('Ошибка запуска: ' + error);
                }
            } catch (error) {
                alert('Ошибка подключения: ' + error.message);
            }
        });
        
        stopBtn.addEventListener('click', async () => {
            try {
                const response = await fetch('/api/stop', { method: 'POST' });
                if (!response.ok) {
                    const error = await response.text();
                    alert('Ошибка остановки: ' + error);
                }
            } catch (error) {
                alert('Ошибка подключения: ' + error.message);
            }
        });
        
        // Периодическое обновление статуса
        async function fetchStatus() {
            try {
                const response = await fetch('/api/status');
                if (response.ok) {
                    const data = await response.json();
                    updateStatus(data);
                }
            } catch (error) {
                console.error('Ошибка получения статуса:', error);
            }
        }
        
        // Первоначальная загрузка и периодическое обновление
        fetchStatus();
        setInterval(fetchStatus, 2000);
    </script>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func main() {
	log.SetFlags(log.LstdFlags)
	fmt.Println("=== ЭМУЛЯТОР МЕДИЦИНСКОГО ОБОРУДОВАНИЯ v7.0 (с Web UI) ===")

	// Инициализируем состояние эмулятора
	emulatorState = &EmulatorState{
		IsRunning: false,
		Sessions:  []SessionInfo{},
	}

	if err := initMQTTClient(); err != nil {
		log.Fatalf("Не удалось инициализировать MQTT клиент: %v", err)
	}
	defer mqttClient.Disconnect(250)

	// Настраиваем HTTP сервер
	http.HandleFunc("/", webInterfaceHandler)
	http.HandleFunc("/api/status", getStatusHandler)
	http.HandleFunc("/api/start", startEmulatorHandler)
	http.HandleFunc("/api/stop", stopEmulatorHandler)

	fmt.Println("🌐 Web интерфейс доступен на: http://localhost:8081")
	fmt.Println("📊 API endpoints:")
	fmt.Println("  GET  /api/status  - получить статус эмулятора")
	fmt.Println("  POST /api/start   - запустить эмуляцию")
	fmt.Println("  POST /api/stop    - остановить эмуляцию")
	fmt.Println()

	if err := http.ListenAndServe("0.0.0.0:8081", nil); err != nil {
		log.Fatalf("Не удалось запустить HTTP сервер: %v", err)
	}
}
