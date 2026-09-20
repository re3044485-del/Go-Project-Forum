package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
}

func Load() (*Config, error) {
	// Игнорируем ошибку, если .env файл отсутствует (в продакшене на Render)
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-dev-secret-change-in-production"
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}, nil
}
