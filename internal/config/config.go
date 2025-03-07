package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Database DatabaseConfig
	App      AppConfig
	Discord  DiscordConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type AppConfig struct {
	ScanInterval string
	Timezone     string
}

type DiscordConfig struct {
	BotToken string
}

func LoadConfig() (*Config, error) {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	} else {
		log.Println("Successfully loaded .env file")
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvAsIntOrDefault("DB_PORT", 5432),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: os.Getenv("DB_PASSWORD"),
			DBName:   getEnvOrDefault("DB_NAME", "memodb"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		},
		App: AppConfig{
			ScanInterval: getEnvOrDefault("SCAN_INTERVAL", "60s"),
			Timezone:     getEnvOrDefault("TIMEZONE", "UTC"),
		},
		Discord: DiscordConfig{
			BotToken: os.Getenv("DISCORD_BOT_TOKEN"),
		},
	}

	// Debug: Print all environment variables
	log.Printf("Environment variables:")
	log.Printf("DB_HOST: %s", config.Database.Host)
	log.Printf("DB_PORT: %d", config.Database.Port)
	log.Printf("DB_USER: %s", config.Database.User)
	log.Printf("DB_NAME: %s", config.Database.DBName)
	log.Printf("DB_SSLMODE: %s", config.Database.SSLMode)
	log.Printf("SCAN_INTERVAL: %s", config.App.ScanInterval)
	log.Printf("TIMEZONE: %s", config.App.Timezone)
	log.Printf("DISCORD_BOT_TOKEN length: %d", len(config.Discord.BotToken))

	// Validate required fields
	if config.Discord.BotToken == "" {
		return nil, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}

	if config.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}

	// Parse scan interval duration
	scanInterval, err := time.ParseDuration(config.App.ScanInterval)
	if err != nil {
		return nil, fmt.Errorf("invalid scan interval format: %w", err)
	}
	config.App.ScanInterval = scanInterval.String()

	// Debug logging (without exposing sensitive data)
	log.Printf("Config loaded successfully")

	return config, nil
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
