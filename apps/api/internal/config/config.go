package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	JWTExpiresIn time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Port:         getEnv("APP_PORT", "8080"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		JWTExpiresIn: 24 * time.Hour,
	}

	if value := os.Getenv("JWT_EXPIRES_HOURS"); value != "" {
		d, err := time.ParseDuration(value + "h")
		if err != nil {
			return Config{}, err
		}
		cfg.JWTExpiresIn = d
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
