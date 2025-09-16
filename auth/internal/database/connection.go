// database/database.go
package database

import (
	"auth/internal/config"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg config.DatabaseConfig) *gorm.DB {
	slog.Info("Connecting to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"database", cfg.DBName,
	)

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.Port,
		cfg.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		slog.Error("Failed to get database instance", "error", err)
		panic(fmt.Sprintf("Failed to get database instance: %v", err))
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	slog.Info("Database connection successful")
	return db
}

func Migrate(db *gorm.DB, models ...interface{}) {
	slog.Info("Starting database migration")

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			slog.Error("Failed to migrate model", "error", err, "model", fmt.Sprintf("%T", model))
			panic(fmt.Sprintf("Failed to migrate model: %v", err))
		}
	}

	slog.Info("Database migration completed successfully", "models_count", len(models))
}
