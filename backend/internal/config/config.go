package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort int
	AppName    string
	Environment string
	Database   DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	Name     string
	Port     string
}

func Load() (*Config, error) {
	// Load .env file - try multiple locations
	var err error
	// Try current directory first
	err = godotenv.Load()
	if err != nil {
		// Try backend directory (when running from project root)
		err = godotenv.Load("backend/.env")
		if err != nil {
			// Try relative to this file
			err = godotenv.Load("../.env")
			if err != nil {
				log.Println("No .env file found, using system environment variables")
			}
		}
	}

	cfg := &Config{
		ServerPort: getEnvInt("PORT", 8080),
		AppName:    getEnv("APP_NAME", "syncflow-backend"),
		Environment: getEnv("ENVIRONMENT", "development"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			Name:     getEnv("DB_NAME", "syncflow"),
			Port:     getEnv("DB_PORT", "5432"),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func setDefaultEnv(key, defaultValue string) {
	if os.Getenv(key) == "" {
		os.Setenv(key, defaultValue)
	}
}

