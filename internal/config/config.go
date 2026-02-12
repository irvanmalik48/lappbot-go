package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken   string
	BotName    string
	BotVersion string
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	BotOwnerID int64

	ValkeyHost     string
	ValkeyPort     int
	ValkeyPassword string
	BotAPIURL      string

	TelegramAPIID   int
	TelegramAPIHash string
	ReportChannelID int64
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	return &Config{
		BotToken:   getEnv("BOT_TOKEN", ""),
		BotName:    getEnv("BOT_NAME", "Lappland"),
		BotVersion: getEnv("BOT_VERSION", "1.0.0"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvAsInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_DATABASE", "lappbot"),
		BotOwnerID: getEnvAsInt64("BOT_OWNER_ID", 0),

		ValkeyHost:     getEnv("VALKEY_HOST", "localhost"),
		ValkeyPort:     getEnvAsInt("VALKEY_PORT", 6379),
		ValkeyPassword: getEnv("VALKEY_PASSWORD", ""),
		BotAPIURL:      getEnv("BOT_API_URL", "http://localhost:8081"),

		TelegramAPIID:   getEnvAsInt("TELEGRAM_API_ID", 0),
		TelegramAPIHash: getEnv("TELEGRAM_API_HASH", ""),
		ReportChannelID: getEnvAsInt64("REPORT_CHANNEL_ID", 0),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}

func getEnvAsInt64(key string, fallback int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return fallback
}
