package database

import (
	"fmt"
	"log"
	"time"

	"CTG_monitor/configs"
	_ "CTG_monitor/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

// InitDatabase инициализирует подключение к базе данных
func InitDatabase(config *configs.Config) (*gorm.DB, error) {
	// Формируем DSN строку подключения
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		config.Database.Host,
		config.Database.User,
		config.Database.Password,
		config.Database.DBName,
		config.Database.Port,
		config.Database.SSLMode,
		config.Database.TimeZone,
	)

	// Настраиваем GORM конфигурацию
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "ctg_", // Префикс для таблиц
			SingularTable: false,  // Использовать множественное число для таблиц
		},
		NowFunc: func() time.Time {
			return time.Now().UTC() // Использовать UTC для временных меток
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)                  // Максимум незанятых соединений
	sqlDB.SetMaxOpenConns(50)                  // Максимум открытых соединений
	sqlDB.SetConnMaxLifetime(time.Hour)        // Максимальное время жизни соединения
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // Максимальное время простоя

	// Проверяем соединение
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось проверить соединение с БД: %w", err)
	}

	DB = db
	log.Println("✅ Успешно подключились к PostgreSQL")

	return db, nil
}

// GetDB возвращает экземпляр базы данных
func GetDB() *gorm.DB {
	return DB
}

// CloseDatabase корректно закрывает соединение с БД
func CloseDatabase() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	log.Println("🔒 Закрываем соединение с базой данных")
	return sqlDB.Close()
}

// HealthCheck проверяет состояние базы данных
func HealthCheck() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("не удалось получить sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("база данных недоступна: %w", err)
	}

	return nil
}
