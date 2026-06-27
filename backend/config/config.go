package config

import "os"

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	UploadDir        string
	Port             string
	WooBaseURL       string
	WooAPIKey        string
	WooAPISecret     string
	WooCategorySlug  string
	TelegramBotToken string
	TelegramChatID   string
}

func Load() *Config {
	return &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable"),
		JWTSecret:        getEnv("JWT_SECRET", "dev-secret-change-me"),
		UploadDir:        getEnv("UPLOAD_DIR", "uploads"),
		Port:             getEnv("PORT", "8080"),
		WooBaseURL:       getEnv("WOO_BASE_URL", "https://flowers.fenny-studio.com"),
		WooAPIKey:        getEnv("WOO_API_KEY", ""),
		WooAPISecret:     getEnv("WOO_API_SECRET", ""),
		WooCategorySlug:  getEnv("WOO_CATEGORY_SLUG", "my-diy"),
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
