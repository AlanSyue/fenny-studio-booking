package config

import "os"

type Config struct {
	DatabaseURL string
	JWTSecret   string
	UploadDir   string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://fenny:fenny@localhost:5432/fenny_booking?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "dev-secret-change-me"),
		UploadDir:   getEnv("UPLOAD_DIR", "uploads"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
