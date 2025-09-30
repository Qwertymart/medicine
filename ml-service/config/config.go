package config

import (
	"os"
)

type Config struct {
	Port     string
	Database DatabaseConfig
	ML       MLConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type MLConfig struct {
	ServiceURL string
	Timeout    int
}

func Load() *Config {
	return &Config{
		Port: "8052",
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "ctg_db"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		ML: MLConfig{
			ServiceURL: getEnv("ML_SERVICE_URL", "http://localhost:8000"),
			Timeout:    30,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
