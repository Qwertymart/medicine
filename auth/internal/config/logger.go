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

		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: replaceTimeAttr,
			AddSource:   true,
		})
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)

	slog.Info("Logger initialized successfully")
}

func replaceTimeAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey {
		return slog.String("time", a.Value.Time().Local().Format("2006-01-02 15:04:05"))
	}
	return a
}
