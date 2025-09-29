// internal/database/migrations.go
package database

import (
	"CTG_monitor/internal/models"
	"fmt"
	"gorm.io/gorm"
	"log"
)

// RunMigrations выполняет миграции базы данных
func RunMigrations(db *gorm.DB) error {
	log.Println("🔄 Запуск миграций базы данных...")

	// Автоматические миграции GORM
	err := db.AutoMigrate(
		&models.CTGSession{},
		// Добавьте здесь другие модели при необходимости
	)

	if err != nil {
		return fmt.Errorf("ошибка миграции: %w", err)
	}

	// Создаем индексы для оптимизации запросов
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("ошибка создания индексов: %w", err)
	}

	log.Println("✅ Миграции выполнены успешно")
	return nil
}

// createIndexes создает дополнительные индексы
func createIndexes(db *gorm.DB) error {
	// Создаем составные индексы для быстрого поиска
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_ctg_sessions_device_active ON ctg_sessions(device_id, end_time) WHERE end_time IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_ctg_sessions_start_time_desc ON ctg_sessions(start_time DESC)",
		"CREATE INDEX IF NOT EXISTS idx_ctg_sessions_card_device ON ctg_sessions(card_id, device_id)",

		// GIN индексы для JSONB полей (для быстрых запросов по содержимому)
		"CREATE INDEX IF NOT EXISTS idx_ctg_sessions_fhr_gin ON ctg_sessions USING GIN (fhr_data)",
		"CREATE INDEX IF NOT EXISTS idx_ctg_sessions_uc_gin ON ctg_sessions USING GIN (uc_data)",

		// Частичные индексы только для активных сессий
		"CREATE INDEX IF NOT EXISTS idx_active_sessions ON ctg_sessions(device_id, start_time) WHERE end_time IS NULL",
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("⚠️ Не удалось создать индекс: %s, ошибка: %v", indexSQL, err)
		} else {
			log.Printf("✅ Индекс создан: %s", indexSQL[:50]+"...")
		}
	}

	return nil
}
