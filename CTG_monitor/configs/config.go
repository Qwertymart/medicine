// configs/config.go
package configs

import (
	"os"
	"strconv"
)

type Config struct {
	Database DatabaseConfig
	App      AppConfig
	MQTT     MQTTConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

type AppConfig struct {
	Port     string // HTTP_PORT из .env
	GRPCPort string // GRPC_PORT из .env
	LogLevel string
}

type MQTTConfig struct {
	Broker   string
	ClientID string
	Username string // ✅ Добавляем MQTT_USERNAME
	Password string // ✅ Добавляем MQTT_PASSWORD
	QoS      int    // ✅ Добавляем MQTT_QOS
}

// LoadConfig загружает конфигурацию из .env файла
func LoadConfig() *Config {

	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "ctg_user"),
			Password: getEnv("DB_PASSWORD", "ctg_password"),
			DBName:   getEnv("DB_NAME", "ctg_monitor"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			TimeZone: getEnv("DB_TIMEZONE", "Europe/Moscow"),
		},
		App: AppConfig{
			Port:     getEnv("HTTP_PORT", "8080"), // ✅ Используем HTTP_PORT из .env
			GRPCPort: getEnv("GRPC_PORT", "50051"),
			LogLevel: getEnv("LOG_LEVEL", "info"),
		},
		MQTT: MQTTConfig{
			Broker:   getEnv("MQTT_BROKER", "tcp://localhost:1883"),
			ClientID: getEnv("MQTT_CLIENT_ID", "ctg_monitor_service"),
			Username: getEnv("MQTT_USERNAME", ""), // ✅ Добавляем MQTT auth
			Password: getEnv("MQTT_PASSWORD", ""), // ✅ Добавляем MQTT auth
			QoS:      getEnvAsInt("MQTT_QOS", 1),  // ✅ Добавляем QoS
		},
	}
}

// getEnv получает переменную окружения или возвращает значение по умолчанию
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt получает переменную окружения как int
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
