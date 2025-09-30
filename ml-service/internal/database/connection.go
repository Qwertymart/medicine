package database

import (
    "fmt"
    "ml-service/config"
    
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

// Connect подключается к базе данных
func Connect(cfg *config.Config) (*gorm.DB, error) {
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
        cfg.Database.Host,
        cfg.Database.User, 
        cfg.Database.Password,
        cfg.Database.DBName,
        cfg.Database.Port,
    )
    
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    // Настроить connection pool
    sqlDB, err := db.DB()
    if err != nil {
        return nil, err
    }
    
    sqlDB.SetMaxIdleConns(10)
    sqlDB.SetMaxOpenConns(100)
    
    return db, nil
}