package config

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func InitLogger() {
	var handler slog.Handler

	if os.Getenv("ENV") == "production" {
		// Продакшен: JSON формат
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		// Разработка: текстовый формат с исходными файлами
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		})
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	slog.Info("Logger initialized successfully")
}
