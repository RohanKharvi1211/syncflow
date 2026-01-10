package config

import (
	"fmt"
	"log"

	"syncflow-backend/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDatabase() {
	var err error

	// Database connection string - using environment variables directly
	// Note: This function should ideally receive Config struct, but kept for backward compatibility
	// For RDS, use sslmode=require. For local dev, use sslmode=disable
	dbHost := getEnv("DB_HOST", "localhost")
	sslMode := "disable" // Default for local development
	if dbHost != "localhost" && dbHost != "127.0.0.1" {
		sslMode = "require" // Use SSL for RDS and other remote databases
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbHost,
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "syncflow"),
		getEnv("DB_PORT", "5432"),
		sslMode,
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Enable UUID extension
	DB.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	// Auto migrate the schema
	err = DB.AutoMigrate(
		// New schema models
		&models.Company{},
		&models.User{},
		&models.App{},
		&models.Connection{},
		&models.Metadata{},
		&models.Integration{},
		&models.SyncJob{},
		&models.SyncRecord{},
		// Legacy models for backward compatibility
		&models.FieldMapping{},
		&models.SyncLog{},
		&models.TallyConfig{},
		&models.LegacyIntegration{},
		&models.LegacySyncRecord{},
	)

	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database connected and migrated successfully")
}
